package smtp

import (
	"net"
	"net/textproto"
	"testing"

	"github.com/gbbr/gomez"
)

func TestCmdHELO_Param(t *testing.T) {
	client, pipe := getTestClient()

	go cmdHELO(client, "other name")
	_, _, err := pipe.ReadResponse(250)
	if err != nil || client.Mode != MODE_MAIL || client.Id != "other name" {
		t.Errorf("Expected 250, Mode 0 and Id 'name' but got %+v, %d, %s", err, client.Mode, client.Id)
	}

	pipe.Close()
}

func TestCmdHELO_No_Param(t *testing.T) {
	client, pipe := getTestClient()

	go cmdHELO(client, "")
	_, _, err := pipe.ReadResponse(501)
	if err != nil {
		t.Errorf("Expected 501 but got %+v", err)
	}

	pipe.Close()
}

func TestCmdEHLO_No_Param(t *testing.T) {
	client, pipe := getTestClient()

	go cmdEHLO(client, "")
	_, _, err := pipe.ReadResponse(501)
	if err != nil {
		t.Errorf("Expected 501 but got %+v", err)
	}

	pipe.Close()
}

func TestCmdEHLO_Param(t *testing.T) {
	client, pipe := getTestClient()

	go cmdEHLO(client, "name")
	_, _, err := pipe.ReadResponse(250)
	if err != nil || client.Mode != MODE_MAIL || client.Id != "name" {
		t.Errorf("Expected 250, Mode 0 and Id 'name' but got %+v, %d, %s", err, client.Mode, client.Id)
	}

	pipe.Close()
}

func TestCmdMAIL_in_MODE_HELO(t *testing.T) {
	client, pipe := getTestClient()

	client.Mode = MODE_HELO
	go cmdMAIL(client, "FROM:<asd>")
	_, _, err := pipe.ReadResponse(503)
	if err != nil {
		t.Errorf("Expected code 503, got: %#v", err)
	}

	pipe.Close()
}

func TestCmdMAIL_in_MODE_RCPT(t *testing.T) {
	client, pipe := getTestClient()

	client.Mode = MODE_RCPT
	go cmdMAIL(client, "FROM:<asd>")
	_, _, err := pipe.ReadResponse(503)
	if err != nil {
		t.Errorf("Expected code 503, got: %#v", err)
	}

	pipe.Close()
}

func TestCmdMAIL_with_Bad_Syntax(t *testing.T) {
	client, pipe := getTestClient()

	client.Mode = MODE_MAIL
	go cmdMAIL(client, "bad syntax <asd>")
	_, _, err := pipe.ReadResponse(501)
	if err != nil {
		t.Errorf("Expected code 501, got: %#v", err)
	}

	pipe.Close()
}

func TestCmdMAIL_with_bad_address(t *testing.T) {
	client, pipe := getTestClient()

	client.Mode = MODE_MAIL
	go cmdMAIL(client, "FROM:<asd>")
	_, _, err := pipe.ReadResponse(501)
	if err != nil {
		t.Errorf("Expected code 501, got: %#v", err)
	}

	pipe.Close()
}

func TestCmdMAIL_Success(t *testing.T) {
	client, pipe := getTestClient()

	testAddr := gomez.Address{"First Last", "asd", "box.com"}
	client.Mode = MODE_MAIL
	go cmdMAIL(client, "from:First Last <asd@box.com>")
	_, _, err := pipe.ReadResponse(250)
	if err != nil || client.Mode != MODE_RCPT || testAddr.String() != client.msg.From().String() {
		t.Errorf("Expected code 250, address %s got: %#v, %s", testAddr, err, client.msg.From())
	}

	pipe.Close()
}

// Returns a test client and the end of a connection pipe
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
