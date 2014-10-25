package mailbox

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"
)

type testJob struct {
	MID             int
	From, Rcpt, Raw string
	Dest            string
}

func TestDequeuer_Dequeue(t *testing.T) {
	EnsureTestDB()

	pb, err := New(dbString)
	if err != nil {
		t.Error("Error starting server")
	}
	defer pb.Close()

	CleanDB(pb.db)

	setupJobs := []testJob{
		testJob{1, "andy@gmail.com", "jim, jane", "hi!", "<iraq@mommy.co>"},
		testJob{5, `"doe jim" <doe@jim.co.uk>`, "tim, tony", "supe!", `a@b.com, "Eric" <c@d.be>`},
		testJob{7, "robert@hotmail.eu", "janise", "am going home guys!", `"Lemur" <b@bb.b>`},
	}

	err = pb.newRunner(setupJobs).run(setupMessages, setupQueue)
	if err != nil {
		t.Errorf("Error settings up messages: %s", err)
	}

	j := make([]*Job, 2)
	n, err := pb.Dequeue(j)
	if err != nil {
		t.Errorf("Error on dequeue: %s", err)
	}

	t.Logf("Dequeued #%d jobs", n)
	for _, job := range j {
		t.Logf("%+v from %s", job.Msg, job.Msg.From())
	}
}

func setupMessages(tx *sql.Tx, j interface{}) error {
	jobs, ok := j.([]testJob)
	if !ok {
		return errors.New("Expected testJobs in setupMessages.")
	}

	query := `insert into messages(id, "from", rcpt, raw) values `
	for i, j := range jobs {
		query += fmt.Sprintf("(%d,'%s','%s','%s')", j.MID, j.From, j.Rcpt, j.Raw)
		if i < len(jobs)-1 {
			query += ", "
		}
	}

	_, err := tx.Exec(query)
	return err
}

func setupQueue(tx *sql.Tx, j interface{}) error {
	jobs, ok := j.([]testJob)
	if !ok {
		return errors.New("Expected testJobs in setupMessages.")
	}

	query := "insert into queue(message_id, rcpt, date_added, attempts) values "
	for i, j := range jobs {
		query += fmt.Sprintf("(%d,'%s',now(),0)", j.MID, j.Dest)
		if i < len(jobs)-1 {
			query += ", "
		}
	}

	_, err := tx.Exec(query)
	return err
}
