package smtp

import (
	"fmt"
	"io"
	"log"
	"net/textproto"
	"strings"

	"github.com/gbbr/gomez"
)

type InputMode int

const (
	MODE_HELO InputMode = iota
	MODE_MAIL
	MODE_RCPT
	MODE_DATA
	MODE_QUIT
)

// A connected client. Holds state information
// and built message.
type Client struct {
	Id   string
	msg  *gomez.Message
	Mode InputMode
	conn *textproto.Conn
	Host MailService
}

// Replies to the client
func (c *Client) Notify(r Reply) error { return c.conn.PrintfLine("%s", r) }

// Serves a new SMTP connection and handles all
// incoming commands
func (c *Client) Serve() {
	for {
		msg, err := c.conn.ReadLine()
		if !isConnectionActive(err) {
			break
		}

		err = c.Host.Run(c, msg)
		if c.Mode == MODE_QUIT || !isConnectionActive(err) {
			break
		}
	}

	c.conn.Close()
}

// Resets the client to "HELO" InputMode
// and empties the message buffer
func (c *Client) Reset() {
	c.msg = new(gomez.Message)
	c.Mode = MODE_MAIL
	c.Id = ""
}

// Checks if an error was returned and logs it.
// If the error was EOF it returns true
func isConnectionActive(err error) bool {
	if err != nil {
		log.Printf("Error processing I/O: %s", err)

		if err == io.EOF {
			return false
		}
	}

	return true
}

// Reply is an SMTP reply. It contains a status
// code and a message. If the message is multiline
// the Msg property can separate lines using the ';'
// character
type Reply struct {
	Code int
	Msg  string
}

// Implements the Stringer interface for pretty printing
// according to SMTP specifications. Should be used to reply.
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
