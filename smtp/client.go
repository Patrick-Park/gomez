package smtp

import (
	"fmt"
	"log"
	"net/textproto"

	"github.com/gbbr/gomez"
)

type InputMode int

const (
	MODE_FREE InputMode = iota
	MODE_HELO
	MODE_MAIL
	MODE_RCPT
	MODE_DATA
	MODE_QUIT
)

// Reply is an SMTP reply. It contains a status
// code and a message.
type Reply struct {
	Code int
	Msg  string
}

// Implements the Stringer interface for pretty printing
func (r Reply) String() string { return fmt.Sprintf("%d %s", r.Code, r.Msg) }

// A connected client. Holds state information
// and built message.
type Client struct {
	Id   string
	msg  *gomez.Message
	Mode InputMode
	conn *textproto.Conn
	Host Host
}

// Serves a new SMTP connection and handles all
// incoming commands
func (c *Client) Serve() {
	for {
		msg, err := c.conn.ReadLine()
		if err != nil {
			log.Printf("Could not read input: %s\n", err)
		}

		c.Host.Run(c, msg)
		if c.Mode == MODE_QUIT {
			return
		}
	}
}

// Replies to the client
func (c *Client) Reply(r Reply) {
	c.conn.PrintfLine("%s", r)
}

// Resets the client to "HELO" InputMode
// and empties the message buffer
func (c *Client) Reset() {
	c.msg = new(gomez.Message)
	c.Mode = MODE_HELO
}
