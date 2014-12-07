package mailbox

import "database/sql"

// PostgreSQL implementation of the mailbox
type mailBox struct {
	db          *sql.DB
	dequeueStmt *sql.Stmt
}

var _ interface {
	Dequeuer
	Enqueuer
} = (*mailBox)(nil)

// New creates a PostBox using the given connection string. Example
// connection strings can be seen at: http://godoc.org/github.com/lib/pq
func New(dbString string) (*mailBox, error) {
	db, err := sql.Open("postgres", dbString)
	if err != nil {
		return nil, err
	}
	stmt, err := db.Prepare(sqlPopQueue)
	if err != nil {
		return nil, err
	}
	return &mailBox{
		db:          db,
		dequeueStmt: stmt,
	}, nil
}

// all rows in table for latest N hosts
var sqlPopQueue = `
-- RhodiumToad
-- change order by date_added, host to order by date_added desc, host desc
-- (in both places) and change the > to a <
--
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
