package gomez

import (
	"io/ioutil"
	"net/mail"
	"strings"
)

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
		return mail.Header{}, err
	}

	return msg.Header, nil
}

// Retrieves the message body
func (m Message) Body() (string, error) {
	r := strings.NewReader(m.Raw)
	_, err := mail.ReadMessage(r)
	if err != nil {
		return "", err
	}

	body, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
