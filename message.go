package gomez

import (
	"net/mail"
	"strings"
)

// A message represents an e-mail message and  holds information about
// sender, recepients and the message body
type Message struct {
	ID   uint64
	from *mail.Address
	rcpt []*mail.Address
	Raw  string
}

// AddRcpt adds new recepients to the message.
func (m *Message) AddRcpt(addr ...*mail.Address) { m.rcpt = append(m.rcpt, addr...) }

// Rcpt returns the message recepients.
func (m Message) Rcpt() []*mail.Address { return m.rcpt }

// SetFrom adds a Return-Path given an address.
func (m *Message) SetFrom(addr *mail.Address) { m.from = addr }

// From retrieves the message Return-Path. 
func (m Message) From() *mail.Address { return m.from }

// Parse returns the message with the headers parsed and a body reader.
func (m Message) Parse() (*mail.Message, error) {
	return mail.ReadMessage(strings.NewReader(m.Raw))
}

// PrependHeader attaches a header at the beginning of the message.
func (m *Message) PrependHeader(name, value string) {
	m.Raw = name + ": " + value + "\r\n" + m.Raw
}

// MakeAddressList returns a string parseable by mail.ParseAddressList
func MakeAddressList(list []*mail.Address) string {
	var r string

	for i, addr := range list {
		r += addr.String()
		if i < len(list)-1 {
			r += ", "
		}
	}

	return r
}
