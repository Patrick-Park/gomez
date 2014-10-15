package smtp

import (
	"fmt"
	"log"
	"net"
	"net/mail"
	"net/textproto"
	"regexp"
	"strings"
	"time"

	"github.com/gbbr/gomez"
)

type SMTPServer interface {
	// Executes an SMTP command.
	Run(ctx *Client, msg string) error

	// Attempts to finalize an SMTP transaction.
	Digest(c *Client) error

	// Returns the server's configuration flags.
	Settings() Config

	// Queries the server for a given address.
	Query(addr *mail.Address) gomez.QueryResult
}

// Host server instance.
type Server struct {
	spec     *CommandSpec
	config   Config
	Enqueuer gomez.Enqueuer
}

// Server's set of supported commands. Maps commands to actions.
type CommandSpec map[string]func(*Client, string) error

// Valid command format. Currently commands must be exactly 4 letters, optionally
// followed by a parameter.
var commandFormat = regexp.MustCompile("^([a-zA-Z]{4})(?:[ ](.*))?$")

// Server configuration object
type Config struct {
	ListenAddr string
	Hostname   string
	Relay      bool
	TLS        bool
	Vrfy       bool
}

// Starts a new SMTP server given an Enqueuer and a configuration.
func Start(mq gomez.Enqueuer, conf Config) error {
	ln, err := net.Listen("tcp", conf.ListenAddr)
	if err != nil {
		return err
	}

	spec := &CommandSpec{
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

	srv := &Server{Enqueuer: mq, config: conf, spec: spec}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Error accepting an incoming connection: %s\r\n", err)
			continue
		}

		go srv.CreateClient(conn)
	}
}

// Creates a new client based on the given connection.
func (s Server) CreateClient(conn net.Conn) {
	c := &Client{
		Message: new(gomez.Message),
		Mode:    MODE_HELO,
		host:    s,
		text:    textproto.NewConn(conn),
		conn:    conn,
	}

	c.Notify(Reply{220, s.config.Hostname + " Gomez SMTP"})
	c.Serve()
}

// Returns the configuration of the server
func (s Server) Settings() Config { return s.config }

// Queries the host mailbox for a user by string or Address
func (s Server) Query(addr *mail.Address) gomez.QueryResult {
	return s.Enqueuer.Query(addr)
}

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

// Finalizes the SMTP transaction by validating the message headers and attempting
// to enqueue it. This method attaches transitional headers as per RFC 5321
func (s Server) Digest(client *Client) error {
	msg, err := client.Message.Parse()
	if err != nil || len(msg.Header["Date"]) == 0 || len(msg.Header["From"]) == 0 {
		return client.Notify(Reply{550, "Message not RFC 2822 compliant."})
	}

	if client.Message.ID == 0 {
		id, err := s.Enqueuer.NextID()
		if err != nil {
			return client.Notify(replyErrorProcessing)
		}

		client.Message.ID = id
	}

	// If the message doesn't have a Message-ID, add it
	if len(msg.Header["Message-Id"]) == 0 {
		client.Message.PrependHeader(
			"Message-ID",
			fmt.Sprintf(
				"<%x.%d@%s>",
				time.Now().UnixNano(), client.Message.ID, s.config.Hostname,
			),
		)
	}

	err = s.prependReceivedHeader(client)
	if err != nil {
		return client.Notify(replyErrorProcessing)
	}

	// Try to enqueue the message
	err = s.Enqueuer.Enqueue(client.Message)
	if err != nil {
		return client.Notify(replyErrorProcessing)
	}

	client.Reset()

	return client.Notify(Reply{250, fmt.Sprintf("message queued (%x)", client.Message.ID)})
}

// Attaches transitional headers to a client's message, such as "Received:".
func (s *Server) prependReceivedHeader(client *Client) error {
	var helloHost string

	// Find the remote connection's IP
	remoteAddress := client.conn.RemoteAddr()
	helloIp, _, err := net.SplitHostPort(remoteAddress.String())
	if err != nil {
		return err
	}

	// Try to resolve the IP's host by doing a reverse look-up
	helloHosts, err := net.LookupAddr(helloIp)
	if len(helloHosts) > 0 {
		helloHost = strings.TrimRight(helloHosts[0], ".") + " "
	}

	// Construct the Received header based on gathered information
	client.Message.PrependHeader(
		"Received",
		fmt.Sprintf(
			"from %s (%s[%s])\r\n\tby %s (Gomez) with ESMTP id %d for %s; %s",
			client.ID, helloHost, helloIp, s.config.Hostname, client.Message.ID,
			client.Message.Rcpt()[0], time.Now(),
		),
	)

	return nil
}
