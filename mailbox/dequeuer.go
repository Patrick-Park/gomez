// This is still a draft. Work in progress.
package mailbox

import (
	"net/mail"

	"github.com/lib/pq"
)

// Indicates the maximum amount of distincts hosts that will be
// retrieved when dequeing.
var MaxHostsPerDequeue = 50

// A Delivery is a set of messages mapped to the recipients that
// they need to be delivered to.
type Delivery map[*Message][]*mail.Address

type Dequeuer interface {
	// Dequeue pulls up to n hosts for delivery and sorts them
	// mapped by the host to package.
	Dequeue() (map[string]Delivery, error)
	// Report updates the status of a delivery. If it was delivered it
	// removes it from the queue, otherwise it updates its status.
	Report(user, host string, msgID uint64, delivered bool) error
}

// Dequeue returns jobs from the queue. It maps hosts to the packages
// that need to be delivered to them. If anything goes wrong, Dequeue
// returns an error. Dequeue will never return more hosts than the
// current MaxHostsPerDequeue value.
func (mb mailBox) Dequeue() (map[string]Delivery, error) {
	rows, err := mb.dequeueStmt.Query(MaxHostsPerDequeue)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	jobs := make(map[string]Delivery)
	cache := make(map[uint64]*Message)

	for rows.Next() {
		var row struct {
			User, Host  string
			Date        pq.NullTime
			Attempts    int
			MID         uint64
			MRaw, MFrom string
		}
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
