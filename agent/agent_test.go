package agent

import (
	"net"
	"testing"

	"github.com/gbbr/gomez/mailbox"
)

func TestAgent_deliverTo(t *testing.T) {
	var cron cronJob
	lookupMX = func(host string) []*net.MX {
		return []*net.MX{{Host: "localhost"}}
	}
	cron.deliverTo("google.com", mailbox.Delivery{})
}
