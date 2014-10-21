package mailbox

import (
	"database/sql"
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

type queueRow struct {
	ID       uint64
	Rcpt     string
	Date     pq.NullTime
	Attempts int
}

type messageRow struct {
	MID uint64
	UID uint64
}

func TestPostBox_Enqueuer(t *testing.T) {
	EnsureTestDB()

	pb, err := NewPostBox(dbString)
	if err != nil {
		t.Errorf("Failed to extract sequence val: %s", err)
	}
	defer pb.Close()

	// Setup
	_, err = pb.db.Exec(`INSERT INTO users (id, name, address) VALUES 
		(5, 'a b', 'a@b.com'),
		(7, 'c d', 'c@d.com')`)

	if err != nil {
		t.Errorf("Error setting up test: %s", err)
	}

	msg := &Message{
		ID:      123,
		Raw:     "MessageBody",
		from:    &mail.Address{"Dummy Guy", "dummy@guy.com"},
		rcptIn:  []*mail.Address{&mail.Address{"a b", "a@b.com"}, &mail.Address{"c d", "c@d.com"}},
		rcptOut: []*mail.Address{&mail.Address{"x z", "x@z.com"}, &mail.Address{"q w", "q@w.eu"}},
	}

	// Test
	err = pb.Enqueue(msg)
	if err != nil {
		t.Errorf("Error enqueuing message: %s", err)
	}

	r, err := pb.db.Query("SELECT message_id, rcpt, date_added, attempts FROM queue")
	if err != nil {
		t.Errorf("Failed to query queue: %s", err)
	}

	q := make([]queueRow, 0, 10)

	for r.Next() {
		var qr queueRow

		r.Scan(&qr.ID, &qr.Rcpt, qr.Date, &qr.Attempts)
		q = append(q, qr)
	}

	// fmt.Println(q)

	r.Close()

	r, err = pb.db.Query("SELECT user_id, message_id FROM mailbox")
	if err != nil {
		t.Errorf("Failed to query queue: %s", err)
	}

	m := make([]messageRow, 0, 10)

	for r.Next() {
		var mr messageRow
		r.Scan(&mr.UID, &mr.MID)
		m = append(m, mr)
	}

	// fmt.Println(m)

	r.Close()

	// Teardown
	CleanDB(pb.db)
}
