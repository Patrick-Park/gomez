package mailbox

import (
	"database/sql"
	"errors"
	"net/mail"

	_ "github.com/lib/pq"
)

// Enqueuer is used to enqueue messages for delivery and to query for addresses.
type Enqueuer interface {
	// Enqueue places a message onto the queue for delivery.
	Enqueue(msg *Message) error
	// GUID obtains a unique message identification number in 64 bits.
	GUID() (uint64, error)
	// query searches the server for an address and returns a query status, which
	// can be QueryNotFound, QuerySuccess, QueryNotLocal or QueryError.
	Query(addr *mail.Address) int
}

const (
	// QueryNotFound indicates that a user is local, but not found.
	QueryNotFound = iota
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
func (mb mailBox) GUID() (id uint64, err error) {
	err = mb.db.QueryRow("SELECT nextval('message_ids')").Scan(&id)
	return
}

// Enqueue delivers to local inboxes and queues remote deliveries.
func (mb mailBox) Enqueue(msg *Message) error {
	return mb.newTransaction(msg).do(
		storeMessage,
		enqueueOutbound,
		deliverInbound,
	)
}

// dataTransaction can execute multiple actions within a database transaction using a context
type dataTransaction struct {
	db      *sql.DB
	context interface{}
}

// newTransaction creates a new dataTransaction in the given context
func (mb mailBox) newTransaction(data interface{}) *dataTransaction {
	return &dataTransaction{mb.db, data}
}

// run executes a set of actions and returns on the first error
func (dt dataTransaction) do(fn ...func(t *sql.Tx, d interface{}) error) error {
	tx, err := dt.db.Begin()
	if err != nil {
		return err
	}
	for _, action := range fn {
		err := action(tx, dt.context)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// storeMessage is a dataTransaction action that saves the message to the db transaction.
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
	stmt, err := tx.Prepare(`
		INSERT INTO queue 
		(host, message_id, "user", date_added, attempts) 
		VALUES ($1, $2, $3, NOW(), 0)`)

	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, dst := range msg.Outbound() {
		u, h := SplitUserHost(dst)
		_, err = stmt.Exec(h, msg.ID, u)
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
	stmt, err := tx.Prepare(`
		INSERT INTO mailbox (user_id, message_id) VALUES 
		((SELECT id FROM users WHERE username=$1 and host=$2), $3)`)

	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, rcv := range msg.Inbound() {
		u, h := SplitUserHost(rcv)
		_, err = stmt.Exec(u, h, msg.ID)
		if err != nil {
			return err
		}
	}
	return nil
}

// query searches for the given address. See int for return types.
func (mb mailBox) Query(addr *mail.Address) int {
	var result string
	user, host := SplitUserHost(addr)
	err := mb.db.
		QueryRow("SELECT username FROM users WHERE username=$1 AND host=$2", user, host).
		Scan(&result)

	switch {
	case err != nil && err != sql.ErrNoRows:
		return QueryError
	case result == user:
		return QuerySuccess
	}
	err = mb.db.
		QueryRow("SELECT host FROM users WHERE host=$1 LIMIT 1", host).
		Scan(&result)

	switch {
	case err != nil && err != sql.ErrNoRows:
		return QueryError
	case result == host:
		return QueryNotFound
	default:
		return QueryNotLocal
	}
}

// Closes the database connection.
func (mb mailBox) Close() error {
	return mb.db.Close()
}
