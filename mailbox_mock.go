package gomez

import "net/mail"

var _ Mailbox = new(MockMailbox)

type MockMailbox struct {
	NextID_ func() (uint64, error)
	Queue_  func(*Message) error
	Query_  func(*mail.Address) QueryStatus
}

func (m MockMailbox) Queue(msg *Message) error {
	if m.Queue_ != nil {
		return m.Queue_(msg)
	}

	return nil
}

func (m MockMailbox) NextID() (uint64, error) {
	if m.NextID_ != nil {
		return m.NextID_()
	}

	return 0, nil
}

func (m MockMailbox) Query(addr *mail.Address) QueryStatus {
	if m.Query_ != nil {
		return m.Query_(addr)
	}

	return QUERY_STATUS_ERROR
}
