package smtp

import (
	"net/mail"
	"strings"

	"github.com/gbbr/gomez"
)

// RFC 2821 4.1.1.1  Extended HELLO (EHLO) or HELLO (HELO)
// This commands are used to identify the SMTP client to the SMTP
// server. The argument field contains the fully-qualified domain name
// of the SMTP client if one is available.
func cmdHELO(ctx *Client, param string) error {
	if len(param) == 0 {
		return ctx.Notify(Reply{501, "Syntax: HELO hostname"})
	}

	ctx.Id = param
	ctx.Mode = MODE_MAIL

	return ctx.Notify(Reply{250, "Gomez SMTPd"})
}

// RFC 2821 4.1.1.1  Extended HELLO (EHLO) or HELLO (HELO)
// This commands are used to identify the SMTP client to the SMTP
// server. The response of EHLO is normally a multi-line one reflecting
// the supported features of this server.
func cmdEHLO(ctx *Client, param string) error {
	if len(param) == 0 {
		return ctx.Notify(Reply{501, "Syntax: EHLO hostname"})
	}

	ctx.Id = param
	ctx.Mode = MODE_MAIL

	return ctx.Notify(Reply{250, "Gomez SMTPd\n8BITMIME\nVRFY"})
}

// RFC 2821 4.1.1.2 MAIL (MAIL)
// This command is used to initiate a mail transaction in which the
// mail data is delivered to an SMTP server
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
// This command is used to identify an individual recipient of the mail
// data; multiple recipients are specified by multiple use of this
// command.
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
			return ctx.Notify(Reply{550, "5.1.1 No such user here. Relaying is not supported."})
		}

		ctx.Message.AddRcpt(addr)
		ctx.Mode = MODE_DATA

		return ctx.Notify(Reply{251, "User not local; will forward to <forward-path>"})

	case gomez.QUERY_STATUS_SUCCESS:
		ctx.Message.AddRcpt(addr)
		ctx.Mode = MODE_DATA

		return ctx.Notify(Reply{250, "OK"})
	}

	return ctx.Notify(replyErrorProcessing)
}

// RFC 2821 4.1.1.4 DATA (DATA)
// This command causes the mail data to be appended to the mail data buffer.
// The mail data is terminated by a line containing only a period, that is,
// the character sequence "<CRLF>.<CRLF>"
func cmdDATA(ctx *Client, param string) error {
	switch ctx.Mode {
	case MODE_HELO:
		return ctx.Notify(Reply{503, "5.5.1 Say HELO/EHLO first."})

	case MODE_MAIL:
		return ctx.Notify(Reply{503, "5.5.1 Bad sequence of commands."})

	case MODE_RCPT:
		return ctx.Notify(Reply{503, "5.5.1 RCPT first."})
	}

	err := ctx.Notify(Reply{354, "End data with <CR><LF>.<CR><LF>"})
	if err != nil {
		return ctx.Notify(replyErrorProcessing)
	}

	msg, err := ctx.conn.ReadDotLines()
	if err != nil {
		return ctx.Notify(replyErrorProcessing)
	}

	err = ctx.Message.FromRaw(strings.Join(msg, "\r\n"))
	if err == gomez.ERR_MESSAGE_NOT_COMPLIANT {
		return ctx.Notify(Reply{550, "Message not RFC 2822 compliant."})
	} else if err != nil {
		return ctx.Notify(replyErrorProcessing)
	}

	err = ctx.host.Digest(ctx)
	if err != nil {
		return ctx.Notify(replyErrorProcessing)
	}

	ctx.Reset()

	return ctx.Notify(Reply{250, "Message queued"})
}

// RFC 2821 4.1.1.5 RESET (RSET)
// This command specifies that the current mail transaction will be aborted.
// Any stored sender, recipients, and mail data MUST be discarded, and all
// buffers and state tables cleared.
func cmdRSET(ctx *Client, param string) error {
	ctx.Reset()
	return ctx.Notify(Reply{250, "2.1.5 Flushed"})
}

// RFC 2821 4.1.1.9 NOOP (NOOP)
// This command does not affect any parameters or previously entered
// commands.
func cmdNOOP(ctx *Client, param string) error {
	return ctx.Notify(Reply{250, "2.0.0 Ok"})
}

// RFC 2821 4.1.1.6 VERIFY (VRFY)
// This command asks the receiver to confirm that the argument
// identifies a user or mailbox.
func cmdVRFY(ctx *Client, param string) error {
	return ctx.Notify(Reply{252, "2.1.5 Send some mail, I'll try my best"})
}

// RFC 2821 4.1.1.10 QUIT (QUIT)
// This command specifies that the receiver MUST send an OK reply, and
// then close the transmission channel.
func cmdQUIT(ctx *Client, param string) error {
	ctx.Mode = MODE_QUIT
	return ctx.Notify(Reply{221, "2.0.0 Adeus"})
}
