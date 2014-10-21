package mailbox

import (
	"fmt"
	"log"
	"net/mail"
	"os"
	"os/exec"
	"sync"
	"testing"

	"github.com/lib/pq"
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

func TestPostBox_DB_Connection(t *testing.T) {
	EnsureTestDB()

	pb, err := NewPostBox(dbString)
	if err != nil {
		t.Errorf("Could not open DB:", err)
	}
	defer pb.Close()

	_, err = pb.db.Query("SELECT * FROM messages LIMIT 1")
	if err != nil {
		t.Errorf("Cannot query: %s", err)
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

func TestPostBox_Enqueuer(t *testing.T) {
	EnsureTestDB()

	pb, err := NewPostBox(dbString)
	if err != nil {
		t.Errorf("Failed to extract sequence val: %s", err)
	}
	defer pb.Close()

	// Setup
	tx, err := pb.db.Begin()
	if err != nil {
		t.Errorf("Error starting transaction: %s", err)
	}

	_, err = tx.Exec(`INSERT INTO users (name, address) VALUES 
		('a b', 'a@b.com'),
		('c d', 'c@d.com')`)
	if err != nil {
		t.Errorf("Error setting up test: %s", err)
	}

	msg := &Message{
		ID:      123,
		Raw:     "MessageBody",
		from:    &mail.Address{"Dummy Guy", "dummy@guy.com"},
		rcptIn:  []*mail.Address{&mail.Address{"a b", "a@b.com"}, &mail.Address{"c d", "c@d.com"}},
		rcptOut: []*mail.Address{&mail.Address{"x z", "x@z.com"}},
	}

	// Test
	err = pb.Enqueue(msg)
	if err != nil {
		t.Errorf("Error enqueuing message: %s", err)
	}

	r, err := tx.Query("SELECT message_id, rcpt, date_added, attempts FROM queue")
	if err != nil {
		t.Errorf("Failed to query queue: %s", err)
	}

	for r.Next() {
		id, rcpt, date, att := uint64(0), "", new(pq.NullTime), 0
		r.Scan(&id, &rcpt, date, &att)
		fmt.Println(id, rcpt, date, att)
	}

	r.Close()

	// Teardown
	tx.Rollback()
}
