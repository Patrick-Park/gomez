package gomez

// Mailbox implements message queueing and
// dequeueing system
type Mailbox interface {
	// Places a message into the mailbox
	// to be picked up by the agent
	Queue(msg Message) error

	// Extracts a slice of messages from the queue.
	// Returns an empty slice if there are no messages.
	Dequeue() ([]*Message, error)

	// Attempts to deliver a message to a local user
	Deliver(msg Message) error
}

// A message represents an e-mail message and
// holds information about sender, recepients
// and the message body
type Message struct {
	recp []Address
	from Address
	body string
}

// Adds a new recepient to the message
func (m *Message) AddRcpt(addr Address) {
	recp = append(recp, addr)
}

// An Address holds a user and a host
type Address struct {
	name, host string
}

// Parses a string and returns a new Address
func NewAddress(addr string) *Address {

}
