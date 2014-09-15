package smtp

import (
	"fmt"
	"regexp"
)

var (
	commandFormat = regexp.MustCompile("^([a-zA-Z]{4})(?:[ ](.*))?$")
	badCommand    = Reply{502, "5.5.2 Error: command not recoginized"}
)

// Reply is an SMTP reply. It contains a status
// code and a message.
type Reply struct {
	Code int
	Msg  string
}

// Implements the Stringer interface for pretty printing
func (r Reply) String() string { return fmt.Sprintf("%d %s", r.Code, r.Msg) }

// A Command is an SMTP supported command that performs
// an action in the context of the calling client and returns
// a Reply.
type Action func(*Client, string) Reply

// Map of supported commands. The server's command specification.
type CommandSpec map[string]Action
