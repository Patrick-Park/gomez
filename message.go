package gomez

import (
	"bufio"
	"errors"
	"net/textproto"
	"strings"
)

// A message represents an e-mail message and
// holds information about sender, recepients
// and the message body
type Message struct {
	from    Address
	rcpt    []Address
	headers textproto.MIMEHeader
	body    string
}

// Adds a new recepient to the message
func (m *Message) AddRcpt(addr ...Address) { m.rcpt = append(m.rcpt, addr...) }

// Returns the message recepients
func (m Message) Rcpt() []Address { return m.rcpt }

// Adds a Reply-To address
func (m *Message) SetFrom(addr Address) { m.from = addr }

// Returns the Reply-To address
func (m Message) From() Address { return m.from }

// Sets the message body
func (m *Message) SetBody(msg string) { m.body = msg }

// Returns the message body
func (m Message) Body() string { return m.body }

// This error is returned by the FromRaw function when the passed
// message body is not RFC 2822 compliant
var ERR_MESSAGE_NOT_COMPLIANT = errors.New("Message is not RFC compliant.")

// Sets the headers and the body from a raw message. If the message
// does not contain the RFC 2822 headers (From and Date) or if it's
// uncompliant, ERR_MESSAGE_NOT_COMPLIANT error is returned.
func (m *Message) FromRaw(raw string) error {
	r := textproto.NewReader(bufio.NewReader(strings.NewReader(raw)))

	headers, err := r.ReadMIMEHeader()
	if err != nil || len(headers.Get("From")) == 0 || len(headers.Get("Date")) == 0 {
		return ERR_MESSAGE_NOT_COMPLIANT
	}

	body, err := r.ReadDotLines()
	if err != nil && err.Error() != "unexpected EOF" {
		return ERR_MESSAGE_NOT_COMPLIANT
	}

	m.headers = headers
	m.body = strings.Join(body, "\r\n")

	return nil
}

// Returns the raw message with all headers
func (m Message) Raw() string {
	return ""
}
