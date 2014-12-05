package agent

import (
	"net"
	"testing"

	"github.com/gbbr/gomez/mailbox"
)

func TestAgent_deliverTo(t *testing.T) {
	return
	var cron cronJob
	lookupMX = func(host string) ([]*net.MX, error) {
		return []*net.MX{{Host: "localhost"}}, nil
	}
	cron.deliverTo("google.com", mailbox.Package{})
}
