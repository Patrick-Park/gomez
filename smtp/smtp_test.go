package smtp

import (
	"github.com/gbbr/gomez"
	"net"
	"net/textproto"
	"sync"
	"testing"
)

var wg sync.WaitGroup

var quitCmd = Command{
	Action: func(c *Client, msg string) Reply {
		c.Mode = MODE_QUIT
		return Reply{221, "2.0.0. Bye"}
	},
	SupportedMode: MODE_FREE,
	ReplyInvalid:  Reply{0, "Invalid"},
}

// If deadlock occurs it means that Client.Serve() did not exit
func TestSmtpClient(t *testing.T) {
	srv := &Server{
		cs: &CommandSpec{
			"DATA": Command{
				Action: func(c *Client, msg string) Reply {
					c.Mode = MODE_DATA
					return Reply{1, msg}
				},
				SupportedMode: MODE_HELO,
				ReplyInvalid:  Reply{0, "Invalid"},
			},
			"QUIT": quitCmd,
		},
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

	cconn.PrintfLine("ABCD")
	rcv, err := cconn.ReadLine()
	if err != nil {
		t.Error(err)
	}

	if rcv != badCommand.String() {
		t.Errorf("Did not return invalid reply for n/a command. Got: %s", rcv)
	}

	cconn.PrintfLine("DATA Hello world")

	rcv, err = cconn.ReadLine()
	if err != nil {
		t.Error(err)
	}

	if rcv != "1 Hello world" {
		t.Errorf("Expected to receive '1 - Hello world' but got '%s'", rcv)
	}

	if client.Mode != MODE_DATA {
		t.Error("Client did not enter DATA mode as instructed")
	}

	cconn.PrintfLine("Line 1")
	cconn.PrintfLine(".")
	rcv, err = cconn.ReadLine()
	if err != nil {
		t.Error(err)
	}

	if client.msg.Body() != "" || client.Mode != MODE_HELO {
		t.Error("Did not reset client after .")
	}

	cconn.PrintfLine("QUIT")
	rcv, err = cconn.ReadLine()
	if err != nil {
		t.Error(err)
	}

	wg.Wait()
}
