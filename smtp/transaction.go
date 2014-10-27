package smtp

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/textproto"
	"strings"

	"github.com/gbbr/gomez/mailbox"
)

const (
	// stateHELO is the initial state. HELO/EHLO command is expected.
	stateHELO = iota
	// stateMAIL is the post-HELO state. The Return-Path address is expected.
	stateMAIL
	// stateRCPT is the post-MAIL mode. The recipient data is expected.
	stateRCPT
	// stateDATA signifies eligibility to receive data.
	stateDATA
)

// transaction holds information about an SMTP transaction.
type transaction struct {
	// The name the client has identified with
	ID string
	// The message that the client is building via the current transaction.
	Message *mailbox.Message
	// The current state of the transaction. A transaction can be in the following
	// states: stateHELO, stateMAIL, stateRCPT and stateDATA.
	Mode int

	host     host            // Host server instance
	conn     net.Conn        // Network connection
	text     *textproto.Conn // Textproto wrapper of network connection
	addrHost string          // addrHost holds the first result of the IP reverse-lookup
	addrIP   string          // addrIP is the connection's IP address
}

// notify sends the given reply back to the connected client.
func (c *transaction) notify(r reply) error { return c.text.PrintfLine("%s", r) }

// serve listens for incoming commands and runs them on the host instance.
func (c *transaction) serve() {
	for {
		msg, err := c.text.ReadLine()
		if isEOF(err) {
			break
		}

		err = c.host.run(c, msg)
		if isEOF(err) {
			break
		}
	}
}

// reset empties the message buffer and sets the state back to HELO.
func (c *transaction) reset() {
	c.Message = new(mailbox.Message)

	if c.Mode > stateHELO {
		c.Mode = stateMAIL
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

// reply is an RFC821 compliant SMTP response.
type reply struct {
	Code int
	Msg  string
}

// String implements the Stringer interface. If a multi-line reply is desired,
// separate lines via LF (\n)
func (r reply) String() string {
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
	replyBadCommand      = reply{502, "5.5.2 Error: command not recoginized"}
	replyErrorProcessing = reply{451, "Requested action aborted: error in processing"}
)
