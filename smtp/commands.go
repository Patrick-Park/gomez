package smtp

import (
	"log"
	"strings"
)

// SMTP HELO command: initiates handshake by
// identifying foreign host
func cmdHELO(ctx *Client, param string) {
	if len(param) == 0 {
		ctx.Notify(Reply{501, "Syntax: HELO hostname"})
		return
	}

	ctx.Notify(Reply{250, "Gomez SMTPd"})
	ctx.Mode = MODE_MAIL
}

// SMTP EHLO command: same as HELO but provides information
// about server capabilities and support
func cmdEHLO(ctx *Client, param string) {
	if len(param) == 0 {
		ctx.Notify(Reply{501, "Syntax: EHLO hostname"})
		return
	}

	ctx.Notify(Reply{250, "Gomez SMTPd"})
	ctx.Notify(Reply{250, "VRFY"})
	ctx.Mode = MODE_MAIL
}

// SMTP MAIL command: sets sender address.
// Format is: MAIL FROM:<address>
func cmdMAIL(ctx *Client, param string) {

}

// SMTP RCPT command: set receipient. Can be executed
// multiple times for multiple receipients.
// Format: RCPT TO:<address>
func cmdRCPT(ctx *Client, param string) {

}

// SMTP DATA command: sets the message body. Can
// also include headers as per RFC 821 / RFC 2821
// Ends with <CR>.<CR>
func cmdDATA(ctx *Client, param string) {
	msg, err := ctx.conn.ReadDotLines()
	if err != nil {
		log.Printf("Could not read dot lines: %s\n", err)
	}

	ctx.msg.SetBody(strings.Join(msg, ""))
	ctx.Host.Digest(ctx)
	ctx.Reset()
	ctx.Notify(Reply{250, "Message queued"})
}

// SMTP RSET command: reset client state
// and message
func cmdRSET(ctx *Client, param string) {
	ctx.Reset()
	ctx.Notify(Reply{250, "2.0.0 Ok"})
}

// SMTP NOOP command: no operation
func cmdNOOP(ctx *Client, param string) {
	ctx.Notify(Reply{250, "2.0.0 Ok"})
	return
}

// SMTP VRFY command: queries the mailbox
// for a user lookup. If more than one entry
// is found, ambigous error is returned
func cmdVRFY(ctx *Client, param string) {

}

func cmdQUIT(ctx *Client, param string) {
	ctx.Mode = MODE_QUIT
	ctx.Notify(Reply{221, "2.0.0 Bye"})
}
