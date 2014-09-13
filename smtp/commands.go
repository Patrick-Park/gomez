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
	Action        func(*Client, string) Reply
	SupportedMode InputMode
	ReplyInvalid  Reply
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
type CommandSpec map[string]Command

// Registers a new SMTP command on the CommandSpec
func (cs *CommandSpec) Register(name string, cmd Command) { (*cs)[name] = cmd }

// Returns and registers commands with a new
// command spec
func NewCommandSpec() *CommandSpec {
	cs := make(CommandSpec)

	// Register

	return &cs
}

// Runs a message from the command spec in the context of a
// given client
func (cs CommandSpec) Run(c *Client, msg string) Reply {
	if !commandFormat.MatchString(msg) {
		return badCommand
	}

	parts := commandFormat.FindStringSubmatch(msg)
	cmd, params := parts[1], strings.Trim(parts[2], " ")

	command, ok := cs[cmd]
	if !ok {
		return badCommand
	}

	// Check if command can be run in the current server mode
	if command.SupportedMode != MODE_FREE && command.SupportedMode != c.Mode {
		return command.ReplyInvalid
	}

	return command.Action(c, params)
}
