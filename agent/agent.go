package agent

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"sync"
	"time"

	"github.com/gbbr/gomez/internal/jamon"
	"github.com/gbbr/gomez/mailbox"
)

type cronJob struct {
	config jamon.Group
	dq     mailbox.Dequeuer
	failed chan report
}

type report struct {
	id  uint64
	msg string
}

func Start(dq mailbox.Dequeuer, conf jamon.Group) error {
	cron := cronJob{dq: dq, config: conf}
	pause, err := strconv.Atoi(cron.config.Get("pause"))
	if err != nil {
		log.Fatal("agent/pause configuration is not numeric")
	}
	cron.failed = make(chan report)
	go func() {
		for {
			select {
			case rep := <-cron.failed:
				_ = rep
				// dq.Flag(rep.id, rep.msg)
			}
		}
	}()
	for {
		time.Sleep(time.Duration(pause) * time.Second)
		jobs, err := cron.dq.Dequeue()
		if err != nil {
			log.Printf("error dequeuing: %s", err)
			continue
		}
		var wg sync.WaitGroup
		for host, pkg := range jobs {
			wg.Add(1)
			go func() {
				defer wg.Done()
				cron.deliverTo(host, pkg)
			}()
		}
		wg.Wait()
	}
	return nil
}

func (cron *cronJob) deliverTo(host string, pkg mailbox.Package) {
	client, err := cron.getSMTPClient(host)
	if err != nil {
		// cron.dq.FlagHost(host)
		return
	}
	defer func() {
		if err = client.Quit(); err != nil {
			log.Printf("error quitting client: %s", err)
		}
	}()
	for msg, rcptList := range pkg {
		failed := cron.sendMessage(client, msg, rcptList)
		go func(id uint64, list []*mail.Address) {
			for _, addr := range list {
				cron.failed <- report{id, addr.String()}
			}
		}(msg.ID, failed)
	}
}

func (cron *cronJob) getSMTPClient(host string) (*smtp.Client, error) {
	MXs, err := lookupMX(host)
	if err != nil {
		return nil, err
	}
	for retry := 0; retry < 2; retry++ {
		for _, mx := range MXs {
			conn, err := net.DialTimeout("tcp", mx.Host+":25", 5*time.Second)
			if err != nil {
				continue
			}
			client, err := smtp.NewClient(conn, "mecca.local")
			if err != nil {
				continue
			}
			return client, nil
		}
	}
	return nil, errFailedHost
}

// We declare inline so we can mock to local in tests.
var lookupMX = func(host string) ([]*net.MX, error) {
	return net.LookupMX(host)
}

var errFailedHost = errors.New("failed connecting to MX hosts after all tries")

// sendMessage attempts to send a message to an SMTP client and returns
// a list of addresses which have failed delivery on success.
func (cron *cronJob) sendMessage(client *smtp.Client, msg *mailbox.Message,
	rcptList []*mail.Address) []*mail.Address {
	if err := client.Mail(msg.From().String()); err != nil {
		return rcptList
	}
	failed := make([]*mail.Address, 0)
	for _, rcpt := range rcptList {
		if err := client.Rcpt(rcpt.String()); err != nil {
			failed = append(failed, rcpt)
		}
	}
	w, err := client.Data()
	if err != nil {
		return rcptList
	}
	_, err = fmt.Fprint(w, msg.Raw)
	if err != nil {
		return rcptList
	}
	if err = w.Close(); err != nil {
		return rcptList
	}
	return failed
}
