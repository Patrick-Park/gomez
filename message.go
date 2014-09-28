package gomez

import (
	"errors"
	"io/ioutil"
	"net/mail"
	"strings"
)

var ERR_MESSAGE_NOT_COMPLIANT = errors.New("Message is not RFC 2822 compliant")

// A message represents an e-mail message and  holds information about
// sender, recepients and the message body
type Message struct {
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

// Retrieves the message headers.
func (m Message) Header() (mail.Header, error) {
	msg, err := mail.ReadMessage(strings.NewReader(m.Raw))
	if err != nil {
		return mail.Header{}, ERR_MESSAGE_NOT_COMPLIANT
	}

	return msg.Header, nil
}

// Retrieves the message body
func (m Message) Body() (string, error) {
	r := strings.NewReader(m.Raw)
	_, err := mail.ReadMessage(r)
	if err != nil {
		return "", ERR_MESSAGE_NOT_COMPLIANT
	}

	body, err := ioutil.ReadAll(r)
	if err != nil {
		return "", ERR_MESSAGE_NOT_COMPLIANT
	}

	return string(body), nil
}
