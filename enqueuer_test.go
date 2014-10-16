package gomez

import (
	"log"
	"os"
	"os/exec"
	"sync"
	"testing"
)

const (
	dbUser         = "Gabriel"
	dbString       = "user=" + dbUser + " dbname=gomez_test sslmode=disable"
	testSchemaFile = "schema_test.sql"
)

var once sync.Once

// Ensures the test database is set up. Should be called before each test
func EnsureTestDB() {
	once.Do(setUpTestDB)
}

// Sets up the test database from the schema file.
func setUpTestDB() {
	file, err := os.Open(testSchemaFile)
	if err != nil {
		log.Fatalf("Error opening schema file: %s", err)
	}
	defer file.Close()

	cmd := exec.Command("psql", "--username="+dbUser, "-q")
	cmd.Stdin = file

	err = cmd.Run()
	if err != nil {
		log.Fatalf("Error setting up DB: %s", err)
	}
}

func TestPostBox_DB_Connection(t *testing.T) {
	EnsureTestDB()

	pb, err := NewPostBox(dbString)
	if err != nil {
		t.Errorf("Could not open DB:", err)
	}

	_, err = pb.db.Query("SELECT * FROM messages LIMIT 1")
	if err != nil {
		t.Errorf("Cannot query: %s", err)
	}
}

func TestPostBox_NextID_Error(t *testing.T) {
	pb, err := NewPostBox("bogus")
	if err != nil {
		t.Errorf("Could not open DB:", err)
	}

	_, err = pb.NextID()
	if err == nil {
		t.Error("Was expecting an error.")
	}
}

func TestPostBox_NextID_Success(t *testing.T) {
	EnsureTestDB()

	pb, err := NewPostBox(dbString)
	if err != nil {
		t.Errorf("Could not open DB:", err)
	}

	id, err := pb.NextID()
	if err != nil {
		t.Errorf("Failed to extract sequence val: %s", err)
	}

	t.Log(id)
}
