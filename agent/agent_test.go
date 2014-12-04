package agent

import (
	"net"
	"testing"

	"github.com/gbbr/gomez/mailbox"
)

func TestAgent_deliverTo(t *testing.T) {
	lookupMX = func(host string) []*net.MX {
		return []*net.MX{{Host: "localhost"}}
	}
	deliverTo("google.com", mailbox.Delivery{})
}
