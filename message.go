package gomez

// A message represents an e-mail message and
// holds information about sender, recepients
// and the message body
type Message struct {
	from Address
	rcpt []Address
	body string
}

// Adds a new recepient to the message
func (m *Message) AddRcpt(addr ...Address) { m.rcpt = append(m.rcpt, addr...) }

// Returns the message recepients
func (m Message) Rcpt() []Address { return m.rcpt }

// Adds a Reply-To address
func (m *Message) SetFrom(addr Address) { m.from = addr }

// Returns the Reply-To address
func (m Message) From() Address { return m.from }

// Sets the message body
func (m *Message) SetBody(msg string) { m.body = msg }

// Returns the message body
func (m Message) Body() string { return m.body }
