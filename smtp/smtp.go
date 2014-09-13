package smtp

import (
	"log"
	"net"
	"net/textproto"
	"strconv"

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
	// Specifies the port to listen on for this service
	Port int

	// Specifies the gomez.Mailbox to use
	Mailbox gomez.Mailbox
}

// SMTP Server instance
type Server struct {
	cs      CommandSpec
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

	for {
		msg, err := c.conn.ReadLine()
		if err != nil {
			log.Printf("Could not read input: %s\n", err)
		}

		if c.Mode == MODE_DATA {
			// if "."

			// else :
			c.msg.AddBody(msg)
		} else {
			// Run commands
			s.cs.Run(c, msg)
		}
	}

	c.conn.Close()
}

type Client struct {
	msg  *gomez.Message
	Mode InputMode
	conn *textproto.Conn
	Host *Server
}
