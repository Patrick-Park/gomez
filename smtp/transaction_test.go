package smtp

import (
	"errors"
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
// and it should be able to alter the client's int
func TestClientServe(t *testing.T) {
	var (
		wg, wgt sync.WaitGroup
		msg     string
		ErrTest = errors.New("Test error")
	)

	hostMock := &mockHost{
		RunMock: func(ctx *transaction, params string) error {
			defer wgt.Done()
			msg = params

			switch params {
			case "MODE":
				ctx.Mode = stateMAIL
				return nil
			case "QUIT":
				ctx.Mode = stateRCPT
				return io.EOF
			case "ERRR":
				return ErrTest
			default:
				return nil
			}
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
		expMode int
	}{
		{"ABCD", stateHELO},
		{"ERRR", stateHELO},
		{"MODE", stateMAIL},
		{"Random message", stateMAIL},
		{"QUIT", stateRCPT}, // serve will stop after QUIT so it must come last
	}

	for _, test := range testCases {
		wgt.Add(1)
		err := cconn.PrintfLine(test.msg)
		if err != ErrTest && err != nil {
			t.Errorf("Error sending command (%s)", err)
		}
		wgt.Wait() // Wait for runner to process

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
			RunMock: func(ctx *transaction, msg string) error {
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

// It should reset the client's state (ID, int and Message)
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
