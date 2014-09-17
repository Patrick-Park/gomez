package smtp

import (
	"fmt"
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

// A connected client. Holds state information
// and built message.
type Client struct {
	Id   string
	msg  *gomez.Message
	Mode InputMode
	conn *textproto.Conn
	Host MailService
}

// Serves a new SMTP connection and handles all
// incoming commands
func (c *Client) Serve() {
	for {
		msg, err := c.conn.ReadLine()
		if err != nil {
			log.Printf("Could not read input: %s\n", err)
		}

		err = c.Host.Run(c, msg)
		if err != nil {
			log.Printf("Error running command '%s': %s", msg, err)
		}

		if c.Mode == MODE_QUIT {
			return
		}
	}

	c.conn.Close()
}

// Replies to the client
func (c *Client) Notify(r Reply) error {
	return c.conn.PrintfLine("%s", r)
}

// Resets the client to "HELO" InputMode
// and empties the message buffer
func (c *Client) Reset() {
	c.msg = new(gomez.Message)
	c.Mode = MODE_MAIL
	c.Id = ""
}
