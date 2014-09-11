package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/textproto"
	"strconv"
	"strings"

	_ "github.com/gbbr/gomez/smtp"
)

var port = flag.Int("port", 25, "specifies the port that the SMTP server starts on")

func handshake(conn net.Conn) {
	fmt.Printf("[SMTP] Connected: %s\n", conn.RemoteAddr())

	text := textproto.NewConn(conn)
	defer text.Close()

	for {
		cmd, err := text.ReadLine()
		if err != nil {
			log.Printf("Error: %s", err)
			break
		}

		if strings.ToUpper(cmd) == "BYE" {
			fmt.Printf("[SMTP] Disconnected %s\n", conn.RemoteAddr())
			break
		}

		fmt.Println("Received: " + cmd)
	}
}

func main() {
	flag.Parse()

	ln, err := net.Listen("tcp", ":"+strconv.Itoa(*port))
	if err != nil {
		log.Printf("Error: %s", err)
		return
	}

	fmt.Printf("[SMTP] Listening on port :%d\n", *port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Error: %s", err)
		}

		go handshake(conn)
	}
}
