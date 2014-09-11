package smtp

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	commandFormat = regexp.MustCompile("^([a-zA-Z]{4})(?: ?)(.*)?$")
	badCommand    = Reply{502, "5.5.2 Error: command not recoginized"}
)

// Command is a function that takes a parameter
// and returns a Reply. It represents an SMTP command
// executor.
type Command func(string) Reply

// Reply is an SMTP reply. It contains a status
// code and a message.
type Reply struct {
	Code int
	Msg  string
}

// Implements the Stringer interface for pretty printing
func (r Reply) String() string { return fmt.Sprintf("%d - %s", r.Code, r.Msg) }

// Map of supported commands. The server's command specification.
// Because it is a map, we initialize a new command spec using make:
// 	cs := make(CommandSpec)
type CommandSpec map[string]Command

// Registers a new SMTP command on the CommandSpec
func (cs *CommandSpec) Register(cmd string, action Command) { (*cs)[cmd] = action }

// Runs a message from the command spec. It should include a valid
// SMTP command and optional parameters
func (cs CommandSpec) Run(msg string) Reply {
	if !commandFormat.MatchString(msg) {
		return badCommand
	}

	parts := commandFormat.FindStringSubmatch(msg)
	cmd, params := parts[1], strings.Trim(parts[2], " ")

	action, ok := cs[cmd]
	if !ok {
		return badCommand
	}

	return action(params)
}
