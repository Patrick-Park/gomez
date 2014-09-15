package smtp

import (
	"fmt"
	"net"
	"net/textproto"
	"regexp"
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

// Reply is an SMTP reply. It contains a status
// code and a message.
type Reply struct {
	Code int
	Msg  string
}

// Implements the Stringer interface for pretty printing
func (r Reply) String() string { return fmt.Sprintf("%d %s", r.Code, r.Msg) }

// An action is performed in the context of a jj given a
// string parameter
type Action func(*Client, string)

// Map of supported commands. The server's command specification.
type CommandSpec map[string]Action

// SMTP host server instance
type Server struct {
	sync.Mutex
	spec    *CommandSpec
	Mailbox gomez.Mailbox
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
