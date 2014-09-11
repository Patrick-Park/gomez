package gomez

import (
	"errors"
	"fmt"
	"regexp"
)

// A message represents an e-mail message and
// holds information about sender, recepients
// and the message body
type Message struct {
	rcpt []Address
	from Address
	body string
}

// Adds a new recepient to the message
func (m *Message) AddRcpt(addr Address) { m.rcpt = append(m.rcpt, addr) }

// Returns the message recepients
func (m Message) Rcpt() []Address { return m.rcpt }

// Adds a Reply-To address
func (m *Message) AddFrom(addr Address) { m.from = addr }

// Returns the Reply-To address
func (m Message) From() Address { return m.from }

// Sets the message body
func (m *Message) AddBody(msg string) { m.body = msg }

// Returns the message body
func (m Message) Body() string { return m.body }

// An Address is a structure used to hold e-mail
// addresses used to communicate
type Address struct {
	Name       string
	User, Host string
}

// Implements the Stringer interface for pretty printing
func (a Address) String() string { return fmt.Sprintf(`"%s" <%s@%s>`, a.Name, a.User, a.Host) }

var (
	pathFormat    = regexp.MustCompile("^\"([a-zA-Z ]{1,})\" <([a-zA-Z1-9]{1,})@([a-zA-Z1-9.]{4,})>$")
	ErrBadAddress = errors.New("Supplied address is invalid")
)

// Attempts to parse a string and return a new Address
// representation of it.
func NewAddress(addr string) (Address, error) {
	if !pathFormat.MatchString(addr) {
		return Address{}, ErrBadAddress
	}

	matches := pathFormat.FindStringSubmatch(addr)
	return Address{matches[1], matches[2], matches[3]}, nil
}
