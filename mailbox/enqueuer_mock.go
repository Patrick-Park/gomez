package mailbox

import "net/mail"

var _ Enqueuer = new(MockEnqueuer)

// MockEnqueuer is a configurable mock that implements the Enqueuer interface.
type MockEnqueuer struct {
	NextIDMock  func() (uint64, error)
	EnqueueMock func(*Message) error
	QueryMock   func(*mail.Address) QueryResult
}

func (m MockEnqueuer) Enqueue(msg *Message) error           { return m.EnqueueMock(msg) }
func (m MockEnqueuer) NextID() (uint64, error)              { return m.NextIDMock() }
func (m MockEnqueuer) Query(addr *mail.Address) QueryResult { return m.QueryMock(addr) }
