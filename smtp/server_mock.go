package smtp

import (
	"net/mail"

	"github.com/gbbr/gomez/mailbox"
	"github.com/gbbr/jamon"
)

type mockHost struct {
	Run_      func(*transaction, string) error
	Digest_   func(*transaction) error
	Settings_ func() jamon.Group
	QueryMock func(*mail.Address) mailbox.QueryResult
}

func (h mockHost) run(c *transaction, m string) error {
	return h.Run_(c, m)
}

func (h mockHost) digest(c *transaction) error {
	return h.Digest_(c)
}

func (h mockHost) settings() jamon.Group {
	return h.Settings_()
}

func (h mockHost) query(addr *mail.Address) mailbox.QueryResult {
	return h.QueryMock(addr)
}
