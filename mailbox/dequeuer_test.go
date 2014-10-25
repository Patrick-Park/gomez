package mailbox

import (
	"database/sql"
	"errors"
	"fmt"
	"net/mail"
	"reflect"
	"testing"
)

type testJob struct {
	MID             int
	From, Rcpt, Raw string
	Dest            string
}

var setupJobs = []testJob{
	testJob{1, "andy@gmail.com", "jim, jane", "hi!", "<iraq@mommy.co>, <daddy@mommy.co>, jim@ji.joe"},
	testJob{5, `"doe jim" <doe@jim.co.uk>`, "tim, tony", "supe!", `a@b.com, "Eric" <c@d.be>`},
	testJob{7, "robert@hotmail.eu", "janise", "am going home guys!", `"Lemur" <b@bb.b>`},
}

func TestDequeuer_Dequeue(t *testing.T) {
	EnsureTestDB()

	pb, err := New(dbString)
	if err != nil {
		t.Error("Error starting server")
	}
	defer pb.Close()

	for _, test := range []struct {
		setupData []testJob
		n         int
		expJobs   []*Job
	}{
		{setupData: setupJobs, n: 0, expJobs: []*Job{}},
		{setupData: []testJob{}, n: 100, expJobs: []*Job{}},
		{
			setupData: setupJobs,
			n:         1,
			expJobs: []*Job{
				&Job{
					&Message{ID: 1, Raw: "hi!", from: &mail.Address{"", "andy@gmail.com"}},
					map[string][]string{
						"mommy.co": []string{"iraq@mommy.co", "daddy@mommy.co"},
						"ji.joe":   []string{"jim@ji.joe"},
					},
				},
			},
		},
		{
			setupData: setupJobs,
			n:         2,
			expJobs: []*Job{
				&Job{
					&Message{ID: 1, Raw: "hi!", from: &mail.Address{"", "andy@gmail.com"}},
					map[string][]string{
						"mommy.co": []string{"iraq@mommy.co", "daddy@mommy.co"},
						"ji.joe":   []string{"jim@ji.joe"},
					},
				},
				&Job{
					&Message{ID: 5, Raw: "supe!", from: &mail.Address{"doe jim", "doe@jim.co.uk"}},
					map[string][]string{
						"b.com": []string{"a@b.com"},
						"d.be":  []string{"c@d.be"},
					},
				},
			},
		},
		{
			setupData: setupJobs,
			n:         5,
			expJobs: []*Job{
				&Job{
					&Message{ID: 1, Raw: "hi!", from: &mail.Address{"", "andy@gmail.com"}},
					map[string][]string{
						"mommy.co": []string{"iraq@mommy.co", "daddy@mommy.co"},
						"ji.joe":   []string{"jim@ji.joe"},
					},
				},
				&Job{
					&Message{ID: 5, Raw: "supe!", from: &mail.Address{"doe jim", "doe@jim.co.uk"}},
					map[string][]string{
						"b.com": []string{"a@b.com"},
						"d.be":  []string{"c@d.be"},
					},
				},
				&Job{
					&Message{ID: 7, Raw: "am going home guys!", from: &mail.Address{"", "robert@hotmail.eu"}},
					map[string][]string{
						"bb.b": []string{"b@bb.b"},
					},
				},
			},
		},
	} {
		CleanDB(pb.db)

		if len(test.setupData) != 0 {
			err = pb.newRunner(test.setupData).run(setupMessages, setupQueue)
			if err != nil {
				t.Errorf("Error settings up messages (is DB clean?): %s", err)
			}
		}

		jobs := make([]*Job, test.n)
		n, err := pb.Dequeue(jobs)
		if test.n == 0 {
			if n != 0 && err != ErrZeroLengthSlice {
				t.Error("Expected error.")
			}
			continue
		}

		if err != nil {
			t.Errorf("Error on dequeue: %s", err)
		}

		if !reflect.DeepEqual(jobs[:n], test.expJobs) {
			t.Errorf("Expected %+v, got %+v", test.expJobs, jobs)
		}
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
