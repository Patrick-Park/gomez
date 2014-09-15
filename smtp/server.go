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

// A Host is a structure that can run commands in the context of
// a child connection, as well as consume their input/state
type Host interface {
	Run(ctx *Client, msg string)
	Digest(c *Client) error
}

// SMTP host server instance
type Server struct {
	sync.Mutex
	spec    *CommandSpec
	Mailbox gomez.Mailbox
}

// Map of supported commands. The server's command specification.
type CommandSpec map[string]func(*Client, string)

// Configuration settings for the SMTP server
type Config struct {
	Port    int
	Mailbox gomez.Mailbox
}

// Starts the SMTP server given the specified configuration.
// Accepts incoming connections and initiates communication.
func Start(conf Config) {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(conf.Port))
	if err != nil {
		log.Fatalf("Could not open port %d.", conf.Port)
	}

	srv := &Server{
		Mailbox: conf.Mailbox,
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
func (s *Server) Run(ctx *Client, msg string) {
	if !commandFormat.MatchString(msg) {
		ctx.Reply(badCommand)
		return
	}

	parts := commandFormat.FindStringSubmatch(msg)
	cmd, params := parts[1], strings.Trim(parts[2], " ")

	command, ok := (*s.spec)[cmd]
	if !ok {
		ctx.Reply(badCommand)
		return
	}

	command(ctx, params)
}

// Creates a new client based on the given connection
func (s *Server) createClient(conn net.Conn) {
	c := &Client{
		msg:  new(gomez.Message),
		Mode: MODE_HELO,
		conn: textproto.NewConn(conn),
		Host: s,
	}

	c.Serve()
	c.conn.Close()
}
