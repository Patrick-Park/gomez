package smtp

import (
	"log"
	"net"
	"net/textproto"
	"strconv"

	"github.com/gbbr/gomez"
)

// Configuration settings for the SMTP server
type Config struct {
	Port    int
	Mailbox gomez.Mailbox
}

type InputMode int

const (
	MODE_FREE InputMode = iota
	MODE_HELO
	MODE_MAIL
	MODE_RCPT
	MODE_DATA
)

type Server struct {
	cs CommandSpec
	mb gomez.Mailbox
}

type Client struct {
	msg  *gomez.Message
	Mode InputMode
	conn *textproto.Conn
}

func (s *Server) createClient(conn net.Conn) {
	c := &Client{
		msg:  new(gomez.Message),
		Mode: MODE_HELO,
		conn: textproto.NewConn(conn),
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

func Start(conf Config) {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(conf.Port))
	if err != nil {
		log.Fatalf("Could not open port %d.", conf.Port)
	}

	srv := &Server{mb: conf.Mailbox, cs: make(CommandSpec)}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Print("Error accepting an incoming connection.")
		}

		go srv.createClient(conn)
	}
}
