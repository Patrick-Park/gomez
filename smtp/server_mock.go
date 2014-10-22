package smtp

import (
	"net/mail"

	"github.com/gbbr/gomez/mailbox"
)

type mockHost struct {
	Run_      func(*transaction, string) error
	Digest_   func(*transaction) error
	Settings_ func() Config
	Query_    func(*mail.Address) mailbox.QueryResult
}

func (h mockHost) run(c *transaction, m string) error {
	return h.Run_(c, m)
}

func (h mockHost) digest(c *transaction) error {
	return h.Digest_(c)
}

func (h mockHost) settings() Config {
	return h.Settings_()
}

func (h mockHost) query(addr *mail.Address) mailbox.QueryResult {
	return h.Query_(addr)
}
