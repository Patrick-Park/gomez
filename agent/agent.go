package agent

import (
	"errors"
	"fmt"
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
	config  jamon.Group
	lastRun time.Time
	group   sync.WaitGroup
	running bool
}

func Start(dq mailbox.Dequeuer, conf jamon.Group) error {
	cron := cronJob{config: conf}
	s, err := strconv.Atoi(cron.config.Get("pause"))
	if err != nil {
		return err
	}
	tick := time.Duration(s) * time.Second
	for {
		cron.lastRun = <-time.After(tick)
		jobs, err := dq.Dequeue()
		if err != nil {
			// log it / alert
			continue
		}
		for host, pkg := range jobs {
			cron.group.Add(1)
			go func() {
				defer cron.group.Done()
				err := cron.deliverTo(host, pkg)
				if err != nil {
					// Could not lookup DNS
					// Could not connect to host
				}
			}()
		}
		cron.group.Wait()
	}
}

func (cron *cronJob) deliverTo(host string, pkg mailbox.Package) error {
	MXs, err := lookupMX(host)
	if err != nil {
		return errFailedDNS
	}
	//TODO(gbbr): Use config retries
	var client *smtp.Client
	for retry := 0; retry < 2; retry++ {
		for _, mx := range MXs {
			//TODO(gbbr): Use config timeout, per MX entry and per host
			conn, err := net.DialTimeout("tcp", mx.Host+":25", 5*time.Second)
			if err != nil {
				continue
			}
			//TODO(gbbr): Use config host
			client, err = smtp.NewClient(conn, "mecca.local")
			if err != nil {
				continue
			}
			goto DELIVERY
		}
	}
	return errFailedHost

DELIVERY:
	defer func() {
		if err = client.Quit(); err != nil {
			// LOG disconnect error
		}
	}()
	for msg, rcptList := range pkg {
		err = cron.sendMessage(client, msg, rcptList)
		if err != nil {
			// Failed to send message, wrong status code
			// RECORD error
			continue
		}
		// ANALYZE report
	}
	return nil
}

// lookupMX returns a list of MX hosts ordered by preference.
// This function is declared inline so we can mock it.
var lookupMX = func(host string) ([]*net.MX, error) {
	return net.LookupMX(host)
}

var (
	errFailedHost = errors.New("failed connecting to MX hosts after all tries")
	errFailedDNS  = errors.New("failed to lookup DNS")
)

// sendMessage attempts to send the message and returns a detailed DeliveryReport
func (cron *cronJob) sendMessage(client *smtp.Client, msg *mailbox.Message, rcptList []*mail.Address) error {
	if err := client.Mail(msg.From().String()); err != nil {
		return err
	}
	for _, rcpt := range rcptList {
		if err := client.Rcpt(rcpt.String()); err != nil {
			// did not get 25 response
			// failed recipient
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(w, msg.Raw)
	if err != nil {
		return err
	}
	if err = w.Close(); err != nil {
		return err
	}
	return nil
}
