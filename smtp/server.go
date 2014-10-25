package smtp

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/mail"
	"net/textproto"
	"regexp"
	"strings"
	"time"

	"github.com/gbbr/gomez/mailbox"
	"github.com/gbbr/jamon"
)

type host interface {
	// run executes an SMTP command.
	run(ctx *transaction, msg string) error

	// digest attempts to finalize an SMTP transaction.
	digest(c *transaction) error

	// settings returns the server's configuration flags.
	settings() jamon.Group

	// query searches on the server for a given address.
	query(addr *mail.Address) mailbox.QueryResult
}

// Host server instance.
type server struct {
	spec     *commandSpec
	config   jamon.Group
	Enqueuer mailbox.Enqueuer
}

// commandSpec holds a set of supported commands, mapping names to actions.
type commandSpec map[string]func(*transaction, string) error

// Valid command format. Currently commands must be exactly 4 letters, optionally
// followed by a parameter.
var commandFormat = regexp.MustCompile("^([a-zA-Z]{4})(?:[ ](.*))?$")

// ErrMinConfig is returned when the minimum configuration is not passed to the server.
// A listen address (host:port) passed via the 'listen' and a 'host' is required.
var ErrMinConfig = errors.New("Minimum config not met. Need at least 'listen' and 'host'")

// Start initiates a new SMTP server given an Enqueuer and a configuration.
func Start(mq mailbox.Enqueuer, cfg jamon.Group) error {
	if !cfg.Has("listen") || !cfg.Has("host") {
		return ErrMinConfig
	}

	ln, err := net.Listen("tcp", cfg.Get("listen"))
	if err != nil {
		return err
	}

	spec := &commandSpec{
		"HELO": cmdHELO,
		"EHLO": cmdEHLO,
		"MAIL": cmdMAIL,
		"RCPT": cmdRCPT,
		"DATA": cmdDATA,
		"RSET": cmdRSET,
		"NOOP": cmdNOOP,
		"VRFY": cmdVRFY,
		"QUIT": cmdQUIT,
	}

	srv := &server{Enqueuer: mq, config: cfg, spec: spec}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Error accepting an incoming connection: %s\r\n", err)
			continue
		}

		go srv.createTransaction(conn)
	}
}

// createTransaction creates a new client based on the given connection.
func (s server) createTransaction(conn net.Conn) {
	defer conn.Close()

	ip, _, err := net.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		return
	}

	c := &transaction{
		Message: new(mailbox.Message),
		Mode:    stateHELO,
		host:    s,
		text:    textproto.NewConn(conn),
		conn:    conn,
		addrIP:  ip,
	}

	if helloHosts, _ := net.LookupAddr(ip); len(helloHosts) > 0 {
		c.addrHost = strings.TrimRight(helloHosts[0], ".") + " "
	}

	c.notify(reply{220, s.config.Get("host") + " Gomez SMTP"})
	c.serve()
}

// settings returns the configuration of the server.
func (s server) settings() jamon.Group { return s.config }

// query asks the attached enqueuer to search for an address.
func (s server) query(addr *mail.Address) mailbox.QueryResult {
	return s.Enqueuer.Query(addr)
}

// run executes a command in the context of a child connection.
func (s server) run(ctx *transaction, msg string) error {
	if !commandFormat.MatchString(msg) {
		return ctx.notify(replyBadCommand)
	}

	parts := commandFormat.FindStringSubmatch(msg)
	cmd, params := parts[1], strings.Trim(parts[2], " ")

	command, ok := (*s.spec)[cmd]
	if !ok {
		return ctx.notify(replyBadCommand)
	}

	return command(ctx, params)
}

// digest finalizes the SMTP transaction by validating the message and attempting
// to enqueue it. This method attaches transitional headers as per RFC 5321.
func (s server) digest(client *transaction) error {
	msg, err := client.Message.Parse()
	if err != nil || len(msg.Header["Date"]) == 0 || len(msg.Header["From"]) == 0 {
		return client.notify(reply{550, "Message not RFC 2822 compliant."})
	}

	if client.Message.ID == 0 {
		id, err := s.Enqueuer.GUID()
		if err != nil {
			return client.notify(replyErrorProcessing)
		}

		client.Message.ID = id
	}

	if len(msg.Header["Message-Id"]) == 0 {
		client.Message.PrependHeader(
			"Message-ID", "<%x.%d@%s>", time.Now().UnixNano(),
			client.Message.ID, s.config.Get("host"),
		)
	}

	client.Message.PrependHeader(
		"Received", "from %s (%s[%s])\r\n\tby %s (Gomez) with ESMTP id %d for %s; %s",
		client.ID, client.addrHost, client.addrIP, s.config.Get("host"),
		client.Message.ID, client.Message.Rcpt()[0], time.Now(),
	)

	err = s.Enqueuer.Enqueue(client.Message)
	if err != nil {
		return client.notify(replyErrorProcessing)
	}

	client.reset()

	return client.notify(reply{250, fmt.Sprintf("message queued (%x)", client.Message.ID)})
}
