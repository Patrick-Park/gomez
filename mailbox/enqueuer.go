package mailbox

import (
	"database/sql"
	"fmt"
	"net/mail"

	_ "github.com/lib/pq"
)

// Enqueuer is used to enqueue messages for delivery and to query for addresses.
type Enqueuer interface {
	// Enqueue places a message onto the queue for delivery.
	Enqueue(msg *Message) error

	// NextID obtains a unique identification number in 64 bits.
	NextID() (uint64, error)

	// query searches the server for an address.
	Query(addr *mail.Address) QueryResult
}

// QueryResult reflects the outcome of the result of querying the mailbox
type QueryResult int

const (
	// This state indicates that a user is local, but not found.
	QueryNotFound QueryResult = iota

	// query was successful, and user was found locally.
	QuerySuccess

	// This status reflects that the user is not local.
	QueryNotLocal

	// An error has occurred
	QueryError
)

// PostgreSQL implementation of the mailbox
type postBox struct{ db *sql.DB }

// New creates a PostBox using the given connection string. Example
// connection strings can be seen at: http://godoc.org/github.com/lib/pq
func New(dbString string) (*postBox, error) {
	db, err := sql.Open("postgres", dbString)
	if err != nil {
		return nil, err
	}

	return &postBox{db}, nil
}

// NextID extracts a unique ID from a database sequence.
func (p *postBox) NextID() (uint64, error) {
	var id uint64

	row := p.db.QueryRow("SELECT nextval('message_ids')")
	err := row.Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

// Enqueue delivers to local inboxes and queues remote deliveries.
func (p *postBox) Enqueue(msg *Message) error {
	tx, err := p.db.Begin()
	if err != nil {
		return err
	}

	ec := make(chan error)

	go func() { ec <- p.storeMessage(tx, msg) }()
	go func() { ec <- p.enqueueOutbound(tx, msg) }()
	go func() { ec <- p.deliverInbound(tx, msg) }()

	for i := 0; i < 3; i++ {
		if err := <-ec; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (p *postBox) storeMessage(tx *sql.Tx, msg *Message) error {
	_, err := tx.Exec(
		`INSERT INTO messages (id, "from", rcpt, raw)
		VALUES ($1, $2, $3, $4)`,
		msg.ID, msg.From().String(), makeAddressList(msg.Rcpt()), msg.Raw,
	)

	return err
}

func (p *postBox) enqueueOutbound(tx *sql.Tx, msg *Message) error {
	if len(msg.Outbound()) > 0 {
		_, err := tx.Exec(
			`INSERT INTO queue (message_id, rcpt, date_added, attempts) 
			VALUES ($1, $2, NOW(), 0)`,
			msg.ID, makeAddressList(msg.Outbound()),
		)

		if err != nil {
			return err
		}
	}

	return nil
}

func (p *postBox) deliverInbound(tx *sql.Tx, msg *Message) error {
	if n := len(msg.Inbound()); n > 0 {
		q := "INSERT INTO mailbox (user_id, message_id) VALUES "
		v := "((SELECT id FROM users WHERE address='%s'), %d)"

		for i, addr := range msg.Inbound() {
			q += fmt.Sprintf(v, addr.Address, msg.ID)
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
func (p *postBox) Query(addr *mail.Address) QueryResult {
	return QuerySuccess
}

// Closes the database connection.
func (p *postBox) Close() error {
	return p.db.Close()
}
