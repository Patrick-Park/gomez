package smtp

import "github.com/gbbr/gomez/post"

type Config struct {
	Mailbox post.Mailbox
	Port    int
}
