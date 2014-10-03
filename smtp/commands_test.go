package smtp

import (
	"io/ioutil"
	"log"
	"net"
	"net/mail"
	"net/textproto"
	"strings"
	"sync"
	"testing"

	"github.com/gbbr/gomez"
)

func TestCmd_Modes_and_Codes(t *testing.T) {
	client, pipe := getTestClient()
	defer pipe.Close()

	testSuite := []struct {
		Fn        func(*Client, string) error
		Param     string
		ExpCode   int
		StartMode InputMode
		ExpMode   InputMode
		Hint      string
	}{
		{cmdHELO, "other name", 250, MODE_HELO, MODE_MAIL, "HELO"},
		{cmdHELO, "", 501, MODE_HELO, MODE_HELO, "HELO"},
		{cmdEHLO, "name", 250, MODE_HELO, MODE_MAIL, "EHLO"},
		{cmdEHLO, "", 501, MODE_HELO, MODE_HELO, "EHLO"},

		{cmdMAIL, "FROM:<asd>", 503, MODE_HELO, MODE_HELO, "MAIL"},
		{cmdMAIL, "FROM:<asd>", 503, MODE_RCPT, MODE_RCPT, "MAIL"},
		{cmdMAIL, "bad syntax", 501, MODE_MAIL, MODE_MAIL, "MAIL"},
		{cmdMAIL, "FROM:<bad_address>", 501, MODE_MAIL, MODE_MAIL, "MAIL"},

		{cmdRCPT, "", 503, MODE_HELO, MODE_HELO, "RCPT"},
		{cmdRCPT, "", 503, MODE_MAIL, MODE_MAIL, "RCPT"},
		{cmdRCPT, "invalid", 501, MODE_RCPT, MODE_RCPT, "RCPT"},
		{cmdRCPT, "TO:<guyhost.tld>", 501, MODE_RCPT, MODE_RCPT, "RCPT"},
		{cmdRCPT, "TO:Guy <not_found@host.tld>", 550, MODE_RCPT, MODE_RCPT, "RCPT"},
		{cmdRCPT, "TO:<not_local@host.tld>", 550, MODE_RCPT, MODE_RCPT, "RCPT"},
		{cmdRCPT, "TO:<success@host.tld>", 250, MODE_RCPT, MODE_DATA, "RCPT"},
		{cmdRCPT, "TO:<error@host.tld>", 451, MODE_RCPT, MODE_RCPT, "RCPT"},

		{cmdDATA, "", 503, MODE_HELO, MODE_HELO, "DATA"},
		{cmdDATA, "", 503, MODE_MAIL, MODE_MAIL, "DATA"},
		{cmdDATA, "", 503, MODE_RCPT, MODE_RCPT, "DATA"},

		{cmdRSET, "", 250, MODE_HELO, MODE_HELO, "RSET"},
		{cmdRSET, "", 250, MODE_RCPT, MODE_MAIL, "RSET"},
		{cmdRSET, "", 250, MODE_DATA, MODE_MAIL, "RSET"},

		{cmdNOOP, "", 250, MODE_RCPT, MODE_RCPT, "NOOP"},
		{cmdNOOP, "", 250, MODE_HELO, MODE_HELO, "NOOP"},

		{cmdQUIT, "", 221, MODE_HELO, MODE_QUIT, "QUIT"},
		{cmdQUIT, "", 221, MODE_MAIL, MODE_QUIT, "QUIT"},
		{cmdQUIT, "", 221, MODE_RCPT, MODE_QUIT, "QUIT"},
		{cmdQUIT, "", 221, MODE_DATA, MODE_QUIT, "QUIT"},

		{cmdVRFY, "", 252, MODE_DATA, MODE_DATA, "VRFY"},
	}

	go func() {
		for _, test := range testSuite {
			client.Mode = test.StartMode
			test.Fn(client, test.Param)
		}
	}()

	for _, test := range testSuite {
		t.Logf("Testing %s %s %d/%d - %d", test.Hint, test.Param, test.StartMode, test.ExpMode, test.ExpCode)
		_, _, err := pipe.ReadResponse(test.ExpCode)
		if err != nil || client.Mode != test.ExpMode {
			t.Errorf("On function %s %s, expected mode %d and code %d, but got %d and code %+v.",
				test.Hint,
				test.Param,
				test.ExpMode,
				test.ExpCode,
				client.Mode,
				err)
		}
	}
}

func TestCmdHELO_Param_Passing(t *testing.T) {
	client, pipe := getTestClient()

	go cmdHELO(client, "other name")
	_, _, err := pipe.ReadResponse(250)
	if err != nil || client.Mode != MODE_MAIL || client.Id != "other name" {
		t.Errorf("Expected 250, Mode 0 and Id 'name' but got %+v, %d, %s", err, client.Mode, client.Id)
	}

	pipe.Close()
}

func TestCmdEHLO_Param_Passing(t *testing.T) {
	client, pipe := getTestClient()

	go cmdEHLO(client, "name")
	_, _, err := pipe.ReadResponse(250)
	if err != nil || client.Mode != MODE_MAIL || client.Id != "name" {
		t.Errorf("Expected 250, Mode 0 and Id 'name' but got %+v, %d, %s", err, client.Mode, client.Id)
	}

	pipe.Close()
}

