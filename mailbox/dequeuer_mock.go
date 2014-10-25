package mailbox

var _ Dequeuer = new(MockDequeuer)

// MockDequeuer is a configurable mock for the Dequeuer interface.
type MockDequeuer struct {
	DequeueMock func([]*Job) (int, error)
	UpdateMock  func(j ...*Job) error
}

func (m MockDequeuer) Dequeue(jobs []*Job) (int, error) { return m.DequeueMock(jobs) }
func (m MockDequeuer) Update(j ...*Job) error           { return m.Update(j...) }
