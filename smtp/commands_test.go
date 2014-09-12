package smtp

import (
	"reflect"
	"testing"
)

func TestCommandSpec(t *testing.T) {

	testCases := []struct {
		// Test server mode
		ServerMode InputMode

		// Name of command to register
		CmdName string

		// Name of test message
		Message string

		// Actual command
		Command Command

		// Expected reply
		Reply Reply
	}{
		{
			MODE_HELO,
			"EHLO",
			"EHLO", Command{makeAction("EHLO"), MODE_FREE, Reply{1, "Invalid"}},
			Reply{0, "EHLO:"},
		},
		{
			MODE_HELO,
			"RCPT",
			"RCPT world", Command{makeAction("RCOT"), MODE_RCPT, Reply{2, "Invalid"}},
			Reply{2, "Invalid"},
		},
		{
			MODE_HELO,
			"ABCD",
			"ABCD   wo rld", Command{makeAction("ABCD"), MODE_HELO, Reply{3, "Invalid"}},
			Reply{0, "ABCD:wo rld"},
		},
		{
			MODE_HELO,
			"RCOT",
			"RCOT bad mode", Command{makeAction("RCOT"), MODE_RCPT, Reply{4, "Invalid"}},
			Reply{5, "Invalid"},
		},
	}

	cs := NewCommandSpec(&Server{mode: MODE_FREE})

	// Register all commands
	for _, test := range testCases {
		cs.Register(test.CmdName, test.Command)
		if _, ok := cs.commands[test.CmdName]; !ok {
			t.Errorf("Was expecting to register %s, but did not.", test.CmdName)
		}
	}

	// Run all tests
	for _, test := range testCases {
		cs.server.mode = test.ServerMode

		rpl := cs.Run(test.Message)
		if !reflect.DeepEqual(rpl, test.Reply) {
			t.Errorf(`Expected "%s" but got "%s".`, test.Reply, rpl)
		}
	}
}

// Dynamically constructs a command's action mock
// returning caller information
func makeAction(cmd string) func(*Server, string) Reply {
	return func(s *Server, param string) Reply {
		return Reply{0, cmd + ":" + param}
	}
}
