package smtp

import (
	"io"
	"net/mail"
	"strings"

	"github.com/gbbr/gomez"
)

// RFC 2821 4.1.1.1  Extended HELLO (EHLO) or HELLO (HELO)
func cmdHELO(ctx *Client, param string) error {
	if len(param) == 0 {
		return ctx.Notify(Reply{501, "Syntax: HELO hostname"})
	}

	ctx.ID = param
	ctx.Mode = MODE_MAIL

	return ctx.Notify(Reply{250, "Gomez SMTPd"})
}

// RFC 2821 4.1.1.1  Extended HELLO (EHLO) or HELLO (HELO)
func cmdEHLO(ctx *Client, param string) error {
	if len(param) == 0 {
		return ctx.Notify(Reply{501, "Syntax: EHLO hostname"})
	}

	ctx.ID = param
	ctx.Mode = MODE_MAIL

	return ctx.Notify(Reply{250, "Gomez SMTPd\nSMTPUTF8\n8BITMIME\nENHANCEDSTATUSCODES\nVRFY"})
}

// RFC 2821 4.1.1.2 MAIL (MAIL)
func cmdMAIL(ctx *Client, param string) error {
	switch {
	case ctx.Mode == MODE_HELO:
		return ctx.Notify(Reply{503, "5.5.1 Say HELO/EHLO first."})

	case ctx.Mode > MODE_MAIL:
		return ctx.Notify(Reply{503, "5.5.1 Error: nested MAIL command"})

	case !strings.HasPrefix(strings.ToUpper(param), "FROM:"):
		return ctx.Notify(Reply{501, "5.5.4 Syntax: MAIL FROM:<address>"})
	}

	addr, err := mail.ParseAddress(param[len("FROM:"):])
	if err != nil {
		return ctx.Notify(Reply{501, "5.1.7 Bad sender address syntax"})
	}

	ctx.Message.SetFrom(addr)
	ctx.Mode = MODE_RCPT

	return ctx.Notify(Reply{250, "2.1.0 Ok"})
}

// RFC 2821 4.1.1.3 RECIPIENT (RCPT)
func cmdRCPT(ctx *Client, param string) error {
	switch {
	case ctx.Mode == MODE_HELO:
		return ctx.Notify(Reply{503, "5.5.1 Say HELO/EHLO first."})

	case ctx.Mode == MODE_MAIL:
		return ctx.Notify(Reply{503, "5.5.1 Error: need MAIL command"})

	case !strings.HasPrefix(strings.ToUpper(param), "TO:"):
		return ctx.Notify(Reply{501, "5.5.4 Syntax: RCPT TO:<address>"})
	}

	addr, err := mail.ParseAddress(param[len("TO:"):])
	if err != nil {
		return ctx.Notify(Reply{501, "5.1.7 Bad recipient address syntax"})
	}

	switch ctx.host.Query(addr) {
	case gomez.QUERY_STATUS_NOT_FOUND:
		return ctx.Notify(Reply{550, "5.1.1 No such user here."})

	case gomez.QUERY_STATUS_NOT_LOCAL:
		if flags := ctx.host.Settings(); !flags.Relay {
			return ctx.Notify(Reply{550, "5.1.1 No such user here."})
		}

		ctx.Message.AddRcpt(addr)
		ctx.Mode = MODE_DATA

		return ctx.Notify(Reply{251, "User not local; will forward to <forward-path>"})

	case gomez.QUERY_STATUS_SUCCESS:
		ctx.Message.AddRcpt(addr)
		ctx.Mode = MODE_DATA

		return ctx.Notify(Reply{250, "2.1.5 Ok"})
	}

	return ctx.Notify(replyErrorProcessing)
}

// RFC 2821 4.1.1.4 DATA (DATA)
func cmdDATA(ctx *Client, param string) error {
	switch ctx.Mode {
	case MODE_HELO:
		return ctx.Notify(Reply{503, "5.5.1 Say HELO/EHLO first."})

	case MODE_MAIL:
		return ctx.Notify(Reply{503, "5.5.1 Bad sequence of commands."})

	case MODE_RCPT:
		return ctx.Notify(Reply{503, "5.5.1 RCPT first."})
	}

	if err := ctx.Notify(Reply{354, "End data with <CR><LF>.<CR><LF>"}); err != nil {
		return err
	}

	msg, err := ctx.text.ReadDotLines()
	if err != nil {
		return ctx.Notify(replyErrorProcessing)
	}

	ctx.Message.Raw = strings.Join(msg, "\r\n")
	return ctx.host.Digest(ctx)
}

// RFC 2821 4.1.1.5 RESET (RSET)
func cmdRSET(ctx *Client, param string) error {
	ctx.Reset()
	return ctx.Notify(Reply{250, "2.1.5 Flushed"})
}

// RFC 2821 4.1.1.9 NOOP (NOOP)
func cmdNOOP(ctx *Client, param string) error {
	return ctx.Notify(Reply{250, "2.0.0 Ok"})
}

// RFC 2821 4.1.1.6 VERIFY (VRFY)
func cmdVRFY(ctx *Client, param string) error {
	return ctx.Notify(Reply{252, "2.1.5 Send some mail, I'll try my best"})
}

// RFC 2821 4.1.1.10 QUIT (QUIT)
func cmdQUIT(ctx *Client, param string) error {
	ctx.Notify(Reply{221, "2.0.0 Adeus"})
	return io.EOF
}
