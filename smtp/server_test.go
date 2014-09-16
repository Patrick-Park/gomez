package smtp

import (
	"net"
	"net/textproto"
	"sync"
	"testing"
)

// Should correctly run commands from spec, echo parameters
// and reject bad commands or commands that are not in the spec
func TestServerRun(t *testing.T) {
	srv := &Server{
		spec: &CommandSpec{
			"HELO": func(ctx *Client, params string) {
				ctx.Notify(Reply{100, params})
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

	for _, test := range testCases {
		go srv.Run(testClient, test.Message)
		rpl, err := cconn.ReadLine()
		if err != nil {
			t.Errorf("Error reading response %s", err)
		}

		if rpl != test.Reply {
			t.Errorf("Expected '%s' but got '%s'", test.Reply, rpl)
		}
	}
}

// Should create a client that picks up commands
func TestServerCreateClient(t *testing.T) {
	var wg sync.WaitGroup

	cc, sc := net.Pipe()
	cconn := textproto.NewConn(cc)

	testServer := &Server{
		spec: &CommandSpec{
			"EXIT": func(ctx *Client, params string) {
				ctx.Mode = MODE_QUIT
			},
		},
	}

	wg.Add(1)
	go func() {
		testServer.createClient(sc)
		wg.Done()
	}()

	cconn.PrintfLine("EXIT")
	wg.Wait()
}

// Test E2E
