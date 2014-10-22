package smtp

import (
	"net/mail"

	"github.com/gbbr/gomez/mailbox"
)

type MockSMTPServer struct {
	Run_      func(*Client, string) error
	Digest_   func(*Client) error
	Settings_ func() Config
	Query_    func(*mail.Address) mailbox.QueryResult
}

func (h MockSMTPServer) Run(c *Client, m string) error {
	return h.Run_(c, m)
}

func (h MockSMTPServer) Digest(c *Client) error {
	return h.Digest_(c)
}

func (h MockSMTPServer) Settings() Config {
	return h.Settings_()
}

func (h MockSMTPServer) Query(addr *mail.Address) mailbox.QueryResult {
	return h.Query_(addr)
}
