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
}

type flushRequest struct {
	done  chan error
	count int
}

type report struct {
	msgID   uint64
	success []*mail.Address
	fail    []*mail.Address
}

func Start(dq mailbox.Dequeuer, conf jamon.Group) error {
	pause, err := strconv.Atoi(conf.Get("pause"))
	if err != nil {
		log.Fatal("agent/pause configuration is not numeric")
	}
	cron := cronJob{
		dq:     dq,
		config: conf,
	}
	for {
		time.Sleep(time.Duration(pause) * time.Second)

		jobs, err := cron.dq.Dequeue()
		if err != nil {
			log.Printf("error dequeuing: %s", err)
			continue
		}

		retry := make(chan []*mail.Address)
		done := make(chan []*mail.Address)
		failed := make(chan *mail.Address)
		go func() {
			for {
				select {
				case list, more := <-done:
					if !more {
						return
					}
					_ = list
					// dq.Done(list)
				case list := <-retry:
					_ = list
					// dq.Retry(list)
				case addr := <-failed:
					_ = addr
					// dq.Flush(addr)
				}
			}
		}()

		var wg sync.WaitGroup
		wg.Add(len(jobs))
		for host, pkg := range jobs {
			go func() {
				defer wg.Done()
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
				for msg, rcpt := range pkg {
					if err := client.Mail(msg.From().String()); err != nil {
						retry <- rcpt
						continue
					}
					var success []*mail.Address
					for _, addr := range rcpt {
						if err := client.Rcpt(addr.String()); err != nil {
							failed <- addr
							continue
						}
						success = append(success, addr)
					}
					w, err := client.Data()
					if err != nil {
						retry <- rcpt
						continue
					}
					_, err = fmt.Fprint(w, msg.Raw)
					if err != nil {
						retry <- rcpt
						continue
					}
					if err = w.Close(); err != nil {
						retry <- rcpt
						continue
					}
					done <- success
				}
			}()
		}

		wg.Wait()
		close(done)
		// dq.Flush()
	}
	return nil
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
