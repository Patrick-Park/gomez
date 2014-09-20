package smtp

import (
	"log"
	"strings"

	"github.com/gbbr/gomez"
)

// SMTP HELO command: initiates handshake by
// identifying foreign host
func cmdHELO(ctx *Client, param string) error {
	if len(param) == 0 {
		return ctx.Notify(Reply{501, "Syntax: HELO hostname"})
	}

	ctx.Id = param
	ctx.Mode = MODE_MAIL

	return ctx.Notify(Reply{250, "Gomez SMTPd"})
}

// SMTP EHLO command: same as HELO but provides information
// about server capabilities and support
func cmdEHLO(ctx *Client, param string) error {
	if len(param) == 0 {
		return ctx.Notify(Reply{501, "Syntax: EHLO hostname"})
	}

	ctx.Id = param
	ctx.Mode = MODE_MAIL

	return ctx.Notify(Reply{250, "Gomez SMTPd\nVRFY"})
}

// SMTP MAIL command: sets sender address.
// Format is: MAIL FROM:<address>
func cmdMAIL(ctx *Client, param string) error {
	switch {

	case ctx.Mode == MODE_HELO:
		return ctx.Notify(Reply{503, "5.5.1 Say HELO/EHLO first."})

	case ctx.Mode > MODE_MAIL:
		return ctx.Notify(Reply{503, "5.5.1 Error: nested MAIL command"})

	case !strings.HasPrefix(strings.ToUpper(param), "FROM:"):
		return ctx.Notify(Reply{501, "5.5.4 Syntax: MAIL FROM:<address>"})

	}

	addr, err := gomez.NewAddress(param[len("FROM:"):])
	if err != nil {
		return ctx.Notify(Reply{501, "5.1.7 Bad sender address syntax"})
	}

	ctx.msg.SetFrom(addr)
	ctx.Mode = MODE_RCPT

	return ctx.Notify(Reply{250, "2.1.0 Ok"})
}

// SMTP RCPT command: set receipient. Can be executed
// multiple times for multiple receipients.
// Format: RCPT TO:<address>
func cmdRCPT(ctx *Client, param string) error {
	switch {

	case ctx.Mode == MODE_HELO:
		return ctx.Notify(Reply{503, "5.5.1 Say HELO/EHLO first."})

	case ctx.Mode == MODE_MAIL:
		return ctx.Notify(Reply{503, "5.5.1 Error: need MAIL command"})

	case !strings.HasPrefix(strings.ToUpper(param), "TO:"):
		return ctx.Notify(Reply{501, "5.5.4 Syntax: RCPT TO:<address>"})

	}

	// Is the address syntax correct?
	addr, err := gomez.NewAddress(param[len("TO:"):])
	if err != nil {
		return ctx.Notify(Reply{501, "5.1.7 Bad recipient address syntax"})
	}

	flags := ctx.Host.Settings()

	switch ctx.Host.Query(addr) {

	case gomez.QUERY_STATUS_NOT_FOUND:
		return ctx.Notify(Reply{550, "No such user here."})

	case gomez.QUERY_STATUS_NOT_LOCAL:
		if !flags.Relay {
			return ctx.Notify(Reply{550, "No such user here. Relaying is not supported."})
		}

		ctx.msg.AddRcpt(addr)
		ctx.Mode = MODE_DATA

		return ctx.Notify(Reply{251, "User not local; will forward to <forward-path>"})

	case gomez.QUERY_STATUS_SUCCESSFUL:
		ctx.msg.AddRcpt(addr)
		ctx.Mode = MODE_DATA

		return ctx.Notify(Reply{250, "OK"})

	}

	return ctx.Notify(Reply{451, "Requested action aborted: error in processing"})
}

// SMTP DATA command: sets the message body. Can
// also include headers as per RFC 821 / RFC 2821
// Ends with <CR>.<CR>
func cmdDATA(ctx *Client, param string) error {
	msg, err := ctx.conn.ReadDotLines()
	if err != nil {
		log.Printf("Could not read dot lines: %s\n", err)
	}

	ctx.msg.SetBody(strings.Join(msg, ""))
	ctx.Host.Digest(ctx)
	ctx.Reset()

	return ctx.Notify(Reply{250, "Message queued"})
}

// SMTP RSET command: reset client state
// and message
func cmdRSET(ctx *Client, param string) error {
	ctx.Reset()
	return ctx.Notify(Reply{250, "2.0.0 Ok"})
}

// SMTP NOOP command: no operation
func cmdNOOP(ctx *Client, param string) error {
	return ctx.Notify(Reply{250, "2.0.0 Ok"})
}

// SMTP VRFY command: queries the mailbox
// for a user lookup. If more than one entry
// is found, ambigous error is returned
func cmdVRFY(ctx *Client, param string) error {
	return nil
}

func cmdQUIT(ctx *Client, param string) error {
	ctx.Mode = MODE_QUIT
	return ctx.Notify(Reply{221, "2.0.0 Adeus"})
}
