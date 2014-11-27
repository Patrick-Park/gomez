// This is still a draft. Work in progress.
package mailbox

import (
	"net/mail"

	"github.com/lib/pq"
)

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
func (mb mailBox) Dequeue(limit int) (map[string]Delivery, error) {
	type queueEntry struct {
		User, Host  string
		Date        pq.NullTime
		Attempts    int
		MID         uint64
		MRaw, MFrom string
	}
	rows, err := mb.dequeueStmt.Query(limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make(map[string]Delivery)
	cache := make(map[uint64]*Message)
	for rows.Next() {
		var row queueEntry
		err := rows.Scan(&row.Host, &row.MID, &row.User, &row.Date,
			&row.Attempts, &row.MRaw, &row.MFrom)
		if err != nil {
			return jobs, err
		}
		if jobs[row.Host] == nil {
			jobs[row.Host] = make(Delivery)
		}
		msg, ok := cache[row.MID]
		if !ok {
			msg = &Message{
				ID:  row.MID,
				Raw: row.MRaw,
			}
			addr, err := mail.ParseAddress(row.MFrom)
			if err != nil {
				return nil, err
			}
			msg.SetFrom(addr)
			cache[row.MID] = msg
		}
		if jobs[row.Host][msg] == nil {
			jobs[row.Host][msg] = make([]*mail.Address, 0, 1)
		}
		dest, err := mail.ParseAddress(row.User + "@" + row.Host)
		if err != nil {
			return nil, err
		}
		jobs[row.Host][msg] = append(jobs[row.Host][msg], dest)
	}
	return jobs, nil
}

// Report removes a delivered task from the queue or marks an extra attempt
// if it has failed.
func (mb mailBox) Report(user, host string, msgID uint64, delivered bool) error {
	return nil
}
