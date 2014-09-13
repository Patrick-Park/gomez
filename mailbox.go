package gomez

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
	Retrieve(usr Address) []*Message

	// Queries the server for a user
	Query(q string) Address
}
