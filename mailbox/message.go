package mailbox

import (
	"fmt"
	"net/mail"
	"strings"
)

// A Message represents an e-mail message and  holds information about
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

// From retrieves the message Return-Path.
func (m Message) From() *mail.Address { return m.from }

// Inbound retrieves local recipients.
func (m Message) Inbound() []*mail.Address { return m.rcptIn }

// Outbound receives all out-bound recipients.
func (m Message) Outbound() []*mail.Address { return m.rcptOut }

// SetFrom adds a Return-Path given an address.
func (m *Message) SetFrom(addr *mail.Address) { m.from = addr }

// AddInbound adds a local recipient.
func (m *Message) AddInbound(rcpt *mail.Address) {
	m.rcptIn = append(m.rcptIn, rcpt)
}

// AddOutbound adds one or more remote recipients.
func (m *Message) AddOutbound(rcpt ...*mail.Address) {
	m.rcptOut = append(m.rcptOut, rcpt...)
}

// Rcpt returns a list of all recipients on this message.
func (m Message) Rcpt() []*mail.Address {
	return append(m.rcptIn, m.rcptOut...)
}

// Parse returns an object of type mail.Message with the headers parsed.
func (m Message) Parse() (*mail.Message, error) {
	return mail.ReadMessage(strings.NewReader(m.Raw))
}

// PrependHeader attaches a header at the beginning of the message. PreprendHeader
// does not validate the message. It is the responsability of the caller.
func (m *Message) PrependHeader(name, value string, params ...interface{}) {
	m.Raw = name + ": " + fmt.Sprintf(value, params...) + "\r\n" + m.Raw
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

// SplitUserHost splits an address into user and host.
func SplitUserHost(addr *mail.Address) (user string, host string) {
	p := strings.Split(addr.Address[0:len(addr.Address)], "@")
	if len(p) != 2 {
		return
	}
	return p[0], p[1]
}
