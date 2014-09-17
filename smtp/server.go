package smtp

import (
	"log"
	"net"
	"net/textproto"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/gbbr/gomez"
)

var (
	commandFormat = regexp.MustCompile("^([a-zA-Z]{4})(?:[ ](.*))?$")
	badCommand    = Reply{502, "5.5.2 Error: command not recoginized"}
)

type MailService interface {
	// Runs a command with arguments in the context
	// of a given client.
	Run(ctx *Client, msg string) error

	// Digests a client's contents and tries to deliver
	// the message.
	Digest(c *Client) error

	// Returns the name of the host
	Name() string

	// Queries the mailbox for a user
	// Query(q string) *gomez.Address
}

// SMTP host server instance
type Server struct {
	sync.Mutex
	spec     *CommandSpec
	Mailbox  gomez.Mailbox
	Hostname string
}

// Maps commands to their actions. Actions run in the context
// of a connected client and a string of params
type CommandSpec map[string]func(*Client, string) error

// Configuration settings for the SMTP server
type Config struct {
	Port     int
	Mailbox  gomez.Mailbox
	Hostname string
}

// Starts the SMTP server given the specified configuration.
// Accepts incoming connections and initiates communication.
func Start(conf Config) {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(conf.Port))
	if err != nil {
		log.Fatalf("Could not open port %d.", conf.Port)
	}

	srv := &Server{
		Mailbox:  conf.Mailbox,
		Hostname: conf.Hostname,
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

		go srv.createClient(conn)
	}
}

// Digests a child connection. Delivers or enqueues messages
// according to reciepients
func (s *Server) Digest(c *Client) error {
	s.Lock()
	defer s.Unlock()

	return nil
}

// Runs a command in the context of a child connection
func (s *Server) Run(ctx *Client, msg string) error {
	if !commandFormat.MatchString(msg) {
		return ctx.Notify(badCommand)
	}

	parts := commandFormat.FindStringSubmatch(msg)
	cmd, params := parts[1], strings.Trim(parts[2], " ")

	command, ok := (*s.spec)[cmd]
	if !ok {
		return ctx.Notify(badCommand)
	}

	return command(ctx, params)
}

// Returns the name of the server
func (s Server) Name() string { return s.Hostname }

// Creates a new client based on the given connection
func (s *Server) createClient(conn net.Conn) {
	c := &Client{
		msg:  new(gomez.Message),
		Mode: MODE_HELO,
		conn: textproto.NewConn(conn),
		Host: s,
	}

	c.Notify(Reply{220, s.Name() + " Gomez SMTP"})
	c.Serve()
}
