package mailbox

import (
	"database/sql"
	"log"
	"net/mail"
	"os"
	"os/exec"
	"reflect"
	"sync"
	"testing"
)

const (
	dbUser         = "Gabriel"
	dbString       = "user=" + dbUser + " dbname=gomez_test sslmode=disable"
	testSchemaFile = "schema/schema_test.sql"
)

var once sync.Once

// Ensures the test database is set up. Should be called before each test
func EnsureTestDB() {
	once.Do(setUpTestDB)
}

// Wipes data from DB
func CleanDB(db *sql.DB) {
	_, err := db.Exec(`
		DELETE FROM mailbox;
		DELETE FROM messages;
		DELETE FROM queue;
		DELETE FROM users`)

	if err != nil {
		log.Fatalf("Error tearing down: %s", err)
	}
}

// Sets up the test database from the schema file.
func setUpTestDB() {
	file, err := os.Open(testSchemaFile)
	if err != nil {
		log.Fatalf("Error opening schema file: %s", err)
	}
	defer file.Close()

	cmd := exec.Command("psql", "--username="+dbUser, "-q")
	cmd.Stdin = file

	err = cmd.Run()
	if err != nil {
		log.Fatalf("Error setting up DB: %s", err)
	}
}

func TestPostBox_NextID_Error(t *testing.T) {
	pb, err := NewPostBox("bogus")
	if err != nil {
		t.Errorf("Could not open DB:", err)
	}
	defer pb.Close()

	_, err = pb.NextID()
	if err == nil {
		t.Error("Was expecting an error.")
	}
}

func TestPostBox_NextID_Success(t *testing.T) {
	EnsureTestDB()

	pb, err := NewPostBox(dbString)
	if err != nil {
		t.Errorf("Could not open DB:", err)
	}
	defer pb.Close()

	id, err := pb.NextID()
	if err != nil {
		t.Errorf("Failed to extract sequence val: %s", err)
	}

	t.Log(id)
}

type queueRow struct {
	MID      uint64
	Rcpt     string
	Attempts int
}

type mailboxRow struct {
	MID uint64
	UID uint64
}

func TestPostBox_Enqueuer(t *testing.T) {
	EnsureTestDB()

	pb, err := NewPostBox(dbString)
	if err != nil {
		t.Errorf("Failed to extract sequence val: %s", err)
	}

	for k, test := range []struct {
		Users        string
		Msg          *Message
		QueueItem    queueRow
		MailboxItems []mailboxRow
		HasErr       bool // Expect error
	}{
		{
			// Mixed
			"(5, 'a b', 'a@b.com'),	(7, 'c d', 'c@d.com')",
			&Message{
				ID:      123,
				Raw:     "MessageBody",
				from:    &mail.Address{"Dummy Guy", "dummy@guy.com"},
				rcptIn:  []*mail.Address{&mail.Address{"a b", "a@b.com"}, &mail.Address{"c d", "c@d.com"}},
				rcptOut: []*mail.Address{&mail.Address{"x z", "x@z.com"}, &mail.Address{"q w", "q@w.eu"}},
			},
			queueRow{123, `"x z" <x@z.com>, "q w" <q@w.eu>`, 0},
			[]mailboxRow{{123, 5}, {123, 7}}, false,
		}, {
			// Only local
			"(5, 'a b', 'a@b.com'),	(7, 'c d', 'c@d.com')",
			&Message{
				ID:      123,
				Raw:     "MessageBody",
				from:    &mail.Address{"Dummy Guy", "dummy@guy.com"},
				rcptIn:  []*mail.Address{&mail.Address{"a b", "a@b.com"}, &mail.Address{"c d", "c@d.com"}},
				rcptOut: []*mail.Address{},
			},
			queueRow{}, []mailboxRow{{123, 5}, {123, 7}}, false,
		}, {
			// Only remote
			"(5, 'a b', 'a@b.com'),	(7, 'c d', 'c@d.com')",
			&Message{
				ID:      123,
				Raw:     "MessageBody",
				from:    &mail.Address{"Dummy Guy", "dummy@guy.com"},
				rcptIn:  []*mail.Address{},
				rcptOut: []*mail.Address{&mail.Address{"x z", "x@z.com"}, &mail.Address{"q w", "q@w.eu"}},
			},
			queueRow{123, `"x z" <x@z.com>, "q w" <q@w.eu>`, 0}, []mailboxRow{}, false,
		}, {
			// Error (duplicate inbound)
			"(5, 'a b', 'a@b.com'),	(7, 'c d', 'c@d.com')",
			&Message{
				ID:      123,
				Raw:     "MessageBody",
				from:    &mail.Address{"Dummy Guy", "dummy@guy.com"},
				rcptIn:  []*mail.Address{&mail.Address{"a b", "a@b.com"}, &mail.Address{"a b", "a@b.com"}},
				rcptOut: []*mail.Address{&mail.Address{"x z", "x@z.com"}, &mail.Address{"q w", "q@w.eu"}},
			},
			queueRow{}, []mailboxRow{}, true,
		},
	} {
		// Teardown
		CleanDB(pb.db)

		// Setup
		_, err = pb.db.Exec(`INSERT INTO users (id, name, address) VALUES ` + test.Users)
		if err != nil {
			t.Errorf("Error setting up test: %s", err)
		}

		err = pb.Enqueue(test.Msg)
		if !test.HasErr && err != nil {
			t.Errorf("Error enqueuing message: %s", err)
		}

		if test.HasErr {
			if err == nil {
				t.Errorf("Expected error on test #%d.", k)
			}

			continue
		}

		// Test that outbound messages were queued
		var q queueRow

		r := pb.db.QueryRow("SELECT message_id, rcpt, attempts FROM queue")
		err = r.Scan(&q.MID, &q.Rcpt, &q.Attempts)
		if err != nil && err != sql.ErrNoRows {
			t.Errorf("Failed to query queue: %s", err)
		}

		if !reflect.DeepEqual(q, test.QueueItem) {
			t.Errorf("Expected %+v, got %+v", test.QueueItem, q)
		}

		// Test that inbound messages were delivered
		m := make([]mailboxRow, 0, 10)
		rows, err := pb.db.Query("SELECT user_id, message_id FROM mailbox")
		if err != nil {
			t.Errorf("Failed to query queue: %s", err)
		}

		for rows.Next() {
			var mr mailboxRow
			rows.Scan(&mr.UID, &mr.MID)
			m = append(m, mr)
		}

		if !reflect.DeepEqual(m, test.MailboxItems) {
			t.Errorf("Expected %+v, got %+v", test.MailboxItems, m)
		}

		rows.Close()
	}

	pb.Close()
}
