package smtp

var _ MailService = new(MockMailService)

type MockMailService struct {
	Run_    func(*Client, string)
	Digest_ func(*Client) error
}

func (h MockMailService) Run(c *Client, m string) {
	if h.Run_ != nil {
		h.Run_(c, m)
	}
}

func (h MockMailService) Digest(c *Client) error {
	if h.Digest_ != nil {
		return h.Digest_(c)
	}

	return nil
}
