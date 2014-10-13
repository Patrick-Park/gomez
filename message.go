package gomez

import (
	"net/mail"
	"strings"
)

// A message represents an e-mail message and  holds information about
// sender, recepients and the message body
type Message struct {
	Id   uint64
	from *mail.Address
	rcpt []*mail.Address
	Raw  string
}

// Adds a new recepient to the message
func (m *Message) AddRcpt(addr ...*mail.Address) { m.rcpt = append(m.rcpt, addr...) }

// Returns the message recepients
func (m Message) Rcpt() []*mail.Address { return m.rcpt }

// Adds a Return-Path address
func (m *Message) SetFrom(addr *mail.Address) { m.from = addr }

// Gets the Return-Path address
func (m Message) From() *mail.Address { return m.from }

// Returns the mail.Message object, containing the parsed headers
// and the Body as an io.Reader to read from
func (m Message) Parse() (*mail.Message, error) {
	return mail.ReadMessage(strings.NewReader(m.Raw))
}

// Prepends a header to the message. If a multiline message is desired,
// separate lines using <CR><LF> as specified in RFC. Go has \r\n for this.
func (m *Message) PrependHeader(name, value string) {
	m.Raw = name + ": " + value + "\r\n" + m.Raw
}

// Makes an AddressList string parseable by mail.ParseAddressList
func MakeAddressList(list []*mail.Address) string {
	var r string

	for i := 0; i < len(list); i++ {
		r += list[i].String()
		if i < len(list)-1 {
			r += ", "
		}
	}

	return r
}
