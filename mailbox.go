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
type postBox struct {
	db *sql.DB
}

// Creates a new PostBox based on the given connection string
func NewPostBox(dbString string) (*postBox, error) {
	pb := new(postBox)

	db, err := sql.Open("postgres", dbString)
	if err != nil {
		return nil, err
	}

	pb.db = db
	return pb, nil
}

// Closes the database connection
func (p *postBox) Close() error {
	return p.db.Close()
}
