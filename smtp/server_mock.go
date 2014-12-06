package smtp

import (
	"net/mail"

	"github.com/gbbr/gomez/internal/jamon"
)

type mockHost struct {
	RunMock      func(*transaction, string) error
	DigestMock   func(*transaction) error
	SettingsMock func() jamon.Group
	QueryMock    func(*mail.Address) int
}

func (h mockHost) run(c *transaction, m string) error {
	return h.RunMock(c, m)
}

func (h mockHost) digest(c *transaction) error {
	return h.DigestMock(c)
}

func (h mockHost) settings() jamon.Group {
	return h.SettingsMock()
}

func (h mockHost) query(addr *mail.Address) int {
	return h.QueryMock(addr)
}
