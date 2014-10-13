package gomez

import (
	"database/sql"
	"net/mail"

	_ "github.com/lib/pq"
)

// Mailbox implements message queueing and
// dequeueing system
type Mailbox interface {
	// Obtains a new message identification number in 64 bits.
	NextID() (uint64, error)

	// Places a message onto the queue for delivery at next pick-up.
	Queue(msg *Message) error

	// Queries the server for a user
	Query(addr *mail.Address) QueryStatus
}

// QueryStatus reflects the outcome of the result of querying the mailbox
type QueryStatus int

const (
	// This state indicates that the user was local (judging by host),
	// but was not found
	QUERY_STATUS_NOT_FOUND QueryStatus = iota

	// This status reflects that the user is not local
	QUERY_STATUS_NOT_LOCAL

	// Query was successful
	QUERY_STATUS_SUCCESS

	// An error has occurred
	QUERY_STATUS_ERROR
)

// PostgreSQL implementation of the mailbox
type postBox struct{ db *sql.DB }

// Creates a new PostBox based on the given connection string
// Example connection strings can be seen at: http://godoc.org/github.com/lib/pq
func NewPostBox(dbString string) (*postBox, error) {
	db, err := sql.Open("postgres", dbString)
	if err != nil {
		return nil, err
	}

	return &postBox{db}, nil
}

// Extracts a unique ID from a database sequence
func (p *postBox) NextID() (uint64, error) {
	var id uint64

	row := p.db.QueryRow("SELECT nextval('message_ids')")
	err := row.Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

// Queues and saves a message
func (p *postBox) Queue(msg *Message) error {
	// _, err := p.db.Exec("INSERT INTO messages(id, from, rcpt, raw) VALUES (?, ?, ?, ?)", ...)
	return nil
}

// Searches for an address in the mailbox
func (p *postBox) Query(addr *mail.Address) QueryStatus {
	return QUERY_STATUS_SUCCESS
}

// Closes the database connection
func (p *postBox) Close() error {
	return p.db.Close()
}
