package smtp

import (
	"net"
	"net/textproto"
	"testing"

	"github.com/gbbr/gomez"
)

// Should reply 501 if no params are present.
// Should reply 250 if params are present, identify the client
// and set the InputMode to MAIL
func TestCmdHELOandEHLO(t *testing.T) {
	client, pipe := getTestClient()

	go cmdHELO(client, "")
	_, _, err := pipe.ReadResponse(501)
	if err != nil {
		t.Errorf("Expected 501 but got %+v", err)
	}

	go cmdEHLO(client, "")
	_, _, err = pipe.ReadResponse(501)
	if err != nil {
		t.Errorf("Expected 501 but got %+v", err)
	}

	go cmdEHLO(client, "name")
	_, _, err = pipe.ReadResponse(250)
	if err != nil || client.Mode != MODE_MAIL || client.Id != "name" {
		t.Errorf("Expected 250, Mode 0 and Id 'name' but got %+v, %d, %s", err, client.Mode, client.Id)
	}

	client.Reset()

	go cmdHELO(client, "other name")
	_, _, err = pipe.ReadResponse(250)
	if err != nil || client.Mode != MODE_MAIL || client.Id != "other name" {
		t.Errorf("Expected 250, Mode 0 and Id 'name' but got %+v, %d, %s", err, client.Mode, client.Id)
	}

	pipe.Close()
}

// Should ask to say HELO if you didn't
// Should not allow nested MAIL command
// Should alert user of correct syntax
// Should set From:, change mode to RCPT and reply 250
func TestCmdMAIL(t *testing.T) {
	client, pipe := getTestClient()

	client.Mode = MODE_HELO
	go cmdMAIL(client, "FROM:<asd>")
	_, _, err := pipe.ReadResponse(503)
	if err != nil {
		t.Errorf("Expected code 503, got: %#v", err)
	}

	client.Mode = MODE_RCPT
	go cmdMAIL(client, "FROM:<asd>")
	_, _, err = pipe.ReadResponse(503)
	if err != nil {
		t.Errorf("Expected code 503, got: %#v", err)
	}

	client.Mode = MODE_MAIL
	go cmdMAIL(client, "bad syntax <asd>")
	_, _, err = pipe.ReadResponse(501)
	if err != nil {
		t.Errorf("Expected code 501, got: %#v", err)
	}

	go cmdMAIL(client, "FROM:<asd>")
	_, _, err = pipe.ReadResponse(501)
	if err != nil {
		t.Errorf("Expected code 501, got: %#v", err)
	}

	testAddr := gomez.Address{"First Last", "asd", "box.com"}
	go cmdMAIL(client, "FROM: First Last <asd@box.com>")
	_, _, err = pipe.ReadResponse(250)
	if err != nil || client.Mode != MODE_RCPT || testAddr.String() != client.msg.From().String() {
		t.Errorf("Expected code 250, address %s got: %#v, %s", testAddr, err, client.msg.From())
	}

	pipe.Close()
}

// Returns a test client and a pipe end to
// receive messages on. The clients host is
// configurable MailService mock
func getTestClient() (*Client, *textproto.Conn) {
	sc, cc := net.Pipe()
	sconn, cconn := textproto.NewConn(sc), textproto.NewConn(cc)

	client := &Client{
		conn: sconn,
		Mode: MODE_HELO,
		msg:  new(gomez.Message),
		Host: new(MockMailService),
	}

	return client, cconn
}
