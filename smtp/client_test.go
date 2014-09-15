package smtp

import (
	"net"
	"net/textproto"
	"sync"
	"testing"
)

func TestClientServe(t *testing.T) {
	var wg sync.WaitGroup

	srv := &Server{
		spec: &CommandSpec{
			"TEST": func(ctx *Client, param string) {
				ctx.Mode = MODE_RCPT
				ctx.Reply(Reply{211, param})
			},
			"EXIT": func(ctx *Client, param string) {
				ctx.Mode = MODE_QUIT
				ctx.Reply(Reply{50, "QUITTING"})
			},
		},
	}

	cc, sc := net.Pipe()
	cconn, sconn := textproto.NewConn(cc), textproto.NewConn(sc)

	testClient := &Client{
		Host: srv,
		Mode: MODE_HELO,
		conn: sconn,
	}

	wg.Add(1)
	go func() {
		testClient.Serve()
		wg.Done()
	}()

	cconn.PrintfLine("TEST callback")
	rpl, err := cconn.ReadLine()
	if err != nil {
		t.Errorf("Error reading line: %s", err)
	}

	if rpl != "211 callback" || testClient.Mode != MODE_RCPT {
		t.Errorf("FAILED: Expected (211 callback) but got (%s).", rpl)
	}

	cconn.PrintfLine("EXIT")
	rpl, err = cconn.ReadLine()
	if err != nil {
		t.Errorf("Error reading line: %s", err)
	}

	if rpl != "50 QUITTING" || testClient.Mode != MODE_QUIT {
		t.Errorf("FAILED: Expected (50 QUITTING) but got (%s).", rpl)
	}

	wg.Wait()
	cconn.Close()

	// Test reset
	testClient.Reset()
	if testClient.Mode != MODE_HELO || testClient.msg.Body() != "" {
		t.Error("Did not reset client correctly.")
	}
}
