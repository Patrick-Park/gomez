package smtp

import (
	"reflect"
	"testing"
)

// Ensures that CommandSpec.Run runs the correction Action
// with the correct parameters and returns the invalid reply
// when the command is inappropriate for the Client's InputMode
func TestCommandSpec(t *testing.T) {
	testSpec := CommandSpec{
		"EHLO": Command{makeAction("EHLO"), MODE_FREE, Reply{1, "Invalid"}},
		"RCPT": Command{makeAction("RCPT"), MODE_RCPT, Reply{2, "Invalid"}},
		"ABCD": Command{makeAction("ABCD"), MODE_HELO, Reply{3, "Invalid"}},
		"RCOT": Command{makeAction("RCOT"), MODE_RCPT, Reply{4, "Invalid"}},
		"CMND": Command{makeAction("CMND"), MODE_RCPT, Reply{4, "Invalid"}},
	}

	testCases := []struct {
		ClientMode InputMode // Server mode for current test
		Message    string    // Message to run
		Reply      Reply     // Expected reply
	}{
		{MODE_HELO, "EHLO", Reply{0, "EHLO:"}},
		{MODE_DATA, "RCPT world", Reply{2, "Invalid"}},
		{MODE_HELO, "ABCD   wo rld", Reply{0, "ABCD:wo rld"}},
		{MODE_HELO, "RCOT bad mode", Reply{4, "Invalid"}},
		{MODE_RCPT, "SOME thing else", badCommand},
	}

	// Run all tests
	for _, test := range testCases {
		rpl := testSpec.Run(&Client{Mode: test.ClientMode}, test.Message)
		if !reflect.DeepEqual(rpl, test.Reply) {
			t.Errorf(`Expected "%s" but got "%s".`, test.Reply, rpl)
		}
	}

	if testSpec.Run(&Client{}, "INVALID") != badCommand {
		t.Error("Matched regexp on invalid command")
	}
}

// Dynamically constructs a command's action mock
// returning caller information
func makeAction(cmd string) func(*Client, string) Reply {
	return func(c *Client, param string) Reply {
		return Reply{0, cmd + ":" + param}
	}
}
