package smtp

import (
	"errors"
	"fmt"
	"io"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gbbr/gomez/mailbox"
	"github.com/gbbr/jamon"
	"github.com/gbbr/mocks"
)

// Should correctly run commands from spec, echo parameters
// and reject bad commands or commands that are not in the spec
func TestServerRun(t *testing.T) {
	srv := server{
		spec: commandSpec{
			"HELO": func(ctx *transaction, params string) error {
				return ctx.notify(reply{100, params})
			},
		},
	}

	testClient, pipe := getTestClient()
	testClient.Mode = stateHELO
	defer pipe.Close()

	testCases := []struct {
		Message string
		reply   string
	}{
		{"BADFORMAT", replyBadCommand.String()},
		{"GOOD FORMAT", replyBadCommand.String()},
		{"HELO  world ", "100 world"},
		{"HELO  world how are you?", "100 world how are you?"},
		{"HELO", "100 "},
	}

	var wg sync.WaitGroup

	for _, test := range testCases {
		wg.Add(1)

		go func() {
			srv.run(testClient, test.Message)
			wg.Done()
		}()

		rpl, err := pipe.ReadLine()
		if err != nil {
			t.Errorf("Error reading response %s", err)
		}

		if rpl != test.reply {
			t.Errorf("Expected '%s' but got '%s'", test.reply, rpl)
		}

		wg.Wait()
	}
}

func TestServerCreateClient_Crash(t *testing.T) {
	testServer := new(server)
	testServer.createTransaction(&mocks.Conn{RAddr: "bogus"})
}

// Should create a client that picks up commands
// and greet the connection with 220 status
func TestServerCreateClient(t *testing.T) {
	var wg sync.WaitGroup

	cc, sc := mocks.Pipe(
		&mocks.Conn{RAddr: "1.1.1.1:123"},
		&mocks.Conn{RAddr: "1.1.1.1:123"},
	)

	cconn := textproto.NewConn(cc)

	testServer := server{
		spec: commandSpec{
			"EXIT": func(ctx *transaction, params string) error {
				return io.EOF
			},
		},
		config: jamon.Group{"host": "mecca.local"},
	}

	wg.Add(1)

	go func() {
		testServer.createTransaction(sc)
		wg.Done()
	}()

	_, _, err := cconn.ReadResponse(220)
	if err != nil {
		t.Errorf("Expected code 220 but got %+v", err)
	}

	cconn.PrintfLine("EXIT")
	wg.Wait()
}

func TestServer_Settings(t *testing.T) {
	testServer := server{
		config: jamon.Group{"host": "test", "relay": "true"},
	}

	flags := testServer.settings()
	if flags.Get("host") != "test" || !flags.Has("relay") {
		t.Error("Did not retrieve correct flags via settings()")
	}
}

func TestServer_Query_Calls_MailBox(t *testing.T) {
	queryCalled := false
	testServer := server{
		Enqueuer: mailbox.MockEnqueuer{
			QueryMock: func(addr *mail.Address) int {
				queryCalled = true
				return mailbox.QuerySuccess
			},
		},
	}

	testServer.query(&mail.Address{})
	if !queryCalled {
		t.Error("server query did not call Enqueuer query")
	}
}

