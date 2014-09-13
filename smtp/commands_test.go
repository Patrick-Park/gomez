package smtp

import (
	"reflect"
	"testing"
)

func TestCommandSpec(t *testing.T) {
	testCases := []struct {
		ClientMode InputMode // Server mode for current test
		CmdName    string    // Name of command to register
		Command    Command   // Actual command
		Message    string    // Message to run
		Reply      Reply     // Expected reply
	}{
		{
			MODE_HELO,
			"EHLO", Command{makeAction("EHLO"), MODE_FREE, Reply{1, "Invalid"}},
			"EHLO",
			Reply{0, "EHLO:"},
		},
		{
			MODE_DATA,
			"RCPT", Command{makeAction("RCPT"), MODE_RCPT, Reply{2, "Invalid"}},
			"RCPT world",
			Reply{2, "Invalid"},
		},
		{
			MODE_HELO,
			"ABCD", Command{makeAction("ABCD"), MODE_HELO, Reply{3, "Invalid"}},
			"ABCD   wo rld",
			Reply{0, "ABCD:wo rld"},
		},
		{
			MODE_HELO,
			"RCOT", Command{makeAction("RCOT"), MODE_RCPT, Reply{4, "Invalid"}},
			"RCOT bad mode",
			Reply{4, "Invalid"},
		},
	}

	cs := NewCommandSpec()

	// Register all commands
	for _, test := range testCases {
		cs.Register(test.CmdName, test.Command)
		if _, ok := (*cs)[test.CmdName]; !ok {
			t.Errorf("Was expecting to register %s, but did not.", test.CmdName)
		}
	}

	// Run all tests
	for _, test := range testCases {
		rpl := cs.Run(&Client{Mode: test.ClientMode}, test.Message)
		if !reflect.DeepEqual(rpl, test.Reply) {
			t.Errorf(`Expected "%s" but got "%s".`, test.Reply, rpl)
		}
	}
}

// Dynamically constructs a command's action mock
// returning caller information
func makeAction(cmd string) func(*Client, string) Reply {
	return func(c *Client, param string) Reply {
		return Reply{0, cmd + ":" + param}
	}
}
