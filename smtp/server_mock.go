package smtp

import (
	"net/mail"

	"github.com/gbbr/gomez"
)

type MockMailService struct {
	Run_      func(*Client, string) error
	Digest_   func(*Client) error
	Settings_ func() Config
	Query_    func(*mail.Address) gomez.QueryStatus
}

func (h MockMailService) Run(c *Client, m string) error {
	if h.Run_ != nil {
		return h.Run_(c, m)
	}

	return nil
}

func (h MockMailService) Digest(c *Client) error {
	if h.Digest_ != nil {
		return h.Digest_(c)
	}

	return nil
}

func (h MockMailService) Settings() Config {
	if h.Settings_ != nil {
		return h.Settings_()
	}

	return Config{}
}

func (h MockMailService) Query(addr *mail.Address) gomez.QueryStatus {
	if h.Query_ != nil {
		return h.Query_(addr)
	}

	return gomez.QUERY_STATUS_ERROR
}
