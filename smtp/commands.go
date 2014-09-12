package smtp

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	commandFormat = regexp.MustCompile("^([a-zA-Z]{4})(?:[ ](.*))?$")
	badCommand    = Reply{502, "5.5.2 Error: command not recoginized"}
)

// A Command is an SMTP supported command where
// Action is a function that takes the command's
// parameters and returns a valid SMTP Reply and
// SupportedMode is the supported InputMode in which
// the Command is allowed to run
type Command struct {
	Action        func(string) Reply
	SupportedMode InputMode
	ReplyInvalid  Reply
	server        *Server
}

// Reply is an SMTP reply. It contains a status
// code and a message.
type Reply struct {
	Code int
	Msg  string
}

// Implements the Stringer interface for pretty printing
func (r Reply) String() string { return fmt.Sprintf("%d - %s", r.Code, r.Msg) }

// Map of supported commands. The server's command specification.
type CommandSpec struct {
	commands map[string]Command
	server   *Server
}

// Registers a new SMTP command on the CommandSpec
func (cs *CommandSpec) Register(name string, cmd Command) {
	cmd.server = cs.server
	cs.commands[name] = cmd
}

// Runs a message from the command spec. It should include a valid
// SMTP command and optional parameters
func (cs CommandSpec) Run(msg string) Reply {
	if !commandFormat.MatchString(msg) {
		return badCommand
	}

	parts := commandFormat.FindStringSubmatch(msg)
	cmd, params := parts[1], strings.Trim(parts[2], " ")

	command, ok := cs.commands[cmd]
	if !ok {
		return badCommand
	}

	// Check if command can be run in the current server mode
	if command.SupportedMode != MODE_FREE && command.SupportedMode != cs.server.Mode() {
		return command.ReplyInvalid
	}

	return command.Action(params)
}
