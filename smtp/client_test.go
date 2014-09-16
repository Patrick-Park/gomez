package smtp

import (
	"net"
	"net/textproto"
	"sync"
	"testing"

	"github.com/gbbr/gomez"
)

// It should pass the correct message to the host's Run
// and it should be able to alter the client's InputMode
func TestClientServe(t *testing.T) {
	var (
		wg  sync.WaitGroup
		msg string
	)

	hostMock := &MockMailService{
		Run_: func(ctx *Client, params string) {
			if params == "MODE" {
				ctx.Mode = MODE_MAIL
			} else if params == "QUIT" {
				ctx.Mode = MODE_QUIT
			}

			msg = params
		},
	}

	cc, sc := net.Pipe()
	cconn, sconn := textproto.NewConn(cc), textproto.NewConn(sc)

	testClient := &Client{
		Host: hostMock,
		Mode: MODE_HELO,
		conn: sconn,
	}

	wg.Add(1)
	go func() {
		testClient.Serve()
		wg.Done()
	}()

	testCases := []struct {
		msg     string
		expMode InputMode
	}{
		{"ABCD", MODE_HELO},
		{"MODE", MODE_MAIL},
		{"Random message", MODE_MAIL},
		{"QUIT", MODE_QUIT}, // Serve will stop after QUIT so it must come last
	}

	for _, test := range testCases {
		err := cconn.PrintfLine(test.msg)
		if err != nil {
			t.Errorf("Error sending command (%s)", err)
		}

		if msg != test.msg || testClient.Mode != test.expMode {
			t.Errorf("Expected msg (%s) and mode (%s), but got (%s) and (%s).", test.expMode, test.expMode, msg, testClient.Mode)
		}
	}

	wg.Wait()
	cconn.Close()
}

// It should reset the client's state (ID, InputMode and Message)
func TestClientReset(t *testing.T) {
	testClient := &Client{
		Mode: MODE_RCPT,
		msg:  new(gomez.Message),
		Id:   "Mike",
	}

	testClient.msg.SetBody("Message body.")

	testClient.Reset()
	if testClient.Mode != MODE_MAIL || testClient.msg.Body() != "" || testClient.Id != "" {
		t.Error("Did not reset client correctly.")
	}
}

// It should reply using the attached network connection
func TestClientNotify(t *testing.T) {
	sc, cc := net.Pipe()
	cconn := textproto.NewConn(cc)

	testClient := &Client{conn: textproto.NewConn(sc)}

	go testClient.Notify(Reply{200, "Hello"})
	msg, err := cconn.ReadLine()
	if err != nil {
		t.Errorf("Error reading end of pipe (%s)", err)
	}

	if msg != "200 Hello" {
		t.Errorf("Got '%s' but expected '%s'", msg, "200 Hello")
	}
}
