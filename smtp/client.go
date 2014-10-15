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
	// Initial state. Expecting HELO/EHLO handshake.
	StateHELO TransactionState = iota

	// The Return-Path address is expected.
	StateMAIL

	// The recipient data is expected.
	StateRCPT

	// Eligible to receive data or further recipients.
	StateDATA
)

// Holds information about an SMTP transaction.
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

// Replies to the current connection using the given Reply value.
func (c *Client) Notify(r Reply) error { return c.text.PrintfLine("%s", r) }

// Handles the transaction by picking up messages and executing them on the host.
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

// Cleans the aggregated message and resets the current state of the transaction.
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

// Pretty-print for the Reply. If a multi-line reply is desired, separate lines
// via LF (\n)
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
