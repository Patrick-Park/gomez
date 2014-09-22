package smtp

import (
	"log"
	"strings"

	"github.com/gbbr/gomez"
)

func cmdHELO(ctx *Client, param string) error {
	if len(param) == 0 {
		return ctx.Notify(Reply{501, "Syntax: HELO hostname"})
	}

	ctx.Id = param
	ctx.Mode = MODE_MAIL

	return ctx.Notify(Reply{250, "Gomez SMTPd"})
}

func cmdEHLO(ctx *Client, param string) error {
	if len(param) == 0 {
		return ctx.Notify(Reply{501, "Syntax: EHLO hostname"})
	}

	ctx.Id = param
	ctx.Mode = MODE_MAIL

	return ctx.Notify(Reply{250, "Gomez SMTPd\nVRFY"})
}

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

func cmdRCPT(ctx *Client, param string) error {
	switch {
	case ctx.Mode == MODE_HELO:
		return ctx.Notify(Reply{503, "5.5.1 Say HELO/EHLO first."})

	case ctx.Mode == MODE_MAIL:
		return ctx.Notify(Reply{503, "5.5.1 Error: need MAIL command"})

	case !strings.HasPrefix(strings.ToUpper(param), "TO:"):
		return ctx.Notify(Reply{501, "5.5.4 Syntax: RCPT TO:<address>"})
	}

	addr, err := gomez.NewAddress(param[len("TO:"):])
	if err != nil {
		return ctx.Notify(Reply{501, "5.1.7 Bad recipient address syntax"})
	}

	switch ctx.Host.Query(addr) {
	case gomez.QUERY_STATUS_NOT_FOUND:
		return ctx.Notify(Reply{550, "No such user here."})

	case gomez.QUERY_STATUS_NOT_LOCAL:
		if flags := ctx.Host.Settings(); !flags.Relay {
			return ctx.Notify(Reply{550, "No such user here. Relaying is not supported."})
		}

		ctx.msg.AddRcpt(addr)
		ctx.Mode = MODE_DATA

		return ctx.Notify(Reply{251, "User not local; will forward to <forward-path>"})

	case gomez.QUERY_STATUS_SUCCESS:
		ctx.msg.AddRcpt(addr)
		ctx.Mode = MODE_DATA

		return ctx.Notify(Reply{250, "OK"})
	}

	return ctx.Notify(Reply{451, "Requested action aborted: error in processing"})
}

func cmdDATA(ctx *Client, param string) error {
	switch ctx.Mode {
	case MODE_HELO:
		return ctx.Notify(Reply{503, "5.5.1 Say HELO/EHLO first."})

	case MODE_MAIL:
		return ctx.Notify(Reply{503, "5.5.1 Bad sequence of commands."})

	case MODE_RCPT:
		return ctx.Notify(Reply{503, "5.5.1 RCPT first."})
	}

	msg, err := ctx.conn.ReadDotLines()
	if err != nil {
		log.Printf("Could not read dot lines: %s\n", err)
		return ctx.Notify(Reply{451, "Requested action aborted: error in processing"})
	}

	err = ctx.msg.FromRaw(strings.Join(msg, "\r\n"))
	if err != nil {
		return ctx.Notify(Reply{550, "Message not RFC 2822 compliant."})
	}

	err = ctx.Host.Digest(ctx)
	if err != nil {
		log.Printf("Error digesting message: %s\n", err)
		return ctx.Notify(Reply{451, "Requested action aborted: error in processing"})
	}

	ctx.Reset()

	return ctx.Notify(Reply{250, "Message queued"})
}

func cmdRSET(ctx *Client, param string) error {
	ctx.Reset()
	return ctx.Notify(Reply{250, "2.0.0 Ok"})
}

func cmdNOOP(ctx *Client, param string) error {
	return ctx.Notify(Reply{250, "2.0.0 Ok"})
}

func cmdVRFY(ctx *Client, param string) error {
	return ctx.Notify(Reply{252, "2.1.5 Send some mail, I'll try my best"})
}

func cmdQUIT(ctx *Client, param string) error {
	ctx.Mode = MODE_QUIT
	return ctx.Notify(Reply{221, "2.0.0 Adeus"})
}
