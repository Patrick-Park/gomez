package smtp

import (
	"errors"
	"flag"
	"net"
	"net/mail"
	"net/textproto"
	"strings"
	"sync"
	"testing"

	"github.com/gbbr/gomez"
	"github.com/gbbr/mocks"
)

var networkEnabled = flag.Bool("network", false, "Signifies if network is enabled or not")

// Should correctly run commands from spec, echo parameters
// and reject bad commands or commands that are not in the spec
func TestServerRun(t *testing.T) {
	srv := &Server{
		spec: &CommandSpec{
			"HELO": func(ctx *Client, params string) error {
				return ctx.Notify(Reply{100, params})
			},
		},
	}

	cc, sc := net.Pipe()
	cconn, sconn := textproto.NewConn(cc), textproto.NewConn(sc)

	testClient := &Client{
		Mode: MODE_HELO,
		text: sconn,
	}

	testCases := []struct {
		Message string
		Reply   string
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
			srv.Run(testClient, test.Message)
			wg.Done()
		}()

		rpl, err := cconn.ReadLine()
		if err != nil {
			t.Errorf("Error reading response %s", err)
		}

		if rpl != test.Reply {
			t.Errorf("Expected '%s' but got '%s'", test.Reply, rpl)
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

	testServer := &Server{
		spec: &CommandSpec{
			"EXIT": func(ctx *Client, params string) error {
				ctx.Mode = MODE_QUIT
				return nil
			},
		},
		config: Config{Hostname: "mecca.local"},
	}

	wg.Add(1)

	go func() {
		testServer.CreateClient(sc)
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
	testServer := &Server{
		config: Config{Hostname: "test", Relay: true},
	}

	flags := testServer.Settings()
	if flags.Hostname != "test" || !flags.Relay {
		t.Error("Did not retrieve correct flags via Settings()")
	}
}

func TestServer_Query_Calls_MailBox(t *testing.T) {
	queryCalled := false
	testServer := &Server{
		Mailbox: &gomez.MockMailbox{
			Query_: func(addr *mail.Address) gomez.QueryStatus {
				queryCalled = true
				return gomez.QUERY_STATUS_SUCCESS
			},
		},
	}

	testServer.Query(&mail.Address{})
	if !queryCalled {
		t.Error("Server Query did not call Mailbox query")
	}
}

func TestServer_Mock(t *testing.T) {
	var mock SMTPServer = &MockSMTPServer{}

	mock.Digest(&Client{})
	mock.Settings()
	mock.Query(&mail.Address{})
	mock.Run(&Client{}, "")
}

func TestServer_Digest_Responses(t *testing.T) {
	server := Server{config: Config{Hostname: "TestHost"}}

	testSuite := []struct {
		Message  *gomez.Message
		Mailbox  gomez.Mailbox
		Address  string
		Response int
	}{
		{
			&gomez.Message{Raw: "Message is not valid"},
			gomez.MockMailbox{},
			"1.2.3.4:1234", 550,
		}, {
			&gomez.Message{Raw: "Subject: Heloo\r\nFrom: Maynard\r\n\r\nMessage is not valid"},
			gomez.MockMailbox{},
			"1.2.3.4:1234", 550,
		}, {
			&gomez.Message{Raw: "From: Mary\r\n\r\nMessage is not valid"},
			gomez.MockMailbox{},
			"1.2.3.4:1234", 550,
		}, {
			&gomez.Message{
				Raw: "From: Mary\r\nDate: Today\r\n\r\nMessage is valid, with DB error.",
			},
			gomez.MockMailbox{
				NextID_: func() (uint64, error) { return 0, errors.New("Error connecting to DB") },
			},
			"1.2.3.4:1234", 451,
		}, {
			&gomez.Message{
				Raw: "From: Mary\r\nDate: Today\r\n\r\nMessage is valid, with queuing error.",
			},
			gomez.MockMailbox{
				NextID_: func() (uint64, error) { return 123, nil },
				Queue_:  func(*gomez.Message) error { return errors.New("Error queueing message.") },
			},
			"1.2.3.4:1234", 451,
		}, {
			&gomez.Message{
				Raw: "From: Mary\r\nDate: Today\r\n\r\nMessage is valid, with queuing error.",
			},
			gomez.MockMailbox{
				NextID_: func() (uint64, error) { return 123, nil },
				Queue_:  func(*gomez.Message) error { return errors.New("Error queueing message.") },
			},
			"1.2.3.4", 451,
		}, {
			&gomez.Message{
				Raw: "From: Mary\r\nDate: Today\r\n\r\nMessage is valid, with no errors.",
			},
			gomez.MockMailbox{
				NextID_: func() (uint64, error) { return 123, nil },
				Queue_:  func(*gomez.Message) error { return nil },
			},
			"1.2.3.4:123", 250,
		},
	}

	var wg sync.WaitGroup

	for _, test := range testSuite {
		server.Mailbox = test.Mailbox
		client, pipe := getTestClient()
		client.conn = &mocks.Conn{RemoteAddress: test.Address}
		client.Message = test.Message
		client.Message.AddRcpt(&mail.Address{"Name", "Addr@es"})

		wg.Add(1)
		go func() {
			server.Digest(client)
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

func TestServer_Digest_Header_Message_Id(t *testing.T) {
	var called bool

	server := Server{config: Config{Hostname: "TestHost"}}
	server.Mailbox = gomez.MockMailbox{
		NextID_: func() (uint64, error) { called = true; return 1, nil },
	}

	client, pipe := getTestClient()
	client.conn = &mocks.Conn{RemoteAddress: "invalid_addr"}
	client.Message = &gomez.Message{
		Raw: "From: Mary\r\nDate: Today\r\n\r\nHey Mary how are you?",
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		server.Digest(client)
		wg.Done()
	}()

	pipe.ReadResponse(451)
	pipe.Close()

	msg, err := client.Message.Parse()
	if err != nil || len(msg.Header["Message-Id"]) == 0 {
		t.Error("Did not set Message-ID header on message")
	}

	if !strings.HasSuffix(msg.Header["Message-Id"][0], ".1@TestHost") {
		t.Errorf("Did not set Message-ID header correctly, got: '%s'", msg.Header["Message-Id"][0])
	}

	if !called || client.Message.Id != 1 {
		t.Error("Did not set message ID")
	}

	wg.Wait()
}

func TestServer_Digest_Has_Header_Message_Id(t *testing.T) {
	var called bool

	server := Server{config: Config{Hostname: "TestHost"}}
	server.Mailbox = gomez.MockMailbox{
		NextID_: func() (uint64, error) { called = true; return 2, nil },
	}

	client, pipe := getTestClient()
	client.conn = &mocks.Conn{RemoteAddress: "invalid_addr"}
	client.Message = &gomez.Message{
		Raw: "From: Mary\r\nMessage-ID: My_ID\r\nDate: Today\r\n\r\nHey Mary how are you?",
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		server.Digest(client)
		wg.Done()
	}()

	pipe.ReadResponse(451)
	pipe.Close()

	msg, err := client.Message.Parse()
	if err != nil || len(msg.Header["Message-Id"]) == 0 {
		t.Error("Did not set Message-ID header on message")
	}

	if msg.Header["Message-Id"][0] != "My_ID" {
		t.Errorf("Did not set Message-ID header correctly, got: '%s'", msg.Header["Message-Id"][0])
	}

	if !called || client.Message.Id != 2 {
		t.Error("Did not set message ID")
	}

	wg.Wait()
}

func TestServer_Digest_Has_Message_Id(t *testing.T) {
	var called bool

	server := Server{config: Config{Hostname: "TestHost"}}
	server.Mailbox = gomez.MockMailbox{
		NextID_: func() (uint64, error) { called = true; return 2, nil },
	}

	client, pipe := getTestClient()
	client.conn = &mocks.Conn{RemoteAddress: "invalid_addr"}

	client.Message = &gomez.Message{
		Raw: "From: Mary\r\nMessage-ID: My_ID\r\nDate: Today\r\n\r\nHey Mary how are you?",
		Id:  53,
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		server.Digest(client)
		wg.Done()
	}()

	pipe.ReadResponse(451)
	pipe.Close()

	msg, err := client.Message.Parse()
	if err != nil || len(msg.Header["Message-Id"]) == 0 {
		t.Error("Did not set Message-ID header on message")
	}

	if called || client.Message.Id != 53 {
		t.Error("Was not supposed to touch already existing client.Message.Id")
	}

	wg.Wait()
}

func TestServer_Digest_Received_Header(t *testing.T) {
	var called bool

	server := Server{config: Config{Hostname: "TestHost"}}
	server.Mailbox = gomez.MockMailbox{
		NextID_: func() (uint64, error) { called = true; return 2, nil },
		Queue_:  func(*gomez.Message) error { return errors.New("Error processing") },
	}

	client, pipe := getTestClient()
	client.conn = &mocks.Conn{RemoteAddress: "1.2.3.4:567"}

	client.Message = &gomez.Message{
		Raw: "From: Mary\r\nMessage-ID: My_ID\r\nDate: Today\r\n\r\nHey Mary how are you?",
		Id:  53,
	}

	client.Message.AddRcpt(&mail.Address{"Name", "Addr@es"})

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		server.Digest(client)
		wg.Done()
	}()

	pipe.ReadResponse(451)
	wg.Wait()

	msg, err := client.Message.Parse()
	if err != nil || len(msg.Header["Received"]) == 0 {
		t.Error("Error parsing message")
	}

	if !strings.HasPrefix(msg.Header["Received"][0],
		`from  ([1.2.3.4]) by TestHost (Gomez) with ESMTP id 53 for "Name" <Addr@es>;`) {
		t.Errorf("Got: %s", msg.Header["Received"][0])
	}

	if testing.Short() {
		t.Log("Short version skips network")
		return
	}

	client.conn = &mocks.Conn{RemoteAddress: "74.125.230.118:1234"}

	wg.Add(1)
	go func() {
		server.Digest(client)
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
		`from  (lhr14s24-in-f22.1e100.net [74.125.230.118]) by TestHost (Gomez) with ESMTP id 53 for "Name" <Addr@es>;`) {
		t.Errorf("Got: %s", msg.Header["Received"][0])
	}
}
