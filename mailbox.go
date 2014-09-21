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
	Query(addr Address) QueryStatus
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

type MockMailbox struct {
	Queue_    func(Message) error
	Dequeue_  func() ([]*Message, error)
	Deliver_  func(Message) error
	Retrieve_ func(Address) []*Message
	Query_    func(Address) QueryStatus
}

var _ Mailbox = new(MockMailbox)

func (m MockMailbox) Queue(msg Message) error {
	if m.Queue_ != nil {
		return m.Queue_(msg)
	}

	return nil
}

func (m MockMailbox) Dequeue() ([]*Message, error) {
	if m.Dequeue_ != nil {
		return m.Dequeue_()
	}

	return nil, nil
}

func (m MockMailbox) Deliver(msg Message) error {
	if m.Deliver_ != nil {
		return m.Deliver_(msg)
	}

	return nil
}

func (m MockMailbox) Retrieve(addr Address) []*Message {
	if m.Retrieve_ != nil {
		return m.Retrieve_(addr)
	}

	return nil
}

func (m MockMailbox) Query(addr Address) QueryStatus {
	if m.Query_ != nil {
		return m.Query_(addr)
	}

	return QUERY_STATUS_ERROR
}
