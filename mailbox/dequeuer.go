/*
*
	This file is work in progress. Implementation is only a draft.
*
*/

package mailbox

import (
	"errors"
	"net/mail"
)

type Dequeuer interface {
	// Dequeue n jobs from the queue, up to the slice length.
	Dequeue(jobs []*Job) (n int, err error)
	// Update one or more given jobs
	Update(j ...*Job) error
}

// Job holds a message and the recipients it needs to arrive at
type Job struct {
	Msg  *Message            // The actual message to be delivered
	Dest map[string][]string // The recipients grouped by host-to-users
}

// host to jobs :: we may send multiple messages to one host
// type jobs map[string][]Job
//
// Job
// type Job struct { Msg *Message; Rcpt string }

var ErrZeroLengthSlice = errors.New("The given slice has length 0.")

// Dequeue will retrieve the given number of jobs ordered by date
func (p *mailBox) Dequeue(jobs []*Job) (n int, err error) {
	if jobs == nil || len(jobs) == 0 {
		return 0, ErrZeroLengthSlice
	}

	rows, err := p.db.Query(sqlGetNJobs, len(jobs)) // SQL is at the bottom of file
	if err != nil {
		return
	}

	for n = 0; rows.Next() && n < len(jobs); n++ {
		job := &Job{new(Message), make(map[string][]string)}

		var dest, from string
		err = rows.Scan(&job.Msg.ID, &from, &dest, &job.Msg.Raw)
		if err != nil {
			return
		}

		destList, err := mail.ParseAddressList(dest)
		if err != nil {
			return n, err
		}

		fromAddr, err := mail.ParseAddress(from)
		if err != nil {
			return n, err
		}

		job.Msg.SetFrom(fromAddr)
		job.Dest = groupByHost(destList)

		jobs[n] = job
	}

	return
}

func groupByHost(list []*mail.Address) map[string][]string {
	grp := make(map[string][]string)

	for _, addr := range list {
		_, h := SplitUserHost(addr)
		if _, ok := grp[h]; !ok {
			grp[h] = make([]string, 0, 1)
		}

		grp[h] = append(grp[h], addr.Address)
	}

	return grp
}

func (p *mailBox) Update(j ...*Job) error {
	return nil
}

var sqlGetNJobs = `
	with jobs as (
		update queue main
		set date_added=now(), attempts=attempts+1
		from (
			select message_id, rcpt
			from queue
			order by date_added asc
			limit $1
			for update
		) sub
		where main.message_id = sub.message_id
		returning main.message_id, main.rcpt
	) 

	select id, "from", jobs.rcpt, raw 
	from jobs 
	inner join messages 
	on (message_id=id)`
