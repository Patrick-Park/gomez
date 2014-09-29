package smtp

import (
	"errors"
	"log"
	"net"
	"net/mail"
	"net/textproto"
	"regexp"
	"strings"
	"sync"

	"github.com/gbbr/gomez"
)

type MailService interface {
	// Runs a command from the MailService's CommandSpec in the
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

// Return by Digest when the message does not comply with RFC 2822 header specifications
var ERR_MESSAGE_NOT_COMPLIANT = errors.New("Message is not RFC 2822 compliant")

// Digests a child connection. Delivers or enqueues messages
// according to reciepients
func (s Server) Digest(c *Client) error {
	msg, err := c.Message.Parse()
	if err != nil || len(msg.Header["Date"]) == 0 || len(msg.Header["From"]) == 0 {
		return ERR_MESSAGE_NOT_COMPLIANT
	}
	// Received: from
	// the name the sending computer gave for itself (the name associated with that computer's IP address [its IP address])
	// by
	// the receiving computer's name (the software that computer uses) (usually Sendmail, qmail or Postfix)
	// with protocol (usually SMTP or ESMTP) id id assigned by local computer for logging;
	// timestamp (usually given in the computer's localtime; see below for how you can convert these all to your time)

	// Received:     from maroon.pobox.com (maroon.pobox.com [208.72.237.40]) by mailstore.pobox.com
	//        (Postfix) with ESMTP id 847989746 for <address>; Wed, 15 Jun 2011 10:42:09 -0400 (EDT)

	// Generate local Message ID
	// If message has no Message-ID header, use the generated one
	// Add retrieve header and log/queue message using generated Message ID
	// Queue message

	s.Lock()
	defer s.Unlock()

	return nil
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