func TestCmdMAIL_Param_Passing(t *testing.T) {
	client, pipe := getTestClient()
	client.Mode = MODE_MAIL

	testAddr := mail.Address{"First Last", "asd@box.com"}

	go cmdMAIL(client, "from:First Last <asd@box.com>")
	_, _, err := pipe.ReadResponse(250)
	if err != nil || client.Mode != MODE_RCPT || testAddr.String() != client.Message.From().String() {
		t.Errorf("Expected code 250, address %s got: %#v, %s", testAddr, err, client.Message.From())
	}

	pipe.Close()
}

func TestCmdRCPT_User_Not_Local(t *testing.T) {
	client, pipe := getTestClient()

	client.Mode = MODE_RCPT

	// Relay is enabled
	client.host = &MockSMTPServer{
		Query_:    func(addr *mail.Address) gomez.QueryStatus { return gomez.QUERY_STATUS_NOT_LOCAL },
		Settings_: func() Config { return Config{Relay: true} },
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		cmdRCPT(client, "TO:<not_local@host.tld>")
		wg.Done()
	}()

	_, _, err := pipe.ReadResponse(251)
	if err != nil || client.Mode != MODE_DATA || client.Message.Rcpt()[0].Address != "not_local@host.tld" {
		t.Errorf("Expected to get a 251 response and a recipient, got: %s and '%s'", err, client.Message.Rcpt()[0])
	}

	// Relay is disabled
	client.host = &MockSMTPServer{
		Query_:    func(addr *mail.Address) gomez.QueryStatus { return gomez.QUERY_STATUS_NOT_LOCAL },
		Settings_: func() Config { return Config{Relay: false} },
	}

	wg.Wait()

	go cmdRCPT(client, "TO:<not_local@host.tld>")
	_, _, err = pipe.ReadResponse(550)
	if err != nil || client.Mode != MODE_DATA || len(client.Message.Rcpt()) != 1 {
		t.Errorf("Expected to get a 550 response and no recipient, got: %+v", err)
	}

	pipe.Close()
}

func TestCmdRCPT_Param_Passing(t *testing.T) {
	client, pipe := getTestClient()
	client.Mode = MODE_RCPT

	go cmdRCPT(client, "TO:<success@host.tld>")
	_, _, err := pipe.ReadResponse(250)
	if err != nil || client.Mode != MODE_DATA || client.Message.Rcpt()[0].Address != "success@host.tld" {
		t.Errorf("Expected to get a 250 response and a recipient, got: %s and '%s'", err, client.Message.Rcpt()[0])
	}

	pipe.Close()
}

func TestCmdRCPT_Internal_Error(t *testing.T) {
	client, pipe := getTestClient()
	client.Mode = MODE_RCPT

	log.SetOutput(ioutil.Discard)
	go cmdRCPT(client, "TO:<error@host.tld>")

	_, _, err := pipe.ReadResponse(451)
	if err != nil || client.Mode != MODE_RCPT || len(client.Message.Rcpt()) != 0 {
		t.Errorf("Expected to get a 451 response and a recipient, got: %s and '%s'", err, client.Message.Rcpt()[0])
	}

	pipe.Close()
}

func TestCmdDATA_Digest(t *testing.T) {
	calledDigest := false

	client, pipe := getTestClient()
	client.host = &MockSMTPServer{
		Digest_: func(c *Client) error {
			calledDigest = true
			return nil
		},
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		client.Mode = MODE_DATA
		client.Id = "Test_ID"
		cmdDATA(client, "")
		wg.Done()
	}()

	pipe.ReadResponse(354)
	pipe.PrintfLine(".")

	wg.Wait()
	pipe.Close()

	if !calledDigest {
		t.Error("Did not call Digest")
	}
}

func TestCmdDATA_Error_Notify(t *testing.T) {
	client, pipe := getTestClient()
	client.Mode = MODE_DATA

	defer pipe.Close()
	go cmdDATA(client, "these params are ignored")
}

func TestCmdDATA_Error_ReadLines(t *testing.T) {
	client, pipe := getTestClient()
	defer pipe.Close()

	client.Mode = MODE_DATA
	go cmdDATA(client, "these params are ignored")

	pipe.ReadResponse(354)

	pipe.PrintfLine("Line 1 of text")
	pipe.PrintfLine("Line 2 of text")
}

func TestCmdRSET(t *testing.T) {
	client, pipe := getTestClient()
	defer pipe.Close()

	client.Mode = MODE_DATA
	client.Message.Raw = "ABCD"
	client.Id = "Jonah"

	go cmdRSET(client, "")
	_, _, err := pipe.ReadResponse(250)
	if err != nil || client.Message.Raw != "" || client.Mode != MODE_MAIL {
		t.Error("Did not reset client correctly")
	}
}

// Returns a test client and the end of a connection pipe
func getTestClient() (*Client, *textproto.Conn) {
	sc, cc := net.Pipe()
	sconn, cconn := textproto.NewConn(sc), textproto.NewConn(cc)

	client := &Client{
		Mode:    MODE_HELO,
		Message: new(gomez.Message),
		text:    sconn,
		host: &MockSMTPServer{
			Query_: func(addr *mail.Address) gomez.QueryStatus {
				switch strings.Split(addr.Address, "@")[0] {
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
