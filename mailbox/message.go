package mailbox

import (
	"net/mail"
	"strings"
)

// A message represents an e-mail message and  holds information about
// sender, recepients and the message body
type Message struct {
	// ID holds the internal Message-ID.
	ID uint64

	// Raw holds the message in raw form.
	Raw string

	from    *mail.Address   // Return-Path address
	rcptIn  []*mail.Address // Inbound recipients
	rcptOut []*mail.Address // Outbount recipients
}

// AddInbound adds a local recipient.
func (m *Message) AddInbound(rcpt *mail.Address) {
	m.rcptIn = append(m.rcptIn, rcpt)
}

// Inbound retrieves local recipients.
func (m Message) Inbound() []*mail.Address { return m.rcptIn }

// AddOutbound adds a remote recipient.
func (m *Message) AddOutbound(rcpt *mail.Address) {
	m.rcptOut = append(m.rcptOut, rcpt)
}

// Outbound receives all out-bound recipients.
func (m Message) Outbound() []*mail.Address { return m.rcptOut }

// Rcpt returns a list of all recipients on this message.
func (m Message) Rcpt() []*mail.Address {
	return append(m.rcptIn, m.rcptOut...)
}

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

// makeAddressList returns a string parseable by mail.ParseAddressList
func makeAddressList(list []*mail.Address) string {
	var r string

	for i, addr := range list {
		r += addr.String()
		if i < len(list)-1 {
			r += ", "
		}
	}

	return r
}
