package smtp

import (
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/textproto"
	"sync"
	"testing"

	"github.com/gbbr/gomez/mailbox"
)

// Test that reply's stringer implentation correctly outputs
// both single-line and multi-line messages.
func TestReplyString(t *testing.T) {
	r := reply{200, "This is a line of text"}
	if r.String() != "200 This is a line of text" {
		t.Errorf("Error stringifying reply %s", r)
	}

	r = reply{123, "This is\nmulti\nline\nresponse"}
	if r.String() != "123-This is\n123-multi\n123-line\n123 response" {
		t.Errorf("Error stringifying reply:\r\n%s", r)
	}
}

// It should pass the correct message to the host's run
// and it should be able to alter the client's transactionState
func TestClientServe(t *testing.T) {
	var (
		wg  sync.WaitGroup
		msg string
	)

	hostMock := &mockHost{
		Run_: func(ctx *transaction, params string) error {
			msg = params

			if params == "MODE" {
				ctx.Mode = stateMAIL
			} else if params == "QUIT" {
				ctx.Mode = stateRCPT
				return io.EOF
			}

			return nil
		},
	}

	cc, sc := net.Pipe()
	cconn, sconn := textproto.NewConn(cc), textproto.NewConn(sc)

	testClient := &transaction{
		Mode: stateHELO,
		host: hostMock,
		text: sconn,
	}

	wg.Add(1)
	go func() {
		testClient.serve()
		wg.Done()
	}()

	testCases := []struct {
		msg     string
		expMode transactionState
	}{
		{"ABCD", stateHELO},
		{"MODE", stateMAIL},
		{"Random message", stateMAIL},
		{"QUIT", stateRCPT}, // serve will stop after QUIT so it must come last
	}

	for _, test := range testCases {
		err := cconn.PrintfLine(test.msg)
		if err != nil {
			t.Errorf("Error sending command (%s)", err)
		}

		if msg != test.msg || testClient.Mode != test.expMode {
			t.Errorf("Expected msg (%v) and mode (%v), but got (%s) and (%v).",
				test.expMode, test.expMode, msg, testClient.Mode)
		}
	}

	wg.Wait()
}

// This test assures that serve does not go into infinite loops
// when clients close connections and that the error branches are hit
func TestClientServe_Error(t *testing.T) {
	cc, sc := net.Pipe()
	cconn, sconn := textproto.NewConn(cc), textproto.NewConn(sc)

	log.SetOutput(ioutil.Discard)

	testClient := &transaction{
		host: &mockHost{
			Run_: func(ctx *transaction, msg string) error {
				return io.EOF
			},
		},
		Mode: stateHELO,
		text: sconn,
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		testClient.serve()
		wg.Done()
	}()

	cc.Close()
	wg.Wait()

	wg.Add(1)
	go func() {
		testClient.serve()
		wg.Done()
	}()

	cconn.PrintfLine("Message")
	wg.Wait()
}

// It should reset the client's state (ID, transactionState and Message)
func TestClientReset(t *testing.T) {
	testClient := &transaction{
		Mode:    stateRCPT,
		Message: new(mailbox.Message),
		ID:      "Mike",
	}

	testClient.Message.Raw = "Message body."

	testClient.reset()
	if testClient.Mode != stateMAIL || testClient.Message.Raw != "" {
		t.Error("Did not reset client correctly.")
	}
}

// It should reply using the attached network connection
func TestClientNotify(t *testing.T) {
	sc, cc := net.Pipe()
	cconn := textproto.NewConn(cc)

	testClient := &transaction{text: textproto.NewConn(sc)}

	go testClient.notify(reply{200, "Hello"})
	msg, err := cconn.ReadLine()
	if err != nil {
		t.Errorf("Error reading end of pipe (%s)", err)
	}

	if msg != "200 Hello" {
		t.Errorf("Got '%s' but expected '%s'", msg, "200 Hello")
	}
}
