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

	"github.com/gbbr/gomez/mailbox"
	"github.com/gbbr/jamon"
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
				cron.deliverTo(host, pkg)
			}()
		}
		cron.group.Wait()
	}
}

func (cron *cronJob) deliverTo(host string, pkg mailbox.Package) {
	client, err := cron.getSMTPClient(host)
	if err != nil {
		// handle error
	}
	defer func() {
		if err = client.Quit(); err != nil {
			// log error
		}
	}()
	for msg, rcptList := range pkg {
		cron.sendMessage(client, msg, rcptList)
	}
}

// lookupMX returns a list of MX hosts ordered by preference.
// This function is declared inline so we can mock it.
var lookupMX = func(host string) ([]*net.MX, error) {
	return net.LookupMX(host)
}

// errFailedConnect is returned when connecting was not possible to any
// of the MX hosts.
var errFailedConnect = errors.New("failed connecting to MX hosts after all tries")

func (cron *cronJob) getSMTPClient(host string) (*smtp.Client, error) {
	MXs, err := lookupMX(host)
	if err != nil {
		return nil, err
	}
	//TODO(gbbr): Use config retries
	for retry := 0; retry < 2; retry++ {
		for _, mx := range MXs {
			//TODO(gbbr): Use config timeout
			conn, err := net.DialTimeout("tcp", mx.Host+":25", 5*time.Second)
			if err != nil {
				continue
			}
			//TODO(gbbr): Use config host
			c, err := smtp.NewClient(conn, "mecca.local")
			if err != nil {
				continue
			}
			return c, nil
		}
	}
	return nil, errFailedConnect
}

// sendMessage attempts to send the message and returns a detailed DeliveryReport
func (cron *cronJob) sendMessage(client *smtp.Client, msg *mailbox.Message, rcptList []*mail.Address) {
	if err := client.Mail(msg.From().String()); err != nil {
		// handle err; increase attempts, continue
	}
	for _, rcpt := range rcptList {
		if err := client.Rcpt(rcpt.String()); err != nil {
			// Failed to deliver to this rcpt
		}
	}
	w, err := client.Data()
	if err != nil {
		// Failed to deliver this msg. Retry?
	}
	_, err = fmt.Fprint(w, msg.Raw)
	if err != nil {
		// Failed to deliver this msg. Retry?
	}
	if err = w.Close(); err != nil {
		// handle err. Are we done?
	}
}
