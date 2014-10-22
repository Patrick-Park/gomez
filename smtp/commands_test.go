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

	"github.com/gbbr/gomez/mailbox"
)

func TestCmd_Modes_and_Codes(t *testing.T) {
	client, pipe := getTestClient()
	defer pipe.Close()

	testSuite := []struct {
		Fn        func(*Client, string) error
		Param     string
		ExpCode   int
		StartMode TransactionState
		ExpMode   TransactionState
		Hint      string
	}{
		{cmdHELO, "other name", 250, StateHELO, StateMAIL, "HELO"},
		{cmdHELO, "", 501, StateHELO, StateHELO, "HELO"},
		{cmdEHLO, "name", 250, StateHELO, StateMAIL, "EHLO"},
		{cmdEHLO, "", 501, StateHELO, StateHELO, "EHLO"},

		{cmdMAIL, "FROM:<asd>", 503, StateHELO, StateHELO, "MAIL"},
		{cmdMAIL, "FROM:<asd>", 503, StateRCPT, StateRCPT, "MAIL"},
		{cmdMAIL, "bad syntax", 501, StateMAIL, StateMAIL, "MAIL"},
		{cmdMAIL, "FROM:<bad_address>", 501, StateMAIL, StateMAIL, "MAIL"},

		{cmdRCPT, "", 503, StateHELO, StateHELO, "RCPT"},
		{cmdRCPT, "", 503, StateMAIL, StateMAIL, "RCPT"},
		{cmdRCPT, "invalid", 501, StateRCPT, StateRCPT, "RCPT"},
		{cmdRCPT, "TO:<guyhost.tld>", 501, StateRCPT, StateRCPT, "RCPT"},
		{cmdRCPT, "TO:Guy <not_found@host.tld>", 550, StateRCPT, StateRCPT, "RCPT"},
		{cmdRCPT, "TO:<not_local@host.tld>", 550, StateRCPT, StateRCPT, "RCPT"},
		{cmdRCPT, "TO:<success@host.tld>", 250, StateRCPT, StateDATA, "RCPT"},
		{cmdRCPT, "TO:<error@host.tld>", 451, StateRCPT, StateRCPT, "RCPT"},

		{cmdDATA, "", 503, StateHELO, StateHELO, "DATA"},
		{cmdDATA, "", 503, StateMAIL, StateMAIL, "DATA"},
		{cmdDATA, "", 503, StateRCPT, StateRCPT, "DATA"},

		{cmdRSET, "", 250, StateHELO, StateHELO, "RSET"},
		{cmdRSET, "", 250, StateRCPT, StateMAIL, "RSET"},
		{cmdRSET, "", 250, StateDATA, StateMAIL, "RSET"},

		{cmdNOOP, "", 250, StateRCPT, StateRCPT, "NOOP"},
		{cmdNOOP, "", 250, StateHELO, StateHELO, "NOOP"},

		{cmdQUIT, "", 221, StateHELO, StateHELO, "QUIT"},
		{cmdQUIT, "", 221, StateMAIL, StateMAIL, "QUIT"},
		{cmdQUIT, "", 221, StateRCPT, StateRCPT, "QUIT"},
		{cmdQUIT, "", 221, StateDATA, StateDATA, "QUIT"},

		{cmdVRFY, "", 252, StateDATA, StateDATA, "VRFY"},
	}

	go func() {
		for _, test := range testSuite {
			client.Mode = test.StartMode
			test.Fn(client, test.Param)
		}
	}()

	for _, test := range testSuite {
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
	if err != nil || client.Mode != StateMAIL || client.ID != "other name" {
		t.Errorf("Expected 250, Mode 0 and ID 'name' but got %+v, %d, %s", err, client.Mode, client.ID)
	}

	pipe.Close()
}

func TestCmdEHLO_Param_Passing(t *testing.T) {
	client, pipe := getTestClient()

	go cmdEHLO(client, "name")
	_, _, err := pipe.ReadResponse(250)
	if err != nil || client.Mode != StateMAIL || client.ID != "name" {
		t.Errorf("Expected 250, Mode 0 and ID 'name' but got %+v, %d, %s", err, client.Mode, client.ID)
	}

	pipe.Close()
}

func TestCmdMAIL_Param_Passing(t *testing.T) {
	client, pipe := getTestClient()
	client.Mode = StateMAIL

	testAddr := mail.Address{"First Last", "asd@box.com"}

	go cmdMAIL(client, "from:First Last <asd@box.com>")
	_, _, err := pipe.ReadResponse(250)
	if err != nil || client.Mode != StateRCPT || testAddr.String() != client.Message.From().String() {
		t.Errorf("Expected code 250, address %s got: %#v, %s", testAddr, err, client.Message.From())
	}

	pipe.Close()
}

func TestCmdRCPT_User_Not_Local(t *testing.T) {
	client, pipe := getTestClient()

	client.Mode = StateRCPT

	// Relay is enabled
	client.host = &MockSMTPServer{
		Query_:    func(addr *mail.Address) mailbox.QueryResult { return mailbox.QueryNotLocal },
		Settings_: func() Config { return Config{Relay: true} },
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		cmdRCPT(client, "TO:<not_local@host.tld>")
		wg.Done()
	}()

	_, _, err := pipe.ReadResponse(251)
	if err != nil || client.Mode != StateDATA || client.Message.Rcpt()[0].Address != "not_local@host.tld" ||
		client.Message.Outbound()[0].Address != "not_local@host.tld" || len(client.Message.Inbound()) != 0 {
		t.Errorf("Expected to get a 251 response and a recipient, got: %s and '%s'", err, client.Message.Rcpt()[0])
	}

	// Relay is disabled
	client.host = &MockSMTPServer{
		Query_:    func(addr *mail.Address) mailbox.QueryResult { return mailbox.QueryNotLocal },
		Settings_: func() Config { return Config{Relay: false} },
	}

	wg.Wait()

	go cmdRCPT(client, "TO:<not_local@host.tld>")
	_, _, err = pipe.ReadResponse(550)
	if err != nil || client.Mode != StateDATA || len(client.Message.Rcpt()) != 1 {
		t.Errorf("Expected to get a 550 response and no recipient, got: %+v", err)
	}

	pipe.Close()
}

func TestCmdRCPT_Param_Passing(t *testing.T) {
	client, pipe := getTestClient()
	client.Mode = StateRCPT

	go cmdRCPT(client, "TO:<success@host.tld>")
	_, _, err := pipe.ReadResponse(250)
	if err != nil || client.Mode != StateDATA || client.Message.Rcpt()[0].Address != "success@host.tld" ||
		client.Message.Inbound()[0].Address != "success@host.tld" || len(client.Message.Outbound()) != 0 {
		t.Errorf("Expected to get a 250 response and a recipient, got: %s and '%s'", err, client.Message.Rcpt()[0])
	}

	pipe.Close()
}

func TestCmdRCPT_Internal_Error(t *testing.T) {
	client, pipe := getTestClient()
	client.Mode = StateRCPT

	log.SetOutput(ioutil.Discard)
	go cmdRCPT(client, "TO:<error@host.tld>")

	_, _, err := pipe.ReadResponse(451)
	if err != nil || client.Mode != StateRCPT || len(client.Message.Rcpt()) != 0 {
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
		client.Mode = StateDATA
		client.ID = "Test_ID"
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
	client.Mode = StateDATA

	defer pipe.Close()
	go cmdDATA(client, "these params are ignored")
}

func TestCmdDATA_Error_ReadLines(t *testing.T) {
	client, pipe := getTestClient()
	defer pipe.Close()

	client.Mode = StateDATA
	go cmdDATA(client, "these params are ignored")

	pipe.ReadResponse(354)

	pipe.PrintfLine("Line 1 of text")
	pipe.PrintfLine("Line 2 of text")
}

func TestCmdRSET(t *testing.T) {
	client, pipe := getTestClient()
	defer pipe.Close()

	client.Mode = StateDATA
	client.Message.Raw = "ABCD"
	client.ID = "Jonah"

	go cmdRSET(client, "")
	_, _, err := pipe.ReadResponse(250)
	if err != nil || client.Message.Raw != "" || client.Mode != StateMAIL {
		t.Error("Did not reset client correctly")
	}
}

// Returns a test client and the end of a connection pipe
func getTestClient() (*Client, *textproto.Conn) {
	sc, cc := net.Pipe()
	sconn, cconn := textproto.NewConn(sc), textproto.NewConn(cc)

	client := &Client{
		Mode:    StateHELO,
		Message: new(mailbox.Message),
		text:    sconn,
		host: &MockSMTPServer{
			Query_: func(addr *mail.Address) mailbox.QueryResult {
				switch strings.Split(addr.Address, "@")[0] {
				case "not_found":
					return mailbox.QueryNotFound
				case "not_local":
					return mailbox.QueryNotLocal
				case "success":
					return mailbox.QuerySuccess
				}

				return mailbox.QueryError
			},
			Settings_: func() Config { return Config{} },
			Digest_:   func(c *Client) error { return nil },
		},
	}

	return client, cconn
}
