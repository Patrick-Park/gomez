/*
*
	This file is work in progress. Implementation is only a draft.
*
*/

package mailbox

import "net/mail"

// A Package is a set of messages mapped to the recipients that
// they need to be delivered to.
type Package map[*Message][]*mail.Address

type Dequeuer interface {
	// Dequeue pulls n messages from the queue and sorts them
	// into packages, mapped by the host that they need to be
	// delivered to.
	Dequeue(n int) (map[string]Package, error)

	// Flush flushes the passed package.
	Flush(pkg *Package) error
}

func (mb *mailBox) Dequeue(n int) (map[string]Package, error) {
	pkgs := make(map[string]Package)

	rows, err := mb.db.Query(sqlGetNJobs, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			f, d string
			msg  Message
		)

		err := rows.Scan(&msg.ID, &f, &d, &msg.Raw)
		if err != nil {
			return nil, err
		}

		from, err := mail.ParseAddress(f)
		if err != nil {
			return nil, err
		}

		rcpt, err := mail.ParseAddressList(d)
		if err != nil {
			return nil, err
		}

		msg.from = from
		extendPackages(pkgs, &msg, rcpt)
	}

	return pkgs, nil
}

// extendPackages extends a given package list by adding the passed message
// based on the set recipients.
func extendPackages(pkgs map[string]Package, msg *Message, rcpt []*mail.Address) {
	for _, addr := range rcpt {
		_, h := SplitUserHost(addr)

		if _, ok := pkgs[h]; !ok {
			pkgs[h] = make(Package)
		}

		if _, ok := pkgs[h][msg]; !ok {
			pkgs[h][msg] = make([]*mail.Address, 0, 1)
		}

		pkgs[h][msg] = append(pkgs[h][msg], addr)
	}
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
