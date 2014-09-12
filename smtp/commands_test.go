package smtp

import "testing"

func TestCommandSpec(t *testing.T) {
	cs := make(CommandSpec, 10)

	testCases := []struct {
		Msg string
		Exp Reply
	}{
		{"EHLO world", Reply{0, "EHLO:world"}},
		{"EHLO    world   ", Reply{0, "EHLO:world"}},
		{"EHLO world how are you  ", Reply{0, "EHLO:world how are you"}},
		{"EHLO", Reply{0, "EHLO:"}},
		{"HELO", Reply{0, "HELO:"}},
		{"HELO  WAZZAAA", Reply{0, "HELO:WAZZAAA"}},
		{"AEHLO", badCommand},
		{"", badCommand},
		{"BADC", badCommand},
		{"BAD COMMAND", badCommand},
	}

	// Test command that echoes params
	makeTestAction := func(cmd string) Command {
		return func(params string) Reply {
			return Reply{0, cmd + ":" + params}
		}
	}

	cs.Register("HELO", makeTestAction("HELO"))
	if _, ok := cs["HELO"]; !ok {
		t.Error("Did not register command correctly")
	}

	cs.Register("EHLO", makeTestAction("EHLO"))
	if _, ok := cs["EHLO"]; !ok {
		t.Error("Did not register command correctly")
	}

	if len(cs) != 2 {
		t.Error("Did not register correct number of commands")
	}

	for _, test := range testCases {
		reply := cs.Run(test.Msg)

		if reply.Msg != test.Exp.Msg {
			t.Errorf("At test %s expected response %#v, but got %#v.", test.Msg, test.Exp, reply)
		}
	}
}
