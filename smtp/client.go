package smtp

import (
	"log"
	"net/textproto"
	"strings"

	"github.com/gbbr/gomez"
)

// A child of a host can reply to its connection,
// serve communication and reset its state
type Child interface {
	Serve()
	Reset()
	Reply(r Reply)
}

// A connected client. Holds state information
// and built message.
type Client struct {
	Id   string
	msg  *gomez.Message
	Mode InputMode
	conn *textproto.Conn
	Host Host
}

type InputMode int

const (
	MODE_FREE InputMode = iota
	MODE_HELO
	MODE_MAIL
	MODE_RCPT
	MODE_DATA
	MODE_QUIT
)

// Serves a new SMTP connection and handles all
// incoming commands
func (c *Client) Serve() {
	for {
		switch c.Mode {
		case MODE_QUIT:
			return

		case MODE_DATA:
			msg, err := c.conn.ReadDotLines()
			if err != nil {
				log.Printf("Could not read dot lines: %s\n", err)
			}

			c.msg.SetBody(strings.Join(msg, ""))
			c.Host.Digest(c)
			c.Reset()
			c.Reply(Reply{250, "Message queued"})

		default:
			msg, err := c.conn.ReadLine()
			if err != nil {
				log.Printf("Could not read input: %s\n", err)
			}

			c.Host.Run(c, msg)
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
