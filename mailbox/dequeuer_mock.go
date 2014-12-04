package mailbox

import "net/mail"

var _ Dequeuer = (*MockDequeuer)(nil)

// MockDequeuer is a configurable mock for the Dequeuer interface.
type MockDequeuer struct {
	DequeueMock func() (map[string]Package, error)
	ReportMock  func(id uint64, succeeded []*mail.Address, failed []*mail.Address) error
}

func (m MockDequeuer) Dequeue() (map[string]Package, error) {
	return m.DequeueMock()
}

func (m MockDequeuer) Report(id uint64, succeeded []*mail.Address, failed []*mail.Address) error {
	return m.ReportMock(id, succeeded, failed)
}
