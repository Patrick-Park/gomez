package gomez

import (
	"database/sql"
	"net/mail"

	_ "github.com/lib/pq"
)

// Enqueuer is used to enqueue messages for delivery and to query for addresses.
type Enqueuer interface {
	// Places a message onto the queue for delivery at next pick-up.
	Enqueue(msg *Message) error

	// Obtains a new message identification number in 64 bits.
	NextID() (uint64, error)

	// Queries the server for an address.
	Query(addr *mail.Address) QueryStatus
}

// QueryStatus reflects the outcome of the result of querying the mailbox
type QueryStatus int

const (
	// This state indicates that a user is local, but not found.
	QUERY_STATUS_NOT_FOUND QueryStatus = iota

	// Query was successful, and user was found locally.
	QUERY_STATUS_SUCCESS

	// This status reflects that the user is not local.
	QUERY_STATUS_NOT_LOCAL

	// An error has occurred
	QUERY_STATUS_ERROR
)

// PostgreSQL implementation of the mailbox
type postBox struct{ db *sql.DB }

// Creates a new PostBox based on the given connection string. Example
// connection strings can be seen at: http://godoc.org/github.com/lib/pq
func NewPostBox(dbString string) (*postBox, error) {
	db, err := sql.Open("postgres", dbString)
	if err != nil {
		return nil, err
	}

	return &postBox{db}, nil
}

// Extracts a unique ID from a database sequence.
func (p *postBox) NextID() (uint64, error) {
	var id uint64

	row := p.db.QueryRow("SELECT nextval('message_ids')")
	err := row.Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

// Places a messages onto the queue.
func (p *postBox) Enqueue(msg *Message) error {
	rcpt := MakeAddressList(msg.Rcpt())

	_, err := p.db.Exec(
		"INSERT INTO messages VALUES (?, ?, ?, ?)",
		msg.ID, msg.From().String(), rcpt, msg.Raw,
	)

	if err != nil {
		return err
	}

	_, err = p.db.Exec(
		"INSERT INTO queue VALUES (?, ?, NOW(), ?)",
		msg.ID, rcpt, 0,
	)

	return err
}

// Does a query for the given address. See QueryStatus for return types.
func (p *postBox) Query(addr *mail.Address) QueryStatus {
	return QUERY_STATUS_SUCCESS
}

// Closes the database connection.
func (p *postBox) Close() error {
	return p.db.Close()
}
