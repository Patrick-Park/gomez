package pop3

import (
	"log"
	"net"
	"regexp"
)

type server struct {
	ln net.Listener
	cs commandSpec
}

type commandSpec map[string]func(*transaction, string) error

type transaction struct {
	conn net.Conn
}

var commandFormat = regexp.MustCompile("^([a-zA-Z]{4})(?:[ ](.*))?$")

func Start() error {
	var srv server
	ln, err := net.Listen("tcp", ":110")
	if err != nil {
		return err
	}
	srv.ln = ln
	srv.cs = commandSpec{
		"USER": cmdUSER,
		"PASS": cmdPASS,
		"STAT": cmdSTAT,
		"LIST": cmdLIST,
		"RETR": cmdRETR,
		"DELE": cmdDELE,
		"NOOP": cmdNOOP,
		"RSET": cmdRSET,
		"QUIT": cmdQUIT,
	}
	for {
		cn, err := ln.Accept()
		if err != nil {
			log.Printf("error accepting incoming connection: %s", err)
			continue
		}
		go srv.createTransaction(cn)
	}
	return nil
}

func (s server) createTransaction(conn net.Conn) {}

func (s server) run(ctx *transaction, msg string) error {
	return nil
}
