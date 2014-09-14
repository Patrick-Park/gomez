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
// a Reply. The command may only run when the client is in the
// supported InputMode, otherwise it returns the invalid reply.
type Command struct {
	Action        func(*Client, string) Reply
	SupportedMode InputMode
	ReplyInvalid  Reply
}

// Map of supported commands. The server's command specification.
type CommandSpec map[string]Command

// Returns and registers commands with a new
// command spec
func NewCommandSpec() *CommandSpec {
	return &CommandSpec{
		"HELO": Command{
			Action:        cmdHELO,
			SupportedMode: MODE_HELO,
			ReplyInvalid:  Reply{1, "We've already met"},
		},
		"EHLO": Command{
			Action:        cmdEHLO,
			SupportedMode: MODE_HELO,
			ReplyInvalid:  Reply{1, "We've already met"},
		},
		"MAIL": Command{
			Action:        cmdMAIL,
			SupportedMode: MODE_MAIL,
			ReplyInvalid:  Reply{1, "We've already met"},
		},
		"RCPT": Command{
			Action:        cmdRCPT,
			SupportedMode: MODE_RCPT,
			ReplyInvalid:  Reply{1, "We've already met"},
		},
		"DATA": Command{
			Action:        cmdDATA,
			SupportedMode: MODE_RCPT,
			ReplyInvalid:  Reply{1, "We've already met"},
		},
		"RSET": Command{
			Action:        cmdRSET,
			SupportedMode: MODE_FREE,
			ReplyInvalid:  Reply{1, "We've already met"},
		},
		"NOOP": Command{
			Action:        cmdNOOP,
			SupportedMode: MODE_FREE,
			ReplyInvalid:  Reply{1, "We've already met"},
		},
		"VRFY": Command{
			Action:        cmdVRFY,
			SupportedMode: MODE_FREE,
			ReplyInvalid:  Reply{1, "We've already met"},
		},
	}
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
