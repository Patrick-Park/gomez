package smtp

var _ Child = new(MockChild)

type MockChild struct {
	Serve_ func()
	Reset_ func()
	Reply_ func(Reply)
}

func (c MockChild) Serve() {
	if c.Serve_ != nil {
		c.Serve_()
	}
}

func (c MockChild) Reset() {
	if c.Reset_ != nil {
		c.Reset_()
	}
}

func (c MockChild) Reply(r Reply) {
	if c.Reply_ != nil {
		c.Reply_(r)
	}
}

var _ Host = new(MockHost)

type MockHost struct {
	Run_    func(Child, string)
	Digest_ func(Child) error
}

func (h MockHost) Run(c Child, m string) {
	if h.Run_ != nil {
		h.Run_(c, m)
	}
}

func (h MockHost) Digest(c Child) error {
	if h.Digest_ != nil {
		return h.Digest_(c)
	}

	return nil
}
