package smtp

import (
	"net"
	"net/textproto"
	"strings"
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

func TestCmdEHLO_Param(t *testing.T) {
	client, pipe := getTestClient()

	go cmdEHLO(client, "name")
	_, _, err := pipe.ReadResponse(250)
	if err != nil || client.Mode != MODE_MAIL || client.Id != "name" {
		t.Errorf("Expected 250, Mode 0 and Id 'name' but got %+v, %d, %s", err, client.Mode, client.Id)
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
	client.Mode = MODE_MAIL

	testAddr := gomez.Address{"First Last", "asd", "box.com"}

	go cmdMAIL(client, "from:First Last <asd@box.com>")
	_, _, err := pipe.ReadResponse(250)
	if err != nil || client.Mode != MODE_RCPT || testAddr.String() != client.msg.From().String() {
		t.Errorf("Expected code 250, address %s got: %#v, %s", testAddr, err, client.msg.From())
	}

	pipe.Close()
}

func TestCmdRCPT_in_MODE_HELO(t *testing.T) {
	client, pipe := getTestClient()
	client.Mode = MODE_HELO

	go cmdRCPT(client, "")
	_, _, err := pipe.ReadResponse(503)
	if err != nil {
		t.Errorf("Expected to get a 503 response, got: %s", err)
	}

	pipe.Close()
}

func TestCmdRCPT_in_MODE_MAIL(t *testing.T) {
	client, pipe := getTestClient()
	client.Mode = MODE_MAIL

	go cmdRCPT(client, "")
	_, _, err := pipe.ReadResponse(503)
	if err != nil {
		t.Errorf("Expected to get a 503 response, got: %s", err)
	}

	pipe.Close()
}

func TestCmdRCPT_Bad_Syntax(t *testing.T) {
	client, pipe := getTestClient()
	client.Mode = MODE_RCPT

	go cmdRCPT(client, "invalid")
	_, _, err := pipe.ReadResponse(501)
	if err != nil {
		t.Errorf("Expected to get a 501 response, got: %s", err)
	}

	pipe.Close()
}

func TestCmdRCPT_Bad_Address(t *testing.T) {
	client, pipe := getTestClient()
	client.Mode = MODE_RCPT

	go cmdRCPT(client, "TO:<guyhost.tld>")
	_, _, err := pipe.ReadResponse(501)
	if err != nil {
		t.Errorf("Expected to get a 501 response, got: %s", err)
	}

	pipe.Close()
}

func TestCmdRCPT_User_Not_Found(t *testing.T) {
	client, pipe := getTestClient()
	client.Mode = MODE_RCPT

	go cmdRCPT(client, "TO:Guy <not_found@host.tld>")
	_, _, err := pipe.ReadResponse(550)
	if err != nil {
		t.Errorf("Expected to get a 550 response, got: %s", err)
	}

	pipe.Close()
}

func TestCmdRCPT_Relay_Disabled_User_Not_Local(t *testing.T) {
	client, pipe := getTestClient()
	client.Mode = MODE_RCPT

	go cmdRCPT(client, "TO:<not_local@host.tld>")
	_, _, err := pipe.ReadResponse(550)
	if err != nil {
		t.Errorf("Expected to get a 550 response, got: %s", err)
	}

	pipe.Close()
}

func TestCmdRCPT_Relay_Enabled_User_Not_Local(t *testing.T) {
	client, pipe := getTestClient()

	client.Mode = MODE_RCPT
	client.Host = &MockMailService{
		Query_:    func(addr gomez.Address) gomez.QueryStatus { return gomez.QUERY_STATUS_NOT_LOCAL },
		Settings_: func() Config { return Config{Relay: true} },
	}

	go cmdRCPT(client, "TO:<not_local@host.tld>")
	_, _, err := pipe.ReadResponse(251)
	if err != nil || client.Mode != MODE_DATA || client.msg.Rcpt()[0].Host != "host.tld" || client.msg.Rcpt()[0].User != "not_local" {
		t.Errorf("Expected to get a 251 response and a recipient, got: %s and '%s'", err, client.msg.Rcpt()[0])
	}

	pipe.Close()
}

func TestCmdRCPT_Success(t *testing.T) {
	client, pipe := getTestClient()
	client.Mode = MODE_RCPT

	go cmdRCPT(client, "TO:<success@host.tld>")
	_, _, err := pipe.ReadResponse(250)
	if err != nil || client.Mode != MODE_DATA || client.msg.Rcpt()[0].Host != "host.tld" || client.msg.Rcpt()[0].User != "success" {
		t.Errorf("Expected to get a 250 response and a recipient, got: %s and '%s'", err, client.msg.Rcpt()[0])
	}

	pipe.Close()
}

func TestCmdRCPT_Internal_Error(t *testing.T) {
	client, pipe := getTestClient()
	client.Mode = MODE_RCPT

	go cmdRCPT(client, "TO:<error@host.tld>")
	_, _, err := pipe.ReadResponse(451)
	if err != nil || client.Mode != MODE_RCPT || len(client.msg.Rcpt()) != 0 {
		t.Errorf("Expected to get a 451 response and a recipient, got: %s and '%s'", err, client.msg.Rcpt()[0])
	}

	pipe.Close()
}

func TestCmdDATA_Wrong_Mode(t *testing.T) {
	client, pipe := getTestClient()
	defer pipe.Close()

	testSuite := []struct {
		Mode      InputMode
		ReplyCode int
	}{
		{MODE_HELO, 503},
		{MODE_MAIL, 503},
		{MODE_RCPT, 503},
	}

	go func() {
		for _, test := range testSuite {
			client.Mode = test.Mode
			cmdDATA(client, "")
		}
	}()

	for _, test := range testSuite {
		_, _, err := pipe.ReadResponse(test.ReplyCode)
		if err != nil {
			t.Errorf("Expected code '%d', but got: '%s'", test.ReplyCode, err)
		}
	}
}

func TestCmdDATA_Success(t *testing.T) {
	client, pipe := getTestClient()
	defer pipe.Close()

	digestCalled := false

	client.Host = &MockMailService{
		Digest_: func(ctx *Client) error {
			digestCalled = true

			if !strings.HasPrefix(ctx.msg.Body(), "Line 1 of") {
				t.Errorf("Digest was not called with desired message, got: %s", ctx.msg.Body())
			}

			return nil
		},
	}

	client.Mode = MODE_DATA
	go cmdDATA(client, "these params are ignored")

	pipe.PrintfLine("Line 1 of text")
	pipe.PrintfLine("Line 2 of text")
	pipe.PrintfLine(".")

	_, _, err := pipe.ReadResponse(250)
	if err != nil || !digestCalled || client.msg.Body() != "" || client.Mode != MODE_MAIL || client.Id != "" {
		t.Errorf("Expected response 250, to call digest and reset client, but got: %s and digestCalled == %+v", err, digestCalled)
	}
}

func TestCmdRSET(t *testing.T) {
	client, pipe := getTestClient()
	defer pipe.Close()

	client.Mode = MODE_DATA
	client.msg.SetBody("ABCD")
	client.Id = "Jonah"

	go cmdRSET(client, "")
	_, _, err := pipe.ReadResponse(250)
	if err != nil || client.Id != "" || client.msg.Body() != "" || client.Mode != MODE_MAIL {
		t.Error("Did not reset client correctly")
	}
}

func TestCmdNOOP(t *testing.T) {
	client, pipe := getTestClient()
	defer pipe.Close()

	go cmdNOOP(client, "")
	_, _, err := pipe.ReadResponse(250)
	if err != nil {
		t.Error("Did not receive OK on NOOP")
	}
}

func TestCmdQUIT(t *testing.T) {
	client, pipe := getTestClient()
	defer pipe.Close()

	go cmdQUIT(client, "")
	_, _, err := pipe.ReadResponse(221)
	if err != nil || client.Mode != MODE_QUIT {
		t.Error("QUIT did not work as expected")
	}
}

func TestCmdVRFY_Is_Disabled(t *testing.T) {
	client, pipe := getTestClient()
	defer pipe.Close()

	go cmdVRFY(client, "some@name.com")
	_, _, err := pipe.ReadResponse(252)
	if err != nil {
		t.Error("VRFY did not work as expected")
	}
}

// Returns a test client and the end of a connection pipe
func getTestClient() (*Client, *textproto.Conn) {
	sc, cc := net.Pipe()
	sconn, cconn := textproto.NewConn(sc), textproto.NewConn(cc)

	client := &Client{
		conn: sconn,
		Mode: MODE_HELO,
		msg:  new(gomez.Message),
		Host: &MockMailService{
			Query_: func(addr gomez.Address) gomez.QueryStatus {
				switch addr.User {
				case "not_found":
					return gomez.QUERY_STATUS_NOT_FOUND
				case "not_local":
					return gomez.QUERY_STATUS_NOT_LOCAL
				case "success":
					return gomez.QUERY_STATUS_SUCCESS
				}

				return gomez.QUERY_STATUS_ERROR
			},
		},
	}

	return client, cconn
}
