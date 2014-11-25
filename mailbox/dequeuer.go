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
	rows, err := mb.db.Query(sqlPopQueue, limit)
	if err != nil {
		return nil, err
	}
	jobs := make(map[string]Delivery)
	for rows.Next() {
		var row struct {
			Host     string
			User     string
			Date     pq.NullTime
			Attempts int
			MID      uint64
			MRaw     string
			MFrom    string
		}
		err := rows.Scan(&row.Host, &row.MID, &row.User, &row.Date,
			&row.Attempts, &row.MRaw, &row.MFrom)
		if err != nil {
			return jobs, err
		}
		if jobs[row.Host] == nil {
			jobs[row.Host] = make(Delivery)
		}
	}
	return jobs, nil
}

// Report removes a delivered task from the queue or marks an extra attempt
// if it has failed.
func (mb mailBox) Report(user, host string, msgID uint64, delivered bool) error {
	return nil
}

// > RhodiumToad:
// if you want the newest, you change order by date_added, host to order by date_added desc, host desc (in both places)
// and change the > to a <

// all rows in table for latest N hosts
var sqlPopQueue = `
with recursive
  qh(last_host, hosts_seen, date_cutoff)
    as ((select host,
                array[host],
                date_added
           from queue
          order by date_added,host
          limit 1)
        union all
        (select q.host,
                qh.hosts_seen || q.host,
                q.date_added
           from qh,
                lateral (select host, date_added
                           from queue q2
                          where q2.host <> ALL (qh.hosts_seen)
                            and (q2.date_added,q2.host) > (qh.date_cutoff,qh.last_host)
                          order by q2.date_added,q2.host
                          limit 1) q
          where array_length(qh.hosts_seen,1) < $1))

        select queue.*, messages.raw, messages.from
          from queue 
	inner join messages 
	        on messages.id=queue.message_id 
	     where host in (select last_host from qh);`
