package smtp

import (
	"net"
	"net/textproto"
	"sync"
	"testing"

	"github.com/gbbr/gomez"
)

var wg sync.WaitGroup

// If deadlock occurs it means that Client.Serve() did not exit
func TestSmtpClient(t *testing.T) {
	spec := make(CommandSpec)

	spec.Register("DATA", Command{
		Action: func(c *Client, msg string) Reply {
			c.Mode = MODE_DATA
			return Reply{1, msg}
		},
		SupportedMode: MODE_HELO,
		ReplyInvalid:  Reply{0, "Invalid"},
	})

	srv := &Server{
		cs:      &spec,
		Mailbox: new(gomez.MockMailbox),
	}

	sc, cc := net.Pipe()
	sconn, cconn := textproto.NewConn(sc), textproto.NewConn(cc)

	client := &Client{
		msg:  new(gomez.Message),
		Mode: MODE_HELO,
		conn: sconn,
		Host: srv,
	}

	wg.Add(1)

	go func() {
		client.Serve()
		wg.Done()
	}()

	cconn.PrintfLine("DATA Hello world")

	rcv, err := cconn.ReadLine()
	if err != nil {
		t.Error(err)
	}

	if rcv != "1 - Hello world" {
		t.Errorf("Expected to receive '1 - Hello world' but got '%s'", rcv)
	}

	if client.Mode != MODE_DATA {
		t.Error("Client did not enter DATA mode as instructed")
	}

	cconn.PrintfLine("Line 1")
	if client.msg.Body() != "Line 1" {
		t.Error("Did not write to message body")
	}

	cconn.PrintfLine(".")
	if client.msg.Body() != "" || client.Mode != MODE_HELO {
		t.Error("Did not reset client after .")
	}

	cconn.PrintfLine("QUIT")
	wg.Wait()
}
