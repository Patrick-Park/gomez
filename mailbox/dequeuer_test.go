package mailbox

import (
	"log"
	"net/mail"
	"testing"
)

// Messages that will be enqueued for the test
// set up and dequeued for the actual test.
type testSetup struct {
	MID    uint64
	Rcpt   string
	Addded string
}

// The actual test. N is the number that will be passed
// to the Dequeuer and Items is what is expected from the
// Dequeuer.
type testWant struct {
	N int
	// host -> msg ID -> address list
	Items  map[string]map[int][]*mail.Address
	HasErr bool
}

var dequeuerTests = []struct {
	msgSetup []testSetup
	want     []testWant
}{
	{
		msgSetup: []testSetup{
			{1, "<jane@doe.com>", "12:05"},
			{1, "<adam@doe.com>", "12:05"},
			{1, "<ann@bree.com>", "12:06"},
			{2, "<jim@doe.com>", "12:07"},
			{3, "<adam@bree.com>", "12:08"},
			{3, "<ann@bree.com>", "12:08"},
			{4, "<jane@doe.com>", "12:09"},
			{4, "<ann@cheese.com>", "12:09"},
		},

		want: []testWant{
			{
				N: 1,
				Items: map[string]map[int][]*mail.Address{
					"doe.com": map[int][]*mail.Address{
						1: alist("jane@doe.com", "adam@doe.com"),
						2: alist("jim@doe.com"),
						4: alist("jane@doe.com"),
					},
				},
			},
		},
	},
}

// Creates a list of *mail.Addresses
func alist(addrs ...string) []*mail.Address {
	list := make([]*mail.Address, len(addrs))
	for _, addr := range addrs {
		list = append(list, &mail.Address{Address: addr})
	}
	return list
}

func TestDequeuer_Dequeue(t *testing.T) {
	EnsureTestDB()

	pb, err := New(dbString)
	if err != nil {
		t.Error("Error starting server")
	}
	defer pb.Close()

	for _, ts := range dequeuerTests {
		CleanDB(pb.db)
		setupDequeuerTest(pb, ts.msgSetup)

		for _, tt := range ts.want {
			jobs, err := pb.Dequeue(tt.N)
			if tt.HasErr {
				if err == nil {
					t.Errorf("Expected error")
				}
				continue
			}
			if err != nil {
				t.Errorf("Unexpected error.")
			}
			log.Printf("%+v", jobs)
		}
	}
}

func setupDequeuerTest(mb *mailBox, msgs []testSetup) {
	chk := func(err error) {
		if err != nil {
			log.Fatal(err)
		}
	}
	stmt, err := mb.db.Prepare("INSERT INTO queue VALUES ($1, $2, $3, $4, 0)")
	chk(err)
	stmt_msg, err := mb.db.Prepare("INSERT INTO messages VALUES ($1, $2, $3, $4)")
	chk(err)
	var lastId uint64
	for _, msg := range msgs {
		if msg.MID != lastId {
			_, err = stmt_msg.Exec(msg.MID, "from@addre.ss", "rcpt@addre.ss", "BODY")
			chk(err)
			lastId = msg.MID
		}
		addr, err := mail.ParseAddress(msg.Rcpt)
		chk(err)
		u, h := SplitUserHost(addr)
		_, err = stmt.Exec(h, msg.MID, u, "1984-10-26 "+msg.Addded)
		chk(err)
	}
}