func TestServer_Digest_Responses(t *testing.T) {
	server := server{config: jamon.Group{"hostname": "TestHost"}}

	testSuite := []struct {
		Message  *mailbox.Message
		Enqueuer mailbox.Enqueuer
		Response error
	}{
		{
			&mailbox.Message{Raw: "Message is not valid"},
			mailbox.MockEnqueuer{},
			errMsgNotCompliant,
		}, {
			&mailbox.Message{Raw: "Subject: Heloo\r\nFrom: Maynard\r\n\r\nMessage is not valid"},
			mailbox.MockEnqueuer{},
			errMsgNotCompliant,
		}, {
			&mailbox.Message{Raw: "From: Mary\r\n\r\nMessage is not valid"},
			mailbox.MockEnqueuer{},
			errMsgNotCompliant,
		}, {
			&mailbox.Message{
				Raw: "From: Mary\r\nDate: Today\r\n\r\nMessage is valid, with DB error.",
			},
			mailbox.MockEnqueuer{
				GUIDMock: func() (uint64, error) { return 0, errors.New("Error connecting to DB") }},
			errProcessing,
		}, {
			&mailbox.Message{
				Raw: "From: Mary\r\nDate: Today\r\n\r\nMessage is valid, with queuing error."},
			mailbox.MockEnqueuer{
				GUIDMock:    func() (uint64, error) { return 123, nil },
				EnqueueMock: func(*mailbox.Message) error { return errors.New("Error queueing message.") }},
			errEnqueuing,
		}, {
			&mailbox.Message{
				Raw: "From: Mary\r\nDate: Today\r\n\r\nMessage is valid, with no errors."},
			mailbox.MockEnqueuer{
				GUIDMock:    func() (uint64, error) { return 123, nil },
				EnqueueMock: func(*mailbox.Message) error { return nil }},
			nil,
		},
	}
	for _, test := range testSuite {
		server.Enqueuer = test.Enqueuer
		client, _ := getTestClient()
		client.Message = test.Message
		client.Message.AddInbound(&mail.Address{"Name", "Addr@es"})

		err := server.digest(client)
		if err != test.Response {
			t.Errorf("expected %+v, got %+v", test.Response, err)
		}
	}
}

func TestServer_Digest_Header_Message_ID(t *testing.T) {
	var called bool
	server := server{config: jamon.Group{"host": "TestHost"}}
	testSuite := []struct {
		Message    *mailbox.Message
		Enqueuer   mailbox.Enqueuer
		Value      string
		ID         uint64
		Response   int
		ShouldCall bool
	}{
		{
			&mailbox.Message{Raw: "From: Mary\r\nDate: Today\r\n\r\nHey Mary how are you?"},
			&mailbox.MockEnqueuer{
				GUIDMock:    func() (uint64, error) { called = true; return 1, nil },
				EnqueueMock: func(m *mailbox.Message) error { return errors.New("error") },
			},
			".1@TestHost>", 1, 451, true,
		}, {
			&mailbox.Message{Raw: "From: Mary\r\nMessage-ID: My_ID\r\nDate: Today\r\n\r\nHey Mary how are you?"},
			&mailbox.MockEnqueuer{
				GUIDMock:    func() (uint64, error) { called = true; return 2, nil },
				EnqueueMock: func(m *mailbox.Message) error { return errors.New("error") },
			},
			"My_ID", 2, 451, true,
		}, {
			&mailbox.Message{Raw: "From: Mary\r\nMessage-ID: My_ID\r\nDate: Today\r\n\r\nHey Mary how are you?", ID: 53},
			&mailbox.MockEnqueuer{
				GUIDMock:    func() (uint64, error) { called = true; return 1, nil },
				EnqueueMock: func(m *mailbox.Message) error { return errors.New("error") },
			},
			"My_ID", 53, 451, false,
		},
	}
	for _, test := range testSuite {
		called = false
		server.Enqueuer = test.Enqueuer
		client, _ := getTestClient()
		client.Message = test.Message
		client.Message.AddInbound(&mail.Address{"", "a@b.com"})

		server.digest(client)

		msg, err := client.Message.Parse()
		if err != nil || len(msg.Header["Message-Id"]) == 0 {
			t.Error("Digested and could not parse or Message-ID is missing.")
		}

		if !strings.HasSuffix(msg.Header["Message-Id"][0], test.Value) {
			t.Errorf("Did not set Message-ID header correctly, got: '%s'", msg.Header["Message-Id"][0])
		}

		if test.ShouldCall && !called {
			t.Error("Did not call mailbox for ID")
		}

		if client.Message.ID != test.ID {
			t.Errorf("Did not set message ID. Wanted %d, got %d", test.ID, client.Message.ID)
		}
	}
}

