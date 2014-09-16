package smtp

import (
	"net"
	"net/textproto"
	"sync"
	"testing"
)

func TestClientServe(t *testing.T) {
	var (
		wg  sync.WaitGroup
		msg string
	)

	cc, sc := net.Pipe()
	cconn, sconn := textproto.NewConn(cc), textproto.NewConn(sc)

	testClient := &Client{
		Host: &MockHost{
			Run_: func(ctx *Client, params string) {
				if params == "MODE" {
					ctx.Mode++
				} else if params == "QUIT" {
					ctx.Mode = MODE_QUIT
				}

				msg = params
			},
		},
		Mode: MODE_HELO,
		conn: sconn,
	}

	wg.Add(1)
	go func() {
		testClient.Serve()
		wg.Done()
	}()

	err := cconn.PrintfLine("MODE")
	if err != nil {
		t.Errorf("Error sending command (%s)", err)
	}

	if msg != "MODE" || testClient.Mode != MODE_MAIL {
		t.Errorf("Expected msg (MODE) and mode (%s), but got (%s) and (%s).", MODE_MAIL, msg, testClient.Mode)
	}

	err = cconn.PrintfLine("QUIT")
	if err != nil {
		t.Errorf("Error sending command (%s)", err)
	}

	if msg != "QUIT" || testClient.Mode != MODE_QUIT {
		t.Errorf("Expected msg (QUIT) and mode (%s), but got (%s) and (%s).", MODE_QUIT, msg, testClient.Mode)
	}

	wg.Wait()
	cconn.Close()

	// Test reset
	testClient.Reset()
	if testClient.Mode != MODE_HELO || testClient.msg.Body() != "" {
		t.Error("Did not reset client correctly.")
	}
}
