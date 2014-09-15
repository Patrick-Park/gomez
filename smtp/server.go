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
	Run(ctx Child, msg string)
	Digest(c Child) error
}

// Reply is an SMTP reply. It contains a status
// code and a message.
type Reply struct {
	Code int
	Msg  string
}

// Implements the Stringer interface for pretty printing
func (r Reply) String() string { return fmt.Sprintf("%d %s", r.Code, r.Msg) }

// A Command is an SMTP supported command that performs
// an action in the context of the calling client and returns
// a Reply.
type Action func(Child, string) Reply

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
func (s *Server) Digest(c Child) error {
	s.Lock()
	defer s.Unlock()

	return nil
}

// Runs a command in the context of a child connection
func (s *Server) Run(ctx Child, msg string) {
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
