package smtp

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gbbr/gomez/mailbox"
	"github.com/gbbr/mocks"
)

// Should correctly run commands from spec, echo parameters
// and reject bad commands or commands that are not in the spec
func TestServerRun(t *testing.T) {
	srv := &server{
		spec: &commandSpec{
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

// Should create a client that picks up commands
// and greet the connection with 220 status
func TestServerCreateClient(t *testing.T) {
	var wg sync.WaitGroup

	cc, sc := net.Pipe()
	cconn := textproto.NewConn(cc)

	testServer := &server{
		spec: &commandSpec{
			"EXIT": func(ctx *transaction, params string) error {
				return io.EOF
			},
		},
		config: Config{Hostname: "mecca.local"},
	}

	wg.Add(1)

	go func() {
		testServer.createClient(sc)
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
	testServer := &server{
		config: Config{Hostname: "test", Relay: true},
	}

	flags := testServer.settings()
	if flags.Hostname != "test" || !flags.Relay {
		t.Error("Did not retrieve correct flags via settings()")
	}
}

func TestServer_Query_Calls_MailBox(t *testing.T) {
	queryCalled := false
	testServer := &server{
		Enqueuer: &mailbox.MockEnqueuer{
			Query_: func(addr *mail.Address) mailbox.QueryResult {
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
	server := server{config: Config{Hostname: "TestHost"}}

	testSuite := []struct {
		Message  *mailbox.Message
		Enqueuer mailbox.Enqueuer
		Address  string
		Response int
	}{
		{
			&mailbox.Message{Raw: "Message is not valid"},
			mailbox.MockEnqueuer{},
			"1.2.3.4:1234", 550,
		}, {
			&mailbox.Message{Raw: "Subject: Heloo\r\nFrom: Maynard\r\n\r\nMessage is not valid"},
			mailbox.MockEnqueuer{},
			"1.2.3.4:1234", 550,
		}, {
			&mailbox.Message{Raw: "From: Mary\r\n\r\nMessage is not valid"},
			mailbox.MockEnqueuer{},
			"1.2.3.4:1234", 550,
		}, {
			&mailbox.Message{
				Raw: "From: Mary\r\nDate: Today\r\n\r\nMessage is valid, with DB error.",
			},
			mailbox.MockEnqueuer{
				NextID_: func() (uint64, error) { return 0, errors.New("Error connecting to DB") },
			},
			"1.2.3.4:1234", 451,
		}, {
			&mailbox.Message{
				Raw: "From: Mary\r\nDate: Today\r\n\r\nMessage is valid, with queuing error.",
			},
			mailbox.MockEnqueuer{
				NextID_:  func() (uint64, error) { return 123, nil },
				Enqueue_: func(*mailbox.Message) error { return errors.New("Error queueing message.") },
			},
			"1.2.3.4:1234", 451,
		}, {
			&mailbox.Message{
				Raw: "From: Mary\r\nDate: Today\r\n\r\nMessage is valid, with split-host error.",
			},
			mailbox.MockEnqueuer{
				NextID_:  func() (uint64, error) { return 123, nil },
				Enqueue_: func(*mailbox.Message) error { return errors.New("Error queueing message.") },
			},
			"1.2.3.4", 451,
		}, {
			&mailbox.Message{
				Raw: "From: Mary\r\nDate: Today\r\n\r\nMessage is valid, with no errors.",
			},
			mailbox.MockEnqueuer{
				NextID_:  func() (uint64, error) { return 123, nil },
				Enqueue_: func(*mailbox.Message) error { return nil },
			},
			"1.2.3.4:123", 250,
		},
	}

	var wg sync.WaitGroup

	for _, test := range testSuite {
		server.Enqueuer = test.Enqueuer
		client, pipe := getTestClient()
		client.conn = &mocks.Conn{RemoteAddress: test.Address}
		client.Message = test.Message
		client.Message.AddInbound(&mail.Address{"Name", "Addr@es"})

		wg.Add(1)
		go func() {
			server.digest(client)
			wg.Done()
		}()

		_, _, err := pipe.ReadResponse(test.Response)
		if err != nil {
			t.Errorf("Expected %d, but got: %+v", test.Response, err)
		}

		pipe.Close()
		wg.Wait()
	}
}

func TestServer_Digest_Header_Message_ID(t *testing.T) {
	var called bool

	server := server{config: Config{Hostname: "TestHost"}}
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
			&mailbox.MockEnqueuer{NextID_: func() (uint64, error) { called = true; return 1, nil }},
			".1@TestHost>", 1, 451, true,
		}, {
			&mailbox.Message{Raw: "From: Mary\r\nMessage-ID: My_ID\r\nDate: Today\r\n\r\nHey Mary how are you?"},
			&mailbox.MockEnqueuer{NextID_: func() (uint64, error) { called = true; return 2, nil }},
			"My_ID", 2, 451, true,
		}, {
			&mailbox.Message{Raw: "From: Mary\r\nMessage-ID: My_ID\r\nDate: Today\r\n\r\nHey Mary how are you?", ID: 53},
			&mailbox.MockEnqueuer{NextID_: func() (uint64, error) { called = true; return 2, nil }},
			"My_ID", 53, 451, false,
		},
	}

	var wg sync.WaitGroup

	for _, test := range testSuite {
		called = false
		server.Enqueuer = test.Enqueuer
		client, pipe := getTestClient()
		client.conn = &mocks.Conn{RemoteAddress: "invalid_addr"}
		client.Message = test.Message

		wg.Add(1)
		go func() {
			server.digest(client)
			wg.Done()
		}()

		pipe.ReadResponse(test.Response)
		pipe.Close()

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

		wg.Wait()
	}
}

func TestServer_Digest_Received_Header(t *testing.T) {
	var called bool

	server := server{config: Config{Hostname: "TestHost"}}
	server.Enqueuer = mailbox.MockEnqueuer{
		NextID_:  func() (uint64, error) { called = true; return 2, nil },
		Enqueue_: func(*mailbox.Message) error { return errors.New("Error processing") },
	}

	client, pipe := getTestClient()
	client.conn = &mocks.Conn{RemoteAddress: "1.2.3.4:567"}
	client.ID = "Doe"

	client.Message = &mailbox.Message{
		Raw: "From: Mary\r\nMessage-ID: My_ID\r\nDate: Today\r\n\r\nHey Mary how are you?",
		ID:  53,
	}

	client.Message.AddOutbound(&mail.Address{"Name", "Addr@es"})

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		server.digest(client)
		wg.Done()
	}()

	pipe.ReadResponse(451)
	wg.Wait()

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

	// Test that reverse lookup is applied in header with known remote address
	// This might fail in the future if the PTR entry for gmail.com changes
	client.conn = &mocks.Conn{RemoteAddress: "127.0.0.1:1234"}

	wg.Add(1)
	go func() {
		server.digest(client)
		wg.Done()
	}()

	pipe.ReadResponse(451)
	pipe.Close()

	wg.Wait()

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
	Start(&mailbox.MockEnqueuer{}, Config{ListenAddr: "bad_addr"})
}

func TestServer_SMTP_Sending(t *testing.T) {
	var err error

	go func() {
		err = Start(&mailbox.MockEnqueuer{
			Enqueue_: func(msg *mailbox.Message) error {
				m, err := msg.Parse()
				if err != nil {
					t.Error("Failed to parse queued message")
				}

				if len(m.Header["Message-Id"]) == 0 {
					t.Error("Queued message with no ID")
				}

				if !strings.HasSuffix(m.Header["Message-Id"][0], ".555@TestHost>") {
					t.Errorf("Got wrong Message-ID: %s", m.Header["Message-Id"][0])
				}

				buf := make([]byte, 22) // Exact length of "This is the email body"
				m.Body.Read(buf)

				if string(buf) != "This is the email body" {
					t.Errorf("Got wrong email body: '%s'", string(buf))
				}

				return nil
			},
			Query_: func(addr *mail.Address) mailbox.QueryResult {
				return mailbox.QuerySuccess
			},
			NextID_: func() (uint64, error) { return 555, nil },
		}, Config{ListenAddr: "127.0.0.1:1234", Hostname: "TestHost"})
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
		t.Error(err)
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
