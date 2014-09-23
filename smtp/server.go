package smtp

import (
	"log"
	"net"
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
	Query(addr gomez.Address) gomez.QueryStatus
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

// Digests a child connection. Delivers or enqueues messages
// according to reciepients
func (s Server) Digest(c *Client) error {
	s.Lock()
	defer s.Unlock()

	return nil
}

var (
	commandFormat = regexp.MustCompile("^([a-zA-Z]{4})(?:[ ](.*))?$")
	badCommand    = Reply{502, "5.5.2 Error: command not recoginized"}
)

// Runs a command in the context of a child connection
func (s Server) Run(ctx *Client, msg string) error {
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

// Returns the configuration of the server
func (s Server) Settings() Config { return s.config }

// Queries the host mailbox for a user by string or gomez.Address
func (s Server) Query(addr gomez.Address) gomez.QueryStatus { return s.Mailbox.Query(addr) }

// Creates a new client based on the given connection
func (s Server) CreateClient(conn net.Conn) {
	c := &Client{
		Message: gomez.NewMessage(),
		Mode:    MODE_HELO,
		host:    s,
		conn:    textproto.NewConn(conn),
		rawConn: conn,
	}

	c.Notify(Reply{220, s.config.Hostname + " Gomez SMTP"})
	c.Serve()
}
