package smtp

import (
	"reflect"
	"testing"
)

func TestServerRun(t *testing.T) {
	var (
		passedMsg string
		passedRpl Reply
		testChild Child
	)

	testChild = &MockChild{
		Reply_: func(r Reply) {
			passedRpl = r
		},
	}

	srv := &Server{
		spec: &CommandSpec{
			"HELO": func(ctx Child, params string) Reply {
				passedMsg = params
				return Reply{}
			},
		},
	}

	passedMsg, passedRpl = "", Reply{}
	srv.Run(testChild, "BADFORMAT")

	if passedMsg != "" || !reflect.DeepEqual(passedRpl, badCommand) {
		t.Errorf("Expected to pass params '%s' and get no reply but instead got reply: '%s', and params: '%s'", badCommand, passedRpl, passedMsg)
	}

	passedMsg, passedRpl = "", Reply{}
	srv.Run(testChild, "GOOD FORMAT but doesn't exist")

	if passedMsg != "" || !reflect.DeepEqual(passedRpl, badCommand) {
		t.Errorf("Expected to pass params '%s' and get no reply but instead got reply: '%s', and params: '%s'", badCommand, passedRpl, passedMsg)
	}

	passedMsg, passedRpl = "", Reply{}
	srv.Run(testChild, "HELO  world ")

	if passedMsg != "world" || passedRpl.String() != "0 " {
		t.Errorf("Expected to pass params and get no reply but instead got reply: '%s', and params: '%s'", passedRpl, passedMsg)
	}
}
