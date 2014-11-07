package mailbox

var _ Dequeuer = (*MockDequeuer)(nil)

// MockDequeuer is a configurable mock for the Dequeuer interface.
type MockDequeuer struct {
	DequeueMock func(n int) (map[string]Delivery, error)
	ReportMock  func(user, host string, msgID uint64, delivered bool) error
}

func (m MockDequeuer) Dequeue(n int) (map[string]Delivery, error) {
	return m.DequeueMock(n)
}

func (m MockDequeuer) Report(user, host string, msgID uint64, delivered bool) error {
	return m.ReportMock(user, host, msgID, delivered)
}
