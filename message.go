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
	Headers textproto.MIMEHeader
	body    string
}

// Adds a new recepient to the message
func (m *Message) AddRcpt(addr ...Address) { m.rcpt = append(m.rcpt, addr...) }

// Returns the message recepients
func (m Message) Rcpt() []Address { return m.rcpt }

// Adds a Return-Path address
func (m *Message) SetFrom(addr Address) { m.from = addr }

// Gets the Return-Path address
func (m Message) From() Address { return m.from }

// Sets the message body
func (m *Message) SetBody(msg string) { m.body = msg }

// Returns the message body
func (m Message) Body() string { return m.body }

// Prepends a message to an existing key if it already exists, otherwise
// it creates it. Some headers need to be in reverse order for validity,
// such as the "Recieved" key.
func (m *Message) AddHeader(key, value string) {
	if m.Headers == nil {
		m.Headers = make(textproto.MIMEHeader)
	}

	if _, ok := m.Headers[key]; !ok {
		m.Headers.Set(key, value)
	} else {
		m.Headers[key] = append([]string{value}, m.Headers[key]...)
	}
}

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

	m.Headers = headers
	m.body = strings.Join(body, "\r\n")

	return nil
}

// Returns the raw message with all headers
func (m Message) Raw() string {
	return ""
}
