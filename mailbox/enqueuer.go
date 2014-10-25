package mailbox

import (
	"database/sql"
	"errors"
	"fmt"
	"net/mail"

	_ "github.com/lib/pq"
)

// Enqueuer is used to enqueue messages for delivery and to query for addresses.
type Enqueuer interface {
	// Enqueue places a message onto the queue for delivery.
	Enqueue(msg *Message) error

	// GUID obtains a unique message identification number in 64 bits.
	GUID() (uint64, error)

	// query searches the server for an address.
	Query(addr *mail.Address) QueryResult
}

// QueryResult reflects the outcome of the result of querying the mailbox
type QueryResult int

const (
	// QueryNotFound indicates that a user is local, but not found.
	QueryNotFound QueryResult = iota

	// QuerySuccess indicates that the user was found locally.
	QuerySuccess

	// QueryNotLocal indicates that the user may exist, but is not local.
	QueryNotLocal

	// QueryError indicates that an error happened while querying.
	QueryError
)

// PostgreSQL implementation of the mailbox
type mailBox struct{ db *sql.DB }

// New creates a PostBox using the given connection string. Example
// connection strings can be seen at: http://godoc.org/github.com/lib/pq
func New(dbString string) (*mailBox, error) {
	db, err := sql.Open("postgres", dbString)
	if err != nil {
		return nil, err
	}

	return &mailBox{db}, nil
}

// GUID extracts a unique ID from a database sequence.
func (p *mailBox) GUID() (uint64, error) {
	var id uint64

	row := p.db.QueryRow("SELECT nextval('message_ids')")
	err := row.Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

// Enqueue delivers to local inboxes and queues remote deliveries.
func (p *mailBox) Enqueue(msg *Message) error {
	tx, err := p.db.Begin()
	if err != nil {
		return err
	}

	return newRunner(tx, msg).run(
		storeMessage,
		enqueueOutbound,
		deliverInbound,
	)
}

// runner executes a set of actions in the context of a message transaction
type runner struct {
	tx   *sql.Tx
	data interface{}
}

// newRunner creates a new runner in the given context
func newRunner(tx *sql.Tx, data interface{}) *runner {
	return &runner{tx, data}
}

// run executes a set of actions and returns on the first error
func (msg *runner) run(fn ...func(t *sql.Tx, d interface{}) error) error {
	for _, action := range fn {
		err := action(msg.tx, msg.data)
		if err != nil {
			msg.tx.Rollback()
			return err
		}
	}

	return msg.tx.Commit()
}

// storeMessage is a runner action that saves the message to the db transaction.
func storeMessage(tx *sql.Tx, ctx interface{}) error {
	msg, ok := ctx.(*Message)
	if !ok {
		return errors.New("Expecting *Message in func storeMessage.")
	}

	_, err := tx.Exec(
		`INSERT INTO messages (id, "from", rcpt, raw)
		VALUES ($1, $2, $3, $4)`,
		msg.ID, msg.From().String(), MakeAddressList(msg.Rcpt()), msg.Raw,
	)

	return err
}

// enqueueOutbound adds the message into the queue if it has remote recipients.
func enqueueOutbound(tx *sql.Tx, ctx interface{}) error {
	msg, ok := ctx.(*Message)
	if !ok {
		return errors.New("Expecting *Message in func enqueueOutbound.")
	}

	if len(msg.Outbound()) > 0 {
		_, err := tx.Exec(
			`INSERT INTO queue (message_id, rcpt, date_added, attempts) 
			VALUES ($1, $2, NOW(), 0)`,
			msg.ID, MakeAddressList(msg.Outbound()),
		)

		if err != nil {
			return err
		}
	}

	return nil
}

// deliverInbound delivers mail to local recipients.
func deliverInbound(tx *sql.Tx, ctx interface{}) error {
	msg, ok := ctx.(*Message)
	if !ok {
		return errors.New("Expecting *Message in func deliverOutbound.")
	}

	if n := len(msg.Inbound()); n > 0 {
		q := "INSERT INTO mailbox (user_id, message_id) VALUES "
		v := "((SELECT id FROM users WHERE username='%s' and host='%s'), %d)"

		for i, addr := range msg.Inbound() {
			user, host := SplitUserHost(addr)
			q += fmt.Sprintf(v, user, host, msg.ID)
			if i < n-1 {
				q += ","
			}
		}

		_, err := tx.Exec(q)
		if err != nil {
			return err
		}
	}

	return nil
}

// query searches for the given address. See QueryResult for return types.
func (p *mailBox) Query(addr *mail.Address) QueryResult {
	var result string
	user, host := SplitUserHost(addr)

	// Try and find user as local
	err := p.db.
		QueryRow("SELECT username FROM users WHERE username=$1 AND host=$2", user, host).
		Scan(&result)

	if err != nil && err != sql.ErrNoRows {
		return QueryError
	}

	if result == user {
		return QuerySuccess
	}

	// See if sought for user is on a local host
	err = p.db.
		QueryRow("SELECT host FROM users WHERE host=$1 LIMIT 1", host).
		Scan(&result)

	if err != nil && err != sql.ErrNoRows {
		return QueryError
	}

	if result == host {
		return QueryNotFound
	}

	return QueryNotLocal
}

// Closes the database connection.
func (p *mailBox) Close() error {
	return p.db.Close()
}
