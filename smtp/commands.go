package smtp

import (
	"io"
	"net/mail"
	"strings"

	"github.com/gbbr/gomez/mailbox"
)

// RFC 2821 4.1.1.1  Extended HELLO (EHLO) or HELLO (HELO)
func cmdHELO(ctx *transaction, param string) error {
	if len(param) == 0 {
		return ctx.notify(reply{501, "Syntax: HELO hostname"})
	}

	ctx.ID = param
	ctx.Mode = stateMAIL

	return ctx.notify(reply{250, "Gomez SMTPd"})
}

// RFC 2821 4.1.1.1  Extended HELLO (EHLO) or HELLO (HELO)
func cmdEHLO(ctx *transaction, param string) error {
	if len(param) == 0 {
		return ctx.notify(reply{501, "Syntax: EHLO hostname"})
	}

	ctx.ID = param
	ctx.Mode = stateMAIL

	return ctx.notify(reply{250, "Gomez SMTPd\nSMTPUTF8\n8BITMIME\nENHANCEDSTATUSCODES\nVRFY"})
}

// RFC 2821 4.1.1.2 MAIL (MAIL)
func cmdMAIL(ctx *transaction, param string) error {
	switch {
	case ctx.Mode == stateHELO:
		return ctx.notify(reply{503, "5.5.1 Say HELO/EHLO first."})

	case ctx.Mode > stateMAIL:
		return ctx.notify(reply{503, "5.5.1 Error: nested MAIL command"})

	case !strings.HasPrefix(strings.ToUpper(param), "FROM:"):
		return ctx.notify(reply{501, "5.5.4 Syntax: MAIL FROM:<address>"})
	}

	addr, err := mail.ParseAddress(param[len("FROM:"):])
	if err != nil {
		return ctx.notify(reply{501, "5.1.7 Bad sender address syntax"})
	}

	ctx.Message.SetFrom(addr)
	ctx.Mode = stateRCPT

	return ctx.notify(reply{250, "2.1.0 Ok"})
}

// RFC 2821 4.1.1.3 RECIPIENT (RCPT)
func cmdRCPT(ctx *transaction, param string) error {
	switch {
	case ctx.Mode == stateHELO:
		return ctx.notify(reply{503, "5.5.1 Say HELO/EHLO first."})

	case ctx.Mode == stateMAIL:
		return ctx.notify(reply{503, "5.5.1 Error: need MAIL command"})

	case !strings.HasPrefix(strings.ToUpper(param), "TO:"):
		return ctx.notify(reply{501, "5.5.4 Syntax: RCPT TO:<address>"})
	}

	addr, err := mail.ParseAddress(param[len("TO:"):])
	if err != nil {
		return ctx.notify(reply{501, "5.1.7 Bad recipient address syntax"})
	}

	switch ctx.host.query(addr) {
	case mailbox.QueryNotFound:
		return ctx.notify(reply{550, "5.1.1 No such user here."})

	case mailbox.QueryNotLocal:
		if flags := ctx.host.settings(); !flags.Relay {
			return ctx.notify(reply{550, "5.1.1 No such user here."})
		}

		ctx.Message.AddOutbound(addr)
		ctx.Mode = stateDATA

		return ctx.notify(reply{251, "User not local; will forward to <forward-path>"})

	case mailbox.QuerySuccess:
		ctx.Message.AddInbound(addr)
		ctx.Mode = stateDATA

		return ctx.notify(reply{250, "2.1.5 Ok"})
	}

	return ctx.notify(replyErrorProcessing)
}

// RFC 2821 4.1.1.4 DATA (DATA)
func cmdDATA(ctx *transaction, param string) error {
	switch ctx.Mode {
	case stateHELO:
		return ctx.notify(reply{503, "5.5.1 Say HELO/EHLO first."})

	case stateMAIL:
		return ctx.notify(reply{503, "5.5.1 Bad sequence of commands."})

	case stateRCPT:
		return ctx.notify(reply{503, "5.5.1 RCPT first."})
	}

	if err := ctx.notify(reply{354, "End data with <CR><LF>.<CR><LF>"}); err != nil {
		return err
	}

	msg, err := ctx.text.ReadDotLines()
	if err != nil {
		return ctx.notify(replyErrorProcessing)
	}

	ctx.Message.Raw = strings.Join(msg, "\r\n")
	return ctx.host.digest(ctx)
}

// RFC 2821 4.1.1.5 RESET (RSET)
func cmdRSET(ctx *transaction, param string) error {
	ctx.reset()
	return ctx.notify(reply{250, "2.1.5 Flushed"})
}

// RFC 2821 4.1.1.9 NOOP (NOOP)
func cmdNOOP(ctx *transaction, param string) error {
	return ctx.notify(reply{250, "2.0.0 Ok"})
}

// RFC 2821 4.1.1.6 VERIFY (VRFY)
func cmdVRFY(ctx *transaction, param string) error {
	return ctx.notify(reply{252, "2.1.5 Send some mail, I'll try my best"})
}

// RFC 2821 4.1.1.10 QUIT (QUIT)
func cmdQUIT(ctx *transaction, param string) error {
	ctx.notify(reply{221, "2.0.0 Adeus"})
	return io.EOF
}
