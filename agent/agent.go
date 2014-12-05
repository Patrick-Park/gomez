package agent

import (
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"sort"
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

func Run(dq mailbox.Dequeuer, conf jamon.Group) error {
	cron := cronJob{config: conf}
	s, err := strconv.Atoi(cron.config.Get("pause"))
	if err != nil {
		return err
	}
	tick := time.Duration(s) * time.Second
	for {
		jobs, err := dq.Dequeue()
		if err != nil {
			return err
		}
		for host, pkg := range jobs {
			cron.group.Add(1)
			go func() {
				defer cron.group.Done()
				cron.deliverTo(host, pkg)
			}()
		}
		cron.group.Wait()
		cron.lastRun = <-time.After(tick)
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

	for msg, rcpt := range pkg {
		if err := client.Mail(msg.From().String()); err != nil {
			// handle err; increase attempts, continue
		}
		for _, addr := range rcpt {
			if err := client.Rcpt(addr.String()); err != nil {
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
}

type byPref []*net.MX

func (b byPref) Len() int           { return len(b) }
func (b byPref) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byPref) Less(i, j int) bool { return b[i].Pref < b[j].Pref }

// lookupMX returns a list of MX hosts ordered by preference.
// This function is declared inline so we can mock it.
var lookupMX = func(host string) ([]*net.MX, error) {
	MXs, err := net.LookupMX(host)
	if err != nil {
		return nil, err
	}
	sort.Sort(byPref(MXs))
	return MXs, err
}

var (
	ErrFailedConnection = errors.New("failed connecting to MX hosts after all tries")
	ErrFailedLookup     = errors.New("failed to lookup MX hosts")
)

func (cron *cronJob) getSMTPClient(host string) (*smtp.Client, error) {
	MXs, err := lookupMX(host)
	if err != nil {
		return nil, ErrFailedLookup
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
	return nil, ErrFailedConnection
}
