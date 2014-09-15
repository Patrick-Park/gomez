package smtp

import (
	"log"
	"net"
	"net/textproto"
	"strconv"
	"strings"
	"sync"

	"github.com/gbbr/gomez"
)

type InputMode int

const (
	MODE_FREE InputMode = iota
	MODE_HELO
	MODE_MAIL
	MODE_RCPT
	MODE_DATA
	MODE_QUIT
)

// Configuration settings for the SMTP server
type Config struct {
	Port    int
	Mailbox gomez.Mailbox
}

type Host interface {
	Digest(c *Client) error
	Run(ctx *Client, msg string)
}

// SMTP Server instance
type Server struct {
	sync.Mutex
	spec    *CommandSpec
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

// Digests a message. Delivers or enqueues messages
// according to reciepients
func (s *Server) Digest(c *Client) error {
	s.Lock()
	defer s.Unlock()

	return nil
}

// Runs a command on the server's command spec
func (s *Server) Run(ctx *Client, msg string) {
	if !commandFormat.MatchString(msg) {
		ctx.Reply(badCommand)
	}

	parts := commandFormat.FindStringSubmatch(msg)
	cmd, params := parts[1], strings.Trim(parts[2], " ")

	command, ok := (*s.spec)[cmd]
	if !ok {
		ctx.Reply(badCommand)
	}

	command(ctx, params)
}

// A connected client. Holds state information
// and built message.
type Client struct {
	Id   string
	msg  *gomez.Message
	Mode InputMode
	conn *textproto.Conn
	Host Host
}

// Serves a new SMTP connection and handles all
// incoming commands
func (c *Client) Serve() {
	for {
		switch c.Mode {
		case MODE_QUIT:
			return

		case MODE_DATA:
			msg, err := c.conn.ReadDotLines()
			if err != nil {
				log.Printf("Could not read dot lines: %s\n", err)
			}

			c.msg.SetBody(strings.Join(msg, ""))
			c.Host.Digest(c)
			c.Reset()
			c.Reply(Reply{250, "Message queued"})

		default:
			msg, err := c.conn.ReadLine()
			if err != nil {
				log.Printf("Could not read input: %s\n", err)
			}

			c.Host.Run(c, msg)
		}
	}
}

// Replies to the client
func (c *Client) Reply(r Reply) {
	c.conn.PrintfLine("%s", r)
}

// Resets the client to "HELO" InputMode
// and empties the message buffer
func (c *Client) Reset() {
	c.msg = new(gomez.Message)
	c.Mode = MODE_HELO
}
