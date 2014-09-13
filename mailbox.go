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
	Query(q string) *Address
}

// Mailbox interface mock, can be written to override
// behavior using custom functions as a replacement.
type MockMailbox struct {
	Queue_    func(Message) error
	Dequeue_  func() ([]*Message, error)
	Deliver_  func(Message) error
	Retrieve_ func(Address) []*Message
	Query_    func(string) *Address
}

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

func (m MockMailbox) Query(q string) *Address {
	if m.Query_ != nil {
		return m.Query_(q)
	}

	return nil
}
