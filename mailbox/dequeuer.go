// This is still a draft. Work in progress.
package mailbox

import "net/mail"

// A Delivery is a set of messages mapped to the recipients that
// they need to be delivered to.
type Delivery map[*Message][]*mail.Address

type Dequeuer interface {
	// Dequeue pulls up to n hosts for delivery and sorts them
	// mapped by the host to package.
	Dequeue(n int) (map[string]Delivery, error)

	// Report updates the status of a delivery. If it was delivered it
	// removes it from the queue, otherwise it updates its status.
	Report(user, host string, msgID uint64, delivered bool) error
}

// Dequeue returns up to limit number of hosts along with their deliveries.
// If it fails, Dequeue will return an error.
func (mb *mailBox) Dequeue(limit int) (map[string]Delivery, error) {
	return nil, nil
}

// Report removes a delivered task from the queue or marks an extra attempt
// if it has failed.
func (mb *mailBox) Report(user, host string, msgID uint64, delivered bool) error {
	return nil
}
