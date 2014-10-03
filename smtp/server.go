package smtp

import (
	"fmt"
	"log"
	"net"
	"net/mail"
	"net/textproto"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gbbr/gomez"
)

type SMTPServer interface {
	// Runs a command from the SMTPServer's CommandSpec in the
	// context of a connected client.
	Run(ctx *Client, msg string) error

	// Digests a client. It attaches necessarry headers to the message
	// and inserts it into the MailBox or queues it for relaying.
	Digest(c *Client) error

	// Returns the server configuration flags.
	Settings() Config

	// Queries the mailbox for a user. See gomez.QueryStatus for
	// information on response types.
	Query(addr *mail.Address) gomez.QueryStatus
}

// SMTP host server instance. Holds the CommandSpec, configuration flags
// and an attached MailBox.
type Server struct {
	sync.Mutex
	spec    *CommandSpec
	config  Config
	Mailbox gomez.Mailbox
}

// Maps commands to their actions. Actions run in the context
// of a connected client and a string of params
type CommandSpec map[string]func(*Client, string) error

// Configuration settings for the SMTP server
type Config struct {
	ListenAddr string
	Hostname   string
	Relay      bool
	TLS        bool
	Vrfy       bool
}

// Starts the SMTP server given the specified configuration.
// Accepts incoming connections and initiates communication.
func Start(mb gomez.Mailbox, conf Config) {
	ln, err := net.Listen("tcp", conf.ListenAddr)
	if err != nil {
		log.Fatalf("Could not listen on %s.", conf.ListenAddr)
	}

	srv := &Server{
		Mailbox: mb,
		config:  conf,
		spec: &CommandSpec{
			"HELO": cmdHELO,
			"EHLO": cmdEHLO,
			"MAIL": cmdMAIL,
			"RCPT": cmdRCPT,
			"DATA": cmdDATA,
			"RSET": cmdRSET,
			"NOOP": cmdNOOP,
			"VRFY": cmdVRFY,
			"QUIT": cmdQUIT,
		},
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Print("Error accepting an incoming connection.")
		}

		go srv.CreateClient(conn)
	}
}

// Analyzes and places a complete message into the queue. If the message does not pass
// all requirements, if the client can not be validated or if an error occurs, Digest
// notifies the client connection.
func (s Server) Digest(client *Client) error {
	var helloHost string

	msg, err := client.Message.Parse()
	if err != nil || len(msg.Header["Date"]) == 0 || len(msg.Header["From"]) == 0 {
		return client.Notify(Reply{550, "Message not RFC 2822 compliant."})
	}

	remoteAddress := client.rawConn.RemoteAddr()
	helloIp, _, err := net.SplitHostPort(remoteAddress.String())
	if err != nil {
		return client.Notify(replyErrorProcessing)
	}

	helloHosts, err := net.LookupAddr(helloIp)
	if len(helloHosts) > 0 {
		helloHost = helloHosts[0] + " "
	}

	id, err := s.Mailbox.NextID()
	if err != nil {
		return client.Notify(replyErrorProcessing)
	}

	if len(msg.Header["Message-ID"]) == 0 {
		messageId := fmt.Sprintf("<%x.%d@%s>", time.Now().UnixNano(), id, s.config.Hostname)
		client.Message.PrependHeader("Message-ID", messageId)
	}

	receivedHeader := fmt.Sprintf("from %s (%s[%s])\r\n\tby %s (Gomez) with ESMTP id %d for %s; %s",
		client.Id,
		helloHost,
		helloIp,
		s.config.Hostname,
		id,
		client.Message.Rcpt()[0],
		time.Now())

	client.Message.PrependHeader("Received", receivedHeader)

	err = s.Mailbox.Queue(client.Message)
	if err != nil {
		return client.Notify(replyErrorProcessing)
	}

	return client.Notify(Reply{250, fmt.Sprintf("message queued (%x)", id)})
}

var commandFormat = regexp.MustCompile("^([a-zA-Z]{4})(?:[ ](.*))?$")

// Runs a command in the context of a child connection
func (s Server) Run(ctx *Client, msg string) error {
	if !commandFormat.MatchString(msg) {
		return ctx.Notify(replyBadCommand)
	}

	parts := commandFormat.FindStringSubmatch(msg)
	cmd, params := parts[1], strings.Trim(parts[2], " ")

	command, ok := (*s.spec)[cmd]
	if !ok {
		return ctx.Notify(replyBadCommand)
	}

	return command(ctx, params)
}

// Returns the configuration of the server
func (s Server) Settings() Config { return s.config }

// Queries the host mailbox for a user by string or Address
func (s Server) Query(addr *mail.Address) gomez.QueryStatus { return s.Mailbox.Query(addr) }

// Creates a new client based on the given connection
func (s Server) CreateClient(conn net.Conn) {
	c := &Client{
		Message: new(gomez.Message),
		Mode:    MODE_HELO,
		host:    s,
		conn:    textproto.NewConn(conn),
		rawConn: conn,
	}

	c.Notify(Reply{220, s.config.Hostname + " Gomez SMTP"})
	c.Serve()
}
