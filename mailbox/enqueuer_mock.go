package mailbox

import "net/mail"

var _ Enqueuer = new(MockEnqueuer)

type MockEnqueuer struct {
	NextID_  func() (uint64, error)
	Enqueue_ func(*Message) error
	Query_   func(*mail.Address) QueryResult
}

func (m MockEnqueuer) Enqueue(msg *Message) error           { return m.Enqueue_(msg) }
func (m MockEnqueuer) NextID() (uint64, error)              { return m.NextID_() }
func (m MockEnqueuer) Query(addr *mail.Address) QueryResult { return m.Query_(addr) }
