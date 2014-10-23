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
	"github.com/gbbr/jamon"
)

func TestCmd_Modes_and_Codes(t *testing.T) {
	client, pipe := getTestClient()
	defer pipe.Close()

	testSuite := []struct {
		Fn        func(*transaction, string) error
		Param     string
		ExpCode   int
		StartMode transactionState
		ExpMode   transactionState
		Hint      string
	}{
		{cmdHELO, "other name", 250, stateHELO, stateMAIL, "HELO"},
		{cmdHELO, "", 501, stateHELO, stateHELO, "HELO"},
		{cmdEHLO, "name", 250, stateHELO, stateMAIL, "EHLO"},
		{cmdEHLO, "", 501, stateHELO, stateHELO, "EHLO"},

		{cmdMAIL, "FROM:<asd>", 503, stateHELO, stateHELO, "MAIL"},
		{cmdMAIL, "FROM:<asd>", 503, stateRCPT, stateRCPT, "MAIL"},
		{cmdMAIL, "bad syntax", 501, stateMAIL, stateMAIL, "MAIL"},
		{cmdMAIL, "FROM:<bad_address>", 501, stateMAIL, stateMAIL, "MAIL"},

		{cmdRCPT, "", 503, stateHELO, stateHELO, "RCPT"},
		{cmdRCPT, "", 503, stateMAIL, stateMAIL, "RCPT"},
		{cmdRCPT, "invalid", 501, stateRCPT, stateRCPT, "RCPT"},
		{cmdRCPT, "TO:<guyhost.tld>", 501, stateRCPT, stateRCPT, "RCPT"},
		{cmdRCPT, "TO:Guy <not_found@host.tld>", 550, stateRCPT, stateRCPT, "RCPT"},
		{cmdRCPT, "TO:<not_local@host.tld>", 550, stateRCPT, stateRCPT, "RCPT"},
		{cmdRCPT, "TO:<success@host.tld>", 250, stateRCPT, stateDATA, "RCPT"},
		{cmdRCPT, "TO:<error@host.tld>", 451, stateRCPT, stateRCPT, "RCPT"},

		{cmdDATA, "", 503, stateHELO, stateHELO, "DATA"},
		{cmdDATA, "", 503, stateMAIL, stateMAIL, "DATA"},
		{cmdDATA, "", 503, stateRCPT, stateRCPT, "DATA"},

		{cmdRSET, "", 250, stateHELO, stateHELO, "RSET"},
		{cmdRSET, "", 250, stateRCPT, stateMAIL, "RSET"},
		{cmdRSET, "", 250, stateDATA, stateMAIL, "RSET"},

		{cmdNOOP, "", 250, stateRCPT, stateRCPT, "NOOP"},
		{cmdNOOP, "", 250, stateHELO, stateHELO, "NOOP"},

		{cmdQUIT, "", 221, stateHELO, stateHELO, "QUIT"},
		{cmdQUIT, "", 221, stateMAIL, stateMAIL, "QUIT"},
		{cmdQUIT, "", 221, stateRCPT, stateRCPT, "QUIT"},
		{cmdQUIT, "", 221, stateDATA, stateDATA, "QUIT"},

		{cmdVRFY, "", 252, stateDATA, stateDATA, "VRFY"},
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
	if err != nil || client.Mode != stateMAIL || client.ID != "other name" {
		t.Errorf("Expected 250, Mode 0 and ID 'name' but got %+v, %d, %s", err, client.Mode, client.ID)
	}

	pipe.Close()
}

func TestCmdEHLO_Param_Passing(t *testing.T) {
	client, pipe := getTestClient()

	go cmdEHLO(client, "name")
	_, _, err := pipe.ReadResponse(250)
	if err != nil || client.Mode != stateMAIL || client.ID != "name" {
		t.Errorf("Expected 250, Mode 0 and ID 'name' but got %+v, %d, %s", err, client.Mode, client.ID)
	}

	pipe.Close()
}

func TestCmdMAIL_Param_Passing(t *testing.T) {
	client, pipe := getTestClient()
	client.Mode = stateMAIL

	testAddr := mail.Address{"First Last", "asd@box.com"}

	go cmdMAIL(client, "from:First Last <asd@box.com>")
	_, _, err := pipe.ReadResponse(250)
	if err != nil || client.Mode != stateRCPT || testAddr.String() != client.Message.From().String() {
		t.Errorf("Expected code 250, address %s got: %#v, %s", testAddr, err, client.Message.From())
	}

	pipe.Close()
}

func TestCmdRCPT_User_Not_Local(t *testing.T) {
	client, pipe := getTestClient()

	client.Mode = stateRCPT

	// Relay is enabled
	client.host = &mockHost{
		QueryMock:    func(addr *mail.Address) mailbox.QueryResult { return mailbox.QueryNotLocal },
		SettingsMock: func() jamon.Group { return jamon.Group{"relay": "true"} },
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		cmdRCPT(client, "TO:<not_local@host.tld>")
		wg.Done()
	}()

	_, _, err := pipe.ReadResponse(251)
	if err != nil || client.Mode != stateDATA || client.Message.Rcpt()[0].Address != "not_local@host.tld" ||
		client.Message.Outbound()[0].Address != "not_local@host.tld" || len(client.Message.Inbound()) != 0 {
		t.Errorf("Expected to get a 251 response and a recipient, got: %s and '%s'", err, client.Message.Rcpt()[0])
	}

	// Relay is disabled
	client.host = &mockHost{
		QueryMock:    func(addr *mail.Address) mailbox.QueryResult { return mailbox.QueryNotLocal },
		SettingsMock: func() jamon.Group { return jamon.Group{} },
	}

	wg.Wait()

	go cmdRCPT(client, "TO:<not_local@host.tld>")
	_, _, err = pipe.ReadResponse(550)
	if err != nil || client.Mode != stateDATA || len(client.Message.Rcpt()) != 1 {
		t.Errorf("Expected to get a 550 response and no recipient, got: %+v", err)
	}

	pipe.Close()
}

func TestCmdRCPT_Param_Passing(t *testing.T) {
	client, pipe := getTestClient()
	client.Mode = stateRCPT

	go cmdRCPT(client, "TO:<success@host.tld>")
	_, _, err := pipe.ReadResponse(250)
	if err != nil || client.Mode != stateDATA || client.Message.Rcpt()[0].Address != "success@host.tld" ||
		client.Message.Inbound()[0].Address != "success@host.tld" || len(client.Message.Outbound()) != 0 {
		t.Errorf("Expected to get a 250 response and a recipient, got: %s and '%s'", err, client.Message.Rcpt()[0])
	}

	pipe.Close()
}

func TestCmdRCPT_Internal_Error(t *testing.T) {
	client, pipe := getTestClient()
	client.Mode = stateRCPT

	log.SetOutput(ioutil.Discard)
	go cmdRCPT(client, "TO:<error@host.tld>")

	_, _, err := pipe.ReadResponse(451)
	if err != nil || client.Mode != stateRCPT || len(client.Message.Rcpt()) != 0 {
		t.Errorf("Expected to get a 451 response and a recipient, got: %s and '%s'", err, client.Message.Rcpt()[0])
	}

	pipe.Close()
}

func TestCmdDATA_Digest(t *testing.T) {
	calledDigest := false

	client, pipe := getTestClient()
	client.host = &mockHost{
		DigestMock: func(c *transaction) error {
			calledDigest = true
			return nil
		},
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		client.Mode = stateDATA
		client.ID = "Test_ID"
		cmdDATA(client, "")
		wg.Done()
	}()

	pipe.ReadResponse(354)
	pipe.PrintfLine(".")

	wg.Wait()
	pipe.Close()

	if !calledDigest {
		t.Error("Did not call digest")
	}
}

func TestCmdDATA_Error_Notify(t *testing.T) {
	client, pipe := getTestClient()
	client.Mode = stateDATA

	defer pipe.Close()
	go cmdDATA(client, "these params are ignored")
}

func TestCmdDATA_Error_ReadLines(t *testing.T) {
	client, pipe := getTestClient()
	defer pipe.Close()

	client.Mode = stateDATA
	go cmdDATA(client, "these params are ignored")

	pipe.ReadResponse(354)

	pipe.PrintfLine("Line 1 of text")
	pipe.PrintfLine("Line 2 of text")
}

func TestCmdRSET(t *testing.T) {
	client, pipe := getTestClient()
	defer pipe.Close()

	client.Mode = stateDATA
	client.Message.Raw = "ABCD"
	client.ID = "Jonah"

	go cmdRSET(client, "")
	_, _, err := pipe.ReadResponse(250)
	if err != nil || client.Message.Raw != "" || client.Mode != stateMAIL {
		t.Error("Did not reset client correctly")
	}
}

// Returns a test client and the end of a connection pipe
func getTestClient() (*transaction, *textproto.Conn) {
	sc, cc := net.Pipe()
	sconn, cconn := textproto.NewConn(sc), textproto.NewConn(cc)

	client := &transaction{
		Mode:    stateHELO,
		Message: new(mailbox.Message),
		text:    sconn,
		host: &mockHost{
			QueryMock: func(addr *mail.Address) mailbox.QueryResult {
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
			SettingsMock: func() jamon.Group { return jamon.Group{} },
			DigestMock:   func(c *transaction) error { return nil },
		},
	}

	return client, cconn
}
