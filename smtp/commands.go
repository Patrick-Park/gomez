package smtp

import (
	"log"
	"strings"
)

func cmdHELO(c *Client, param string) {

}

func cmdEHLO(c *Client, param string) {

}

func cmdMAIL(c *Client, param string) {

}

func cmdRCPT(c *Client, param string) {

}

func cmdDATA(c *Client, param string) {
	msg, err := c.conn.ReadDotLines()
	if err != nil {
		log.Printf("Could not read dot lines: %s\n", err)
	}

	c.msg.SetBody(strings.Join(msg, ""))
	c.Host.Digest(c)
	c.Reset()
	c.Reply(Reply{250, "Message queued"})
}

func cmdRSET(c *Client, param string) {
	c.Reset()
}

func cmdNOOP(c *Client, param string) {
	return
}

func cmdVRFY(c *Client, param string) {

}

func cmdQUIT(c *Client, param string) {
	c.Mode = MODE_QUIT
}
