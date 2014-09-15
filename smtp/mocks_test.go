package smtp

var _ Host = new(MockHost)

type MockHost struct {
	Run_    func(*Client, string)
	Digest_ func(*Client) error
}

func (h MockHost) Run(c *Client, m string) {
	if h.Run_ != nil {
		h.Run_(c, m)
	}
}

func (h MockHost) Digest(c *Client) error {
	if h.Digest_ != nil {
		return h.Digest_(c)
	}

	return nil
}
