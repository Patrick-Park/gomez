package mailbox

var _ Dequeuer = (*MockDequeuer)(nil)

// MockDequeuer is a configurable mock for the Dequeuer interface.
type MockDequeuer struct {
	DequeueMock func() (map[string]Package, error)
	ReportMock  func(user, host string, msgID uint64, delivered bool) error
}

func (m MockDequeuer) Dequeue() (map[string]Package, error) {
	return m.DequeueMock()
}

func (m MockDequeuer) Report(user, host string, msgID uint64, delivered bool) error {
	return m.ReportMock(user, host, msgID, delivered)
}
