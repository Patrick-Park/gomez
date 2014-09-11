package gomez

import (
	"errors"
	"fmt"
	"regexp"
)

var ErrBadAddress = errors.New("Supplied address is invalid")

// Mailbox implements message queueing and
// dequeueing system
type Mailbox interface {
	// Places a message onto the queue for delivery at
	// next pick-up
	Queue(msg Message) error

	// Retrieves all messages from the queue and empties it
	Dequeue() ([]*Message, error)

	// Attempts to deliver a message to a local mailbox
	Deliver(msg Message) error

	// Retrieves all messages for a given address and
	// empties mailbox
	Retrieve(usr Address) []*Message
}

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

// A correct path representation has the format
// "First Last" <user@host> according to RFC 2821
var pathFormat = regexp.MustCompile("^\"([a-zA-Z ]{1,})\" <([a-zA-Z1-9]{1,})@([a-zA-Z1-9.]{4,})>$")

// Implements the Stringer interface for pretty printing
func (a Address) String() string { return fmt.Sprintf(`"%s" <%s@%s>`, a.Name, a.User, a.Host) }

// Attempts to parse a string and return a new Address
// representation of it.
func NewAddress(addr string) (Address, error) {
	if !pathFormat.MatchString(addr) {
		return Address{}, ErrBadAddress
	}

	matches := pathFormat.FindStringSubmatch(addr)
	return Address{matches[1], matches[2], matches[3]}, nil
}
