package mailbox

import (
	"log"
	"net/mail"
	"testing"
	"time"
)

// Messages that will be enqueued for the test
// set up and dequeued for the actual test.
type testSetup struct {
	MID  uint64
	Rcpt string
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
		/*
			doe.com    -> 1 -> jane
			doe.com    -> 2 -> jim
			doe.com    -> 4 -> jane
			bree.com   -> 1 -> ann
			bree.com   -> 3 -> adam
			bree.com   -> 3 -> ann
			cheese.com -> 4 -> ann
		*/

		msgSetup: []testSetup{
			{1, "<jane@doe.com>, <ann@bree.com>, <adam@doe.com>"},
			{2, "<jim@doe.com>"},
			{3, "<adam@bree.com>, <ann@bree.com>"},
			{4, "<jane@doe.com>, <ann@cheese.com>"},
		},

		want: []testWant{
			{
				N: 1,
				Items: map[string]map[int][]*mail.Address{
					"doe.com": map[int][]*mail.Address{
						1: []*mail.Address{{Address: "jane@doe.com"}, {Address: "adam@doe.com"}},
						2: []*mail.Address{{Address: "jim@doe.com"}},
						4: []*mail.Address{{Address: "jane@doe.com"}},
					},
				},
			},
		},
	},
}

func TestDequeuer_Dequeue(t *testing.T) {
	EnsureTestDB()

	pb, err := New(dbString)
	if err != nil {
		t.Error("Error starting server")
	}
	defer pb.Close()

	for _, ts := range dequeuerTests {
		enqueueMsgs(pb, ts.msgSetup)
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

func enqueueMsgs(mb *mailBox, msgs []testSetup) {
	chk := func(err error) {
		if err != nil {
			log.Fatal(err)
		}
	}
	CleanDB(mb.db)
	for _, msg := range msgs {
		list, err := mail.ParseAddressList(msg.Rcpt)
		chk(err)
		err = mb.Enqueue(&Message{
			ID:      msg.MID,
			Raw:     "BODY",
			from:    &mail.Address{Address: "from@addre.ss"},
			rcptOut: list,
		})
		chk(err)
		time.Sleep(time.Millisecond) // allows for different timestamps in DB
	}
}
