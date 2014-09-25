package gomez

import "net/mail"

var _ Mailbox = new(MockMailbox)

type MockMailbox struct {
	Queue_    func(Message) error
	Dequeue_  func() ([]*Message, error)
	Deliver_  func(Message) error
	Retrieve_ func(*mail.Address) []*Message
	Query_    func(*mail.Address) QueryStatus
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

func (m MockMailbox) Retrieve(addr *mail.Address) []*Message {
	if m.Retrieve_ != nil {
		return m.Retrieve_(addr)
	}

	return nil
}

func (m MockMailbox) Query(addr *mail.Address) QueryStatus {
	if m.Query_ != nil {
		return m.Query_(addr)
	}

	return QUERY_STATUS_ERROR
}
