package mailbox

var _ Dequeuer = new(MockDequeuer)

// MockDequeuer is a configurable mock for the Dequeuer interface.
type MockDequeuer struct {
	DequeueMock func(int) ([]*Job, error)
	UpdateMock  func(j ...*Job) error
}

func (m MockDequeuer) Dequeue(n int) ([]*Job, error) { return m.DequeueMock(n) }
func (m MockDequeuer) Update(j ...*Job) error        { return m.Update(j...) }
