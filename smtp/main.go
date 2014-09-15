package smtp

import (
	"log"
	"net"
	"strconv"

	"github.com/gbbr/gomez"
)

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
