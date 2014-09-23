package gomez

import (
	"bufio"
	"bytes"
	"errors"
	"net/textproto"
	"strings"
)

// OrderedHeader is a wrapper on textproto.MIMEHeader providing additional
// functionalities, such as ordered headers (as needed for "Received")
type OrderedHeader struct{ textproto.MIMEHeader }

// Prepends a message to an existing key if it already exists, otherwise
// it creates it. Some headers need to be in reverse order for validity.
func (h OrderedHeader) Prepend(key, value string) {
	if _, ok := h.MIMEHeader[key]; !ok {
		h.MIMEHeader.Set(key, value)
	} else {
		h.MIMEHeader[key] = append([]string{value}, h.MIMEHeader[key]...)
	}
}

// A message represents an e-mail message and  holds information about
// sender, recepients and the message body
type Message struct {
	from    Address
	rcpt    []Address
	Headers OrderedHeader
	Body    string
}

// Creates a new empty message with all values initialized
func NewMessage() *Message {
	return &Message{Headers: OrderedHeader{make(textproto.MIMEHeader)}}
}

// Adds a new recepient to the message
func (m *Message) AddRcpt(addr ...Address) { m.rcpt = append(m.rcpt, addr...) }

// Returns the message recepients
func (m Message) Rcpt() []Address { return m.rcpt }

// Adds a Return-Path address
func (m *Message) SetFrom(addr Address) { m.from = addr }

// Gets the Return-Path address
func (m Message) From() Address { return m.from }

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

	m.Headers.MIMEHeader = headers
	m.Body = strings.Join(body, "\r\n")
	m.Body = strings.TrimLeft(m.Body, "\r\n")

	return nil
}

// Returns the raw message with all headers. The order of the keys
// will vary on each call but the order of the values will be always
// be as is.
func (m Message) Raw() string {
	var raw bytes.Buffer

	for key, values := range m.Headers.MIMEHeader {
		for _, value := range values {
			raw.WriteString(key + ": " + value + "\r\n")
		}
	}

	raw.WriteString("\r\n\r\n" + m.Body)

	return raw.String()
}
