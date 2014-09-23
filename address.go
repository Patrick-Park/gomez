package gomez

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// An Address is a structure used to hold e-mail
// addresses used to communicate
type Address struct {
	Name       string
	User, Host string
}

// Implements the Stringer interface for pretty printing
func (a Address) String() string { return fmt.Sprintf(`%s <%s@%s>`, a.Name, a.User, a.Host) }

// Address format, can be <user_name123@host.tld> or First Last <user@host.tld>
var pathFormat = regexp.MustCompile("^(?:([a-zA-Z ]{1,})(?: ))?<([a-zA-Z1-9_]{1,})@([a-zA-Z1-9.]{4,})>$")

// Attempts to parse a string and return a new Address
// representation of it.
func NewAddress(addr string) (Address, error) {
	if !pathFormat.MatchString(addr) {
		return Address{}, errors.New("Supplied address is invalid")
	}

	matches := pathFormat.FindStringSubmatch(addr)
	return Address{strings.Trim(matches[1], " "), matches[2], matches[3]}, nil
}
