package mailbox

import (
	"log"
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

// Dequeue will retrieve the given number of jobs ordered by date
func (p *mailBox) Dequeue(jobs []*Job) (n int, err error) {
	rows, err := p.db.Query(`
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
		on (message_id=id)`, len(jobs))

	if err != nil {
		return
	}

	job := &Job{new(Message), make(map[string][]string)}

	for rows.Next() {
		var rcptList, from string
		var addrList []*mail.Address
		var addrFrom *mail.Address

		err = rows.Scan(&job.Msg.ID, &from, &rcptList, &job.Msg.Raw)
		if err != nil {
			return
		}

		addrList, err = mail.ParseAddressList(rcptList)
		if err != nil {
			log.Println(rcptList)
			return
		}
		addrFrom, err = mail.ParseAddress(from)
		if err != nil {
			return
		}
		job.Msg.SetFrom(addrFrom)

		log.Printf("Job:\r\n%+v\r\nList: %+v\r\n\r\n", job, addrList)

		n++
	}

	return
}

func (p *mailBox) Update(j ...*Job) error {
	return nil
}
