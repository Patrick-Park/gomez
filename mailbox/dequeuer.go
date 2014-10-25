package mailbox

// Dequeuer returns jobs off the queue and updates their completion status
type Dequeuer interface {
	// Dequeue n jobs from the queue, up to the slice length.
	Dequeue(jobs []*Job) (n int, err error)

	// Update updates one or more given jobs
	Update(j ...*Job) error
}

// Job holds a message and the recipients it needs to arrive at
type Job struct {
	Msg  Message             // The actual message to be delivered
	Dest map[string][]string // The recipients grouped by host-to-users
}

// Dequeue will retrieve the given number of jobs ordered by date
func (p *mailBox) Dequeue(jobs []*Job) (n int, err error) {
	return 0, nil
}

func (p *mailBox) Update(j ...*Job) error {
	return nil
}
