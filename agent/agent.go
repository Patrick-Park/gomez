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

	report chan report
	flush  chan flushRequest
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
		report: make(chan report),
		flush:  make(chan flushRequest),
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
				case r := <-cron.report:
					_ = r
					// dq.Report(r.msgID, r.success, r.failed)
				case req := <-cron.flush:
					// dq.Flush(req.count) // compare and finish, find potential misses
					req.done <- nil
					break
				}
			}
		}()

		counter := make(chan int)
		var wg sync.WaitGroup
		for host, pkg := range jobs {
			wg.Add(1)
			go func() {
				defer wg.Done()
				client, err := cron.getSMTPClient(host)
				if err != nil {
					// cron.dq.FlagHost(host)
				}
				counter <- cron.deliver(client, pkg)
			}()
		}

		go func() {
			wg.Wait()
			close(counter)
		}()

		var count int
		for k := range counter {
			count += k
		}
		ok := make(chan error)
		cron.flush <- flushRequest{ok, count}
		<-ok
	}
	return nil
}

func (cron *cronJob) deliver(client *smtp.Client, pkg mailbox.Package) int {
	defer func() {
		if err := client.Quit(); err != nil {
			log.Printf("error quitting client: %s", err)
		}
	}()
	var rcpts int
	for msg, rcptList := range pkg {
		rcpts += len(rcptList)
		succes, fail := cron.sendMessage(client, msg, rcptList)
		cron.report <- report{msg.ID, succes, fail}
	}
	return rcpts
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
	rcptList []*mail.Address) ([]*mail.Address, []*mail.Address) {
	if err := client.Mail(msg.From().String()); err != nil {
		return nil, rcptList
	}
	var failed, succes []*mail.Address
	for _, rcpt := range rcptList {
		if err := client.Rcpt(rcpt.String()); err != nil {
			failed = append(failed, rcpt)
			continue
		}
		succes = append(succes, rcpt)
	}
	w, err := client.Data()
	if err != nil {
		return nil, rcptList
	}
	_, err = fmt.Fprint(w, msg.Raw)
	if err != nil {
		return nil, rcptList
	}
	if err = w.Close(); err != nil {
		return nil, rcptList
	}
	return succes, failed
}
