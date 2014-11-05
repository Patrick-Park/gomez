package mailbox

var _ Dequeuer = (*MockDequeuer)(nil)

// MockDequeuer is a configurable mock for the Dequeuer interface.
type MockDequeuer struct {
	DequeueMock func(n int) (map[string]Package, error)
	FlushMock   func(pkg *Package) error
}

func (m MockDequeuer) Dequeue(n int) (map[string]Package, error) {
	return m.DequeueMock(n)
}

func (m MockDequeuer) Flush(pkg *Package) error {
	return m.FlushMock(pkg)
}
