package smtp

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/textproto"
	"strings"

	"github.com/gbbr/gomez"
)

// TransactionState is the state an SMTP transaction is in.
type TransactionState int

const (
	// StateHELO is the initial state. HELO/EHLO command is expected.
	StateHELO TransactionState = iota

	// StateMAIL is the post-HELO state. The Return-Path address is expected.
	StateMAIL

	// StateRCPT is the post-MAIL mode. The recipient data is expected.
	StateRCPT

	// StateDATA signifies eligibility to receive data.
	StateDATA
)

// Client holds information about an SMTP transaction.
type Client struct {
	// The name the client has identified with
	ID string

	// The message that the client is building via the current transaction.
	Message *gomez.Message

	// The current state of the transaction.
	Mode TransactionState

	host SMTPServer      // Host server instance
	conn net.Conn        // Network connection
	text *textproto.Conn // Textproto wrapper of network connection
}

// Notify sends the given reply back to the connected client.
func (c *Client) Notify(r Reply) error { return c.text.PrintfLine("%s", r) }

// Serve listens for incoming commands and runs them on the host instance.
func (c *Client) Serve() {
	for {
		msg, err := c.text.ReadLine()
		if isEOF(err) {
			break
		}

		err = c.host.Run(c, msg)
		if isEOF(err) {
			break
		}
	}

	c.text.Close()
}

// Reset empties the message buffer and sets the state back to HELO.
func (c *Client) Reset() {
	c.Message = new(gomez.Message)

	if c.Mode > StateHELO {
		c.Mode = StateMAIL
	}
}

// Logs error and validates whether it was EOF.
func isEOF(err error) bool {
	if err == io.EOF {
		return true
	}

	if err != nil {
		log.Printf("Error processing I/O: %s\r\n", err)
	}

	return false
}

// Reply is an RFC821 compliant SMTP response.
type Reply struct {
	Code int
	Msg  string
}

// String implements the Stringer interface. If a multi-line reply is desired,
// separate lines via LF (\n)
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

var (
	replyBadCommand      = Reply{502, "5.5.2 Error: command not recoginized"}
	replyErrorProcessing = Reply{451, "Requested action aborted: error in processing"}
)
