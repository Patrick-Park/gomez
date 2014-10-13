package gomez

import "net/mail"

var _ Enqueuer = new(MockEnqueuer)

type MockEnqueuer struct {
	NextID_  func() (uint64, error)
	Enqueue_ func(*Message) error
	Query_   func(*mail.Address) QueryStatus
}

func (m MockEnqueuer) Enqueue(msg *Message) error {
	if m.Enqueue_ != nil {
		return m.Enqueue_(msg)
	}

	return nil
}

func (m MockEnqueuer) NextID() (uint64, error) {
	if m.NextID_ != nil {
		return m.NextID_()
	}

	return 0, nil
}

func (m MockEnqueuer) Query(addr *mail.Address) QueryStatus {
	if m.Query_ != nil {
		return m.Query_(addr)
	}

	return QUERY_STATUS_ERROR
}
