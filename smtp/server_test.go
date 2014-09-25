package smtp

import (
	"net"
	"net/mail"
	"net/textproto"
	"sync"
	"testing"

	"github.com/gbbr/gomez"
)

// Should correctly run commands from spec, echo parameters
// and reject bad commands or commands that are not in the spec
func TestServerRun(t *testing.T) {
	srv := &Server{
		spec: &CommandSpec{
			"HELO": func(ctx *Client, params string) error {
				return ctx.Notify(Reply{100, params})
			},
		},
	}

	cc, sc := net.Pipe()
	cconn, sconn := textproto.NewConn(cc), textproto.NewConn(sc)

	testClient := &Client{
		Mode: MODE_HELO,
		conn: sconn,
	}

	testCases := []struct {
		Message string
		Reply   string
	}{
		{"BADFORMAT", badCommand.String()},
		{"GOOD FORMAT", badCommand.String()},
		{"HELO  world ", "100 world"},
		{"HELO  world how are you?", "100 world how are you?"},
		{"HELO", "100 "},
	}

	var wg sync.WaitGroup

	for _, test := range testCases {
		wg.Add(1)

		go func() {
			srv.Run(testClient, test.Message)
			wg.Done()
		}()

		rpl, err := cconn.ReadLine()
		if err != nil {
			t.Errorf("Error reading response %s", err)
		}

		if rpl != test.Reply {
			t.Errorf("Expected '%s' but got '%s'", test.Reply, rpl)
		}

		wg.Wait()
	}
}

// Should create a client that picks up commands
// and greet the connection with 220 status
func TestServerCreateClient(t *testing.T) {
	var wg sync.WaitGroup

	cc, sc := net.Pipe()
	cconn := textproto.NewConn(cc)

	testServer := &Server{
		spec: &CommandSpec{
			"EXIT": func(ctx *Client, params string) error {
				ctx.Mode = MODE_QUIT
				return nil
			},
		},
		config: Config{Hostname: "mecca.local"},
	}

	wg.Add(1)

	go func() {
		testServer.CreateClient(sc)
		wg.Done()
	}()

	_, _, err := cconn.ReadResponse(220)
	if err != nil {
		t.Errorf("Expected code 220 but got %+v", err)
	}

	cconn.PrintfLine("EXIT")
	wg.Wait()
}

func TestServer_Settings(t *testing.T) {
	testServer := &Server{
		config: Config{Hostname: "test", Relay: true},
	}

	flags := testServer.Settings()
	if flags.Hostname != "test" || !flags.Relay {
		t.Error("Did not retrieve correct flags via Settings()")
	}
}

func TestServer_Query_Calls_MailBox(t *testing.T) {
	queryCalled := false
	testServer := &Server{
		Mailbox: &gomez.MockMailbox{
			Query_: func(addr *mail.Address) gomez.QueryStatus {
				queryCalled = true
				return gomez.QUERY_STATUS_SUCCESS
			},
		},
	}

	testServer.Query(&mail.Address{})
	if !queryCalled {
		t.Error("Server Query did not call Mailbox query")
	}
}

func TestServer_Mock(t *testing.T) {
	var mock MailService = &MockMailService{}

	mock.Digest(&Client{})
	mock.Settings()
	mock.Query(&mail.Address{})
	mock.Run(&Client{}, "")
}

// Test E2E
