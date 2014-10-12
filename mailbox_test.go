package gomez

import "testing"

const dbString = "user=Gabriel dbname=gomez sslmode=disable"

func TestPostBox_DB_Connection(t *testing.T) {
	pb, err := NewPostBox(dbString)
	if err != nil {
		t.Errorf("Could not open DB:", err)
	}

	_, err = pb.db.Query("SELECT * FROM messages LIMIT 1")
	if err != nil {
		t.Errorf("Cannot query: %s", err)
	}
}
