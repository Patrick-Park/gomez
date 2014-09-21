package smtp

import (
	"fmt"
	"io"
	"log"
	"net/textproto"
	"strings"

	"github.com/gbbr/gomez"
)

// InputMode is the current state that a client is in and
// represents a step in the SMTP process.
type InputMode int

const (
	MODE_HELO InputMode = iota
	MODE_MAIL
	MODE_RCPT
	MODE_DATA
	MODE_QUIT
)

// Client is a form of context for the each connected user.
// It holds the current SMTP state and the created message,
// as well as exposes the host mail service.
type Client struct {
	Id   string
	msg  *gomez.Message
	Mode InputMode
	conn *textproto.Conn
	Host MailService
}

// Replies to the client via the attached connection. To find
// out more about how Reply is printed look into Reply.String()
func (c *Client) Notify(r Reply) error { return c.conn.PrintfLine("%s", r) }

// Initiates a new channel of communication between the connected
// client and the attached mailing service. It picks up commands
// from the client and executes them on the host. Serve is a blocking
// operation which completes after the client closes the connection
// or issues the QUIT command.
func (c *Client) Serve() {
	for {
		msg, err := c.conn.ReadLine()
		if isEOF(err) {
			break
		}

		err = c.Host.Run(c, msg)
		if isEOF(err) || c.Mode == MODE_QUIT {
			break
		}
	}

	c.conn.Close()
}

// Resets the state of the client. It resets the accumulated message and
// reverts the mode to MODE_MAIL. The only thing that is kept is the ID
// of the client received via the EHLO/HELO command.
func (c *Client) Reset() {
	c.msg = new(gomez.Message)
	c.Mode = MODE_MAIL
}

// Checks if the given error is of type io.EOF, otherwise
// it logs the error to StdErr and/or returns false
func isEOF(err error) bool {
	if err != nil {
		log.Printf("Error processing I/O: %s", err)

		if err == io.EOF {
			return true
		}
	}

	return false
}

// Reply is an RFC821 compliant SMTP response. It contains
// a code and a message which can be one line or multi-line.
// Multi-line messages should be separated by \n.
type Reply struct {
	Code int
	Msg  string
}

// Pretty-print for Reply. Implements the Stringer interface.
// It prints the Reply in an RFC821 compliant fashion and allows
// multi-line messages.
func (r Reply) String() string {
	var output string

	lines := strings.Split(r.Msg, "\n")
	for n, line := range lines {
		format := "%d-%s\n"
		if len(lines)-1 == n {
			format = "%d %s"
		}

		output += fmt.Sprintf(format, r.Code, line)
	}

	return output
}