func TestServer_Digest_Received_Header(t *testing.T) {
	var called bool

	server := server{config: jamon.Group{"host": "TestHost"}}
	server.Enqueuer = mailbox.MockEnqueuer{
		GUIDMock:    func() (uint64, error) { called = true; return 2, nil },
		EnqueueMock: func(*mailbox.Message) error { return errors.New("Error processing") },
	}

	client, _ := getTestClient()
	client.addrIP = "1.2.3.4"
	client.ID = "Doe"
	client.Message = &mailbox.Message{
		Raw: "From: Mary\r\nMessage-ID: My_ID\r\nDate: Today\r\n\r\nHey Mary how are you?",
		ID:  53,
	}
	client.Message.AddOutbound(&mail.Address{"Name", "Addr@es"})

	server.digest(client)
	msg, err := client.Message.Parse()
	if err != nil || len(msg.Header["Received"]) == 0 {
		t.Error("Error parsing message")
	}

	if !strings.HasPrefix(msg.Header["Received"][0],
		`from Doe ([1.2.3.4]) by TestHost (Gomez) with ESMTP id 53 for "Name" <Addr@es>;`) {
		t.Errorf("Got: %s", msg.Header["Received"][0])
	}

	if testing.Short() {
		t.Log("Short version skips network")
		return
	}

	// Check that host is added
	client.addrIP = "127.0.0.1"
	client.addrHost = "localhost "
	server.digest(client)

	msg, err = client.Message.Parse()
	if err != nil || len(msg.Header["Received"]) == 0 {
		t.Error("Error parsing message")
	}
	if !strings.HasPrefix(msg.Header["Received"][0],
		`from Doe (localhost [127.0.0.1]) by TestHost (Gomez) with ESMTP id 53 for "Name" <Addr@es>;`) {
		t.Errorf("Got (on reverse): %s", msg.Header["Received"][0])
	}
}

func TestServer_Start_Error(t *testing.T) {
	if Start(&mailbox.MockEnqueuer{}, jamon.Group{"host": "wha", "listen": "bad_addr"}) == nil {
		t.Error("Expected error")
	}

	if Start(&mailbox.MockEnqueuer{}, jamon.Group{"listen": "bad_addr"}) != ErrMinConfig {
		t.Error("ErrMinConfig not returned")
	}
}

func TestServer_SMTP_Sending(t *testing.T) {
	var err error

	go func() {
		err = Start(&mailbox.MockEnqueuer{
			EnqueueMock: func(msg *mailbox.Message) error {
				buf := make([]byte, 22) // Exact length of "This is the email body"

				m, err := msg.Parse()
				m.Body.Read(buf)

				switch {
				case err != nil:
					t.Error("Failed to parse queued message")

				case len(m.Header["Message-Id"]) == 0:
					t.Error("Queued message with no ID")

				case !strings.HasSuffix(m.Header["Message-Id"][0], ".555@TestHost>"):
					t.Errorf("Got wrong Message-ID: %s", m.Header["Message-Id"][0])

				case msg.From().String() != "<sender@example.org>":
					t.Errorf("Was expecting sender@example.org, but got '%s'.", msg.From())

				case msg.Inbound()[0].String() != "<recipient@example.net>":
					t.Errorf("Expected <recipient@example.net>, got '%s'.", msg.Inbound()[0])

				case string(buf) != "This is the email body":
					t.Errorf("Got wrong email body: '%s'", string(buf))
				}

				return nil
			},
			QueryMock: func(addr *mail.Address) int {
				return mailbox.QuerySuccess
			},
			GUIDMock: func() (uint64, error) { return 555, nil },
		}, jamon.Group{"listen": ":1234", "host": "TestHost"})
	}()

	if err != nil {
		t.Logf("Did not run SMTP test. Failed to start server: %s", err)
		return
	}

	firstTry := time.Now()
	c, err := smtp.Dial("127.0.0.1:1234")

	// If connection fails initially, give server a chance to start
	// and try reconnecting on a 1 second time-out
	if err != nil {
		for {
			c, err = smtp.Dial("127.0.0.1:1234")
			if err == nil || time.Since(firstTry).Seconds() > 1 {
				break
			}
		}
	}

	if err != nil {
		t.Fatal(err)
	}

	// Set the sender and recipient first
	if err := c.Mail("sender@example.org"); err != nil {
		t.Error(err)
	}
	if err := c.Rcpt("recipient@example.net"); err != nil {
		t.Error(err)
	}

	// Send the email body.
	wc, err := c.Data()
	if err != nil {
		t.Error(err)
	}
	_, err = fmt.Fprintf(wc, "From: Me\r\nDate: Today\r\n\r\nThis is the email body")
	if err != nil {
		t.Error(err)
	}
	err = wc.Close()
	if err != nil {
		t.Error(err)
	}

	// Send the QUIT command and close the connection.
	err = c.Quit()
	if err != nil {
		t.Error(err)
	}
}
