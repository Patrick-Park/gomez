package mailbox

import (
	"log"
	"net/mail"
	"reflect"
	"sort"
	"testing"
)

// Maps message to destination by ID.
// MessageID -> Destination addresses
type DeliveryByID map[uint64][]*mail.Address

// Messages that will be enqueued for the test
// set up and dequeued for the actual test.
type queueItem struct {
	MID    uint64
	Rcpt   string
	Addded string
}

// Creates a list of *mail.Addresses from a list of strings.
func addrList(addrs ...string) []*mail.Address {
	list := make([]*mail.Address, 0, len(addrs))
	for _, addr := range addrs {
		list = append(list, &mail.Address{Address: addr})
	}
	return list
}

func TestDequeuer_Dequeue(t *testing.T) {
	EnsureTestDB()
	// The actual test. N is the number that will be passed
	// to the Dequeuer and Items is what is expected from the
	// Dequeuer.
	type testCase struct {
		N      int
		Items  map[string]DeliveryByID
		HasErr bool
	}
	log.Println(dbString)
	pb, err := New(dbString)
	if err != nil {
		t.Error("Error starting server")
	}
	defer pb.Close()
	for _, ts := range []struct {
		msgSetup []queueItem
		want     []testCase
	}{
		{
			// This is the setup for the tests that follow.
			msgSetup: []queueItem{
				{1, "<jane@doe.com>", "12:05"},
				{1, "<adam@doe.com>", "12:05"},
				{1, "<ann@bree.com>", "12:06"},
				{2, "<jim@doe.com>", "12:07"},
				{3, "<adam@bree.com>", "12:08"},
				{3, "<ann@bree.com>", "12:08"},
				{4, "<jane@doe.com>", "12:09"},
				{4, "<brad@cheese.com>", "12:09"},
			},
			// This is a series of tests that act on the above setup
			want: []testCase{
				{N: 1, Items: map[string]DeliveryByID{
					"doe.com": DeliveryByID{
						1: addrList("adam@doe.com", "jane@doe.com"),
						2: addrList("jim@doe.com"),
						4: addrList("jane@doe.com"),
					},
				}},
				{N: 2, Items: map[string]DeliveryByID{
					"doe.com": DeliveryByID{
						1: addrList("adam@doe.com", "jane@doe.com"),
						2: addrList("jim@doe.com"),
						4: addrList("jane@doe.com"),
					},
					"bree.com": DeliveryByID{
						1: addrList("ann@bree.com"),
						3: addrList("adam@bree.com", "ann@bree.com"),
					},
				}},
				{N: 5, Items: map[string]DeliveryByID{
					"doe.com": DeliveryByID{
						1: addrList("adam@doe.com", "jane@doe.com"),
						2: addrList("jim@doe.com"),
						4: addrList("jane@doe.com"),
					},
					"bree.com": DeliveryByID{
						1: addrList("ann@bree.com"),
						3: addrList("adam@bree.com", "ann@bree.com"),
					},
					"cheese.com": DeliveryByID{
						4: addrList("brad@cheese.com"),
					},
				}},
			},
		}, {
			msgSetup: []queueItem{
				{1, "james@john.com", "12:00"},
				{2, "jenny@jane.com", "12:01"},
				{3, "adams@dimm.com", "12:02"},
				{4, "donny@jims.com", "12:03"},
				{1, "jimmy@john.com", "12:04"},
				{1, "eliza@dimm.com", "12:05"},
				{2, "eliza@dimm.com", "12:06"},
				{3, "jenny@jane.com", "12:07"},
			},
			want: []testCase{
				{N: 1, Items: map[string]DeliveryByID{
					"john.com": DeliveryByID{
						1: addrList("james@john.com", "jimmy@john.com"),
					},
				}},
				{N: 3, Items: map[string]DeliveryByID{
					"john.com": DeliveryByID{
						1: addrList("james@john.com", "jimmy@john.com"),
					},
					"jane.com": DeliveryByID{
						2: addrList("jenny@jane.com"),
						3: addrList("jenny@jane.com"),
					},
					"dimm.com": DeliveryByID{
						1: addrList("eliza@dimm.com"),
						2: addrList("eliza@dimm.com"),
						3: addrList("adams@dimm.com"),
					},
				}},
			},
		},
	} {
		setupDequeuerTest(pb, ts.msgSetup)
		for _, tt := range ts.want {
			MaxHostsPerDequeue = tt.N
			jobs, err := pb.Dequeue()
			if tt.HasErr {
				if err == nil {
					t.Errorf("Expected error")
				}
				continue
			}
			if err != nil {
				t.Errorf("Unexpected error.")
			}
			if got, same := compareResults(jobs, tt.Items); !same {
				t.Errorf("Got %+v, want %+v", got, tt.Items)
			}
		}
	}
}

