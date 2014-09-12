package smtp

import (
	"github.com/gbbr/gomez"
)

// Configuration settings for the SMTP server
type Config struct {
	Port    int
	Mailbox gomez.Mailbox
}

type InputMode int

const (
	MODE_FREE InputMode = iota
	MODE_HELO
	MODE_MAIL
	MODE_RCPT
	MODE_DATA
)

type Server struct {
	mode InputMode
	cs   *CommandSpec
	mb   gomez.Mailbox
	msg  *gomez.Message
}

func NewServer(conf Config) *Server {
	s := &Server{
		mode: MODE_HELO,
		mb:   conf.Mailbox,
	}

	s.cs = NewCommandSpec(s)
	s.cs.Register("HELO", Command{
		Action:        func(s *Server, a string) Reply { return Reply{} },
		SupportedMode: MODE_HELO,
		ReplyInvalid:  Reply{},
	})

	return s
}

func (s Server) Mode() InputMode { return s.mode }

func (s *Server) Reset() {
	s.mode = MODE_HELO
	s.msg = new(gomez.Message)
}
