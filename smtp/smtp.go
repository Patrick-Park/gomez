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
)

// Configuration settings for the SMTP server
type Config struct {
	Port    int
	Mailbox gomez.Mailbox
}

// SMTP Server instance
type Server struct {
	sync.Mutex
	cs      *CommandSpec
	Mailbox gomez.Mailbox
}

// Starts the SMTP server given the specified configuration.
// Accepts incoming connections and initiates communication.
func Start(conf Config) {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(conf.Port))
	if err != nil {
		log.Fatalf("Could not open port %d.", conf.Port)
	}

	srv := &Server{Mailbox: conf.Mailbox, cs: NewCommandSpec()}

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
	c.Accept()
	c.conn.Close()
}

// Digests a message. Delivers or enqueues messages
// according to reciepients
func (s *Server) digest(msg *gomez.Message) error {
	s.Lock()
	defer s.Unlock()

	return nil
}

// A connected client. Holds state information
// and built message.
type Client struct {
	msg  *gomez.Message
	Mode InputMode
	conn *textproto.Conn
	Host *Server
}

// Accepts a new SMTP connection and handles all
// incoming commands
func (c *Client) Accept() {
	for {
		msg, err := c.conn.ReadLine()
		if err != nil {
			log.Printf("Could not read input: %s\n", err)
		}

		if strings.ToUpper(msg) == "QUIT" {
			break
		}

		switch c.Mode {
		case MODE_DATA:
			if msg != "." {
				c.msg.AddBody(msg)
			} else {
				c.Host.digest(c.msg)
				c.Reset()
			}
		default:
			c.Host.cs.Run(c, msg)
		}
	}
}

// Resets the client to "HELO" InputMode
// and empties the message buffer
func (c *Client) Reset() {
	c.msg = new(gomez.Message)
	c.Mode = MODE_HELO
}
