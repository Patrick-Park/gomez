// draft
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

	"github.com/gbbr/gomez/mailbox"
	"github.com/gbbr/jamon"
)

type cronJob struct {
	config jamon.Group
	dq     mailbox.Dequeuer
	done   chan report
	failed chan report
	retry  chan report
}

type report struct {
	msgID  uint64
	rcpt   []*mail.Address
	reason error
}

func Start(dq mailbox.Dequeuer, conf jamon.Group) error {
	pause, err := strconv.Atoi(conf.Get("pause"))
	if err != nil {
		log.Fatal("agent/pause configuration is not numeric")
	}
	cron := cronJob{
		dq:     dq,
		config: conf,
		failed: make(chan report),
		done:   make(chan report),
		retry:  make(chan report),
	}
	for {
		time.Sleep(time.Duration(pause) * time.Second)

		jobs, err := cron.dq.Dequeue()
		if err != nil {
			log.Printf("error dequeuing: %s", err)
			continue
		}

		go func() {
			for {
				select {
				case r, more := <-cron.done:
					if !more {
						return
					}
					dq.Delivered(r.msgID, r.rcpt)
				case r := <-cron.retry:
					dq.Retry(r.msgID, r.rcpt, r.reason)
				case r := <-cron.failed:
					dq.Failed(r.msgID, r.rcpt, r.reason)
				}
			}
		}()

		var wg sync.WaitGroup
		wg.Add(len(jobs))
		for host, pkg := range jobs {
			go func() {
				defer wg.Done()
				cron.deliverTo(host, pkg)
			}()
		}
		wg.Wait()
		close(cron.done)
		dq.Flush()
	}
	return nil
}

func (cron *cronJob) deliverTo(host string, pkg mailbox.Package) {
	client, err := cron.getSMTPClient(host)
	if err != nil {
		// cron.log <- err
		return
	}
	defer func() {
		if err := client.Quit(); err != nil {
			log.Printf("error quitting client: %s", err)
		}
	}()
	for msg, all := range pkg {
		if err := client.Mail(msg.From().String()); err != nil {
			cron.retry <- report{msgID: msg.ID, rcpt: all, reason: err}
			continue
		}
		ok := report{msgID: msg.ID, rcpt: make([]*mail.Address, 0, len(all))}
		for _, rcpt := range all {
			if err := client.Rcpt(rcpt.String()); err != nil {
				cron.failed <- report{msg.ID, []*mail.Address{rcpt}, err}
				continue
			}
			ok.rcpt = append(ok.rcpt, rcpt)
		}
		w, err := client.Data()
		if err != nil {
			ok.reason = err
			cron.retry <- ok
			continue
		}
		_, err = fmt.Fprint(w, msg.Raw)
		if err != nil {
			ok.reason = err
			cron.retry <- ok
			continue
		}
		if err = w.Close(); err != nil {
			ok.reason = err
			cron.retry <- ok
			continue
		}
		cron.done <- ok
	}
}

var errFailedHost = errors.New("failed connecting to MX hosts after all tries")

// We declare inline so we can mock to local in tests.
var lookupMX = func(host string) ([]*net.MX, error) {
	return net.LookupMX(host)
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
