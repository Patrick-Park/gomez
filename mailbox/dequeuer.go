package mailbox

// Dequeuer returns jobs off the queue and updates their completion status
type Dequeuer interface {
	// Dequeue retrieves a given number of messages from the queue
	Dequeue(n int) ([]*Job, error)

	// Update updates one or more given jobs
	Update(j ...*Job) error
}

// Job holds a message and the recipients it needs to arrive at
type Job struct {
	Msg  Message             // The actual message to be delivered
	Dest map[string][]string // The recipients grouped by host-to-users
}

// Dequeue will retrieve the given number of jobs ordered by date
func (p *mailBox) Dequeue(n int) ([]*Job, error) {
	return nil, nil
}

func (p *mailBox) Update(j ...*Job) error {
	return nil
}
