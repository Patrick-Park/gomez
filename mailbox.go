package gomez

import "net/mail"

// Mailbox implements message queueing and
// dequeueing system
type Mailbox interface {
	// Places a message onto the queue for delivery at
	// next pick-up
	Queue(msg Message) error

	// Retrieves all messages from the queue and empties it
	Dequeue() ([]*Message, error)

	// Attempts to deliver a message to a local mailbox
	Deliver(msg Message) error

	// Retrieves all messages for a given address and
	// empties mailbox
	Retrieve(usr *mail.Address) []*Message

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