// compareResults compares a map of host->Delivery to a map of host->DeliveryByID
// where ID matches the ID of the message from the gotten Delivery.
func compareResults(
	got map[string]Delivery,
	want map[string]DeliveryByID,
) (map[string]DeliveryByID, bool) {

	cgot := make(map[string]DeliveryByID)
	for host, delivery := range got {
		cgot[host] = make(DeliveryByID)
		for msg, addrs := range delivery {
			if cgot[host][msg.ID] == nil {
				cgot[host][msg.ID] = make([]*mail.Address, 0, len(addrs))
			}
			for _, addr := range addrs {
				cgot[host][msg.ID] = append(cgot[host][msg.ID], addr)
			}
			if _, ok := want[host][msg.ID]; !ok {
				return cgot, false
			}
			// Sort slice so that we can perform a deep equal
			// and overlook order.
			sort.Sort(byAddress(cgot[host][msg.ID]))
			sort.Sort(byAddress(want[host][msg.ID]))
		}
	}
	return cgot, reflect.DeepEqual(cgot, want)
}

type byAddress []*mail.Address

func (by byAddress) Len() int           { return len(by) }
func (by byAddress) Less(i, j int) bool { return by[i].Address < by[j].Address }
func (by byAddress) Swap(i, j int)      { by[i], by[j] = by[j], by[i] }

func Test_compareResults(t *testing.T) {
	for _, tt := range []struct {
		got    map[string]Delivery
		want   map[string]DeliveryByID
		expect bool
	}{
		{
			got: map[string]Delivery{"addr.net": Delivery{
				&Message{ID: 1}: addrList("jane@addr.net", "john@addr.net"),
				&Message{ID: 2}: addrList("a@b.com", "c@d.com", "x@y.com"),
			}},
			want: map[string]DeliveryByID{"addr.net": DeliveryByID{
				1: addrList("john@addr.net", "jane@addr.net"),
				2: addrList("x@y.com", "c@d.com", "a@b.com"),
			}},
			expect: true,
		},
		{
			got: map[string]Delivery{"addr.net": Delivery{
				&Message{ID: 1}: addrList("jane@addr.net", "john@addr.net"),
				&Message{ID: 2}: addrList("a@b.com", "c@d.com", "x@y.com"),
			}},
			want: map[string]DeliveryByID{"addr.net": DeliveryByID{
				1: addrList("jane@addr.net", "john@addr.net"),
				2: addrList("x@y.com", "SOMETHING DIFFERENT", "a@b.com"),
			}},
			expect: false,
		},
		{
			got: map[string]Delivery{"addr.net": Delivery{
				&Message{ID: 1}: addrList("jane@addr.net", "john@addr.net"),
				&Message{ID: 2}: addrList("a@b.com", "c@d.com", "x@y.com"),
			}},
			want: map[string]DeliveryByID{"addr.net": DeliveryByID{
				1: addrList("jane@addr.net", "john@addr.net"),
				2: addrList("x@y.com", "c@d.com", "a@b.com"),
				3: addrList("x@y.com", "c@d.com", "a@b.com"),
			}, "xcv.net": DeliveryByID{
				1: addrList("jane@addr.net", "john@addr.net"),
				2: addrList("x@y.com", "c@d.com", "a@b.com"),
			}},
			expect: false,
		},
	} {
		if _, exp := compareResults(tt.got, tt.want); exp != tt.expect {
			t.Error("Incorect comparison")
		}
	}
}

func setupDequeuerTest(mb *mailBox, msgs []queueItem) {
	chk := func(err error) {
		if err != nil {
			log.Fatal(err)
		}
	}
	CleanDB(mb.db)
	stmt, err := mb.db.Prepare("INSERT INTO queue VALUES ($1, $2, $3, $4, 0)")
	chk(err)
	stmt_msg, err := mb.db.Prepare("INSERT INTO messages VALUES ($1, $2, $3, $4)")
	chk(err)
	var msgIDs []uint64
	for _, msg := range msgs {
		for _, id := range msgIDs {
			if id == msg.MID {
				goto addQueue
			}
		}
		_, err = stmt_msg.Exec(msg.MID, "from@addre.ss", "rcpt@addre.ss", "BODY")
		chk(err)
		msgIDs = append(msgIDs, msg.MID)
	addQueue:
		addr, err := mail.ParseAddress(msg.Rcpt)
		chk(err)
		u, h := SplitUserHost(addr)
		_, err = stmt.Exec(h, msg.MID, u, "1984-10-26 "+msg.Addded)
		chk(err)
	}
}
