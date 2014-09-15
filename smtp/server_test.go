package smtp

import (
	// "log"
	"net"
	"net/textproto"
	"testing"
)

func TestServerRun(t *testing.T) {
	var passedMsg string

	srv := &Server{
		spec: &CommandSpec{
			"HELO": func(ctx *Client, params string) {
				passedMsg = params
				ctx.Reply(Reply{100, "Hi"})
			},
		},
	}

	cc, sc := net.Pipe()
	cconn, sconn := textproto.NewConn(cc), textproto.NewConn(sc)

	testClient := &Client{
		Mode: MODE_HELO,
		conn: sconn,
	}

	testCases := []struct {
		Message        string
		Reponse, Reply string
	}{
		{"BADFORMAT", "", badCommand.String()},
		{"GOOD FORMAT", "", badCommand.String()},
		{"HELO  world ", "world", "100 Hi"},
		{"HELO", "", "100 Hi"},
	}

	for _, test := range testCases {
		passedMsg = ""

		go srv.Run(testClient, test.Message)
		rpl, err := cconn.ReadLine()
		if err != nil {
			t.Errorf("Error reading response %s", err)
		}

		if passedMsg != test.Reponse || rpl != test.Reply {
			t.Errorf("Expected params (%s) and message (%s), got (%s) and (%s).", test.Reponse, test.Reply, passedMsg, rpl)
		}
	}
}
