package agent

import (
	"fmt"
	"log"
	"net"
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
			go cron.deliverTo(host, pkg)
		}
		cron.group.Wait()
		cron.lastRun = <-time.After(tick)
	}
}

type byPref []*net.MX

func (b byPref) Len() int           { return len(b) }
func (b byPref) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byPref) Less(i, j int) bool { return b[i].Pref < b[j].Pref }

// lookupMX returns a list of MX hosts ordered by preference.
// This function is declared inline so we can mock it.
var lookupMX = func(host string) []*net.MX {
	MXs, err := net.LookupMX(host)
	if err != nil {
		log.Println(err)
	}
	sort.Sort(byPref(MXs))
	return MXs
}

func (cron *cronJob) deliverTo(host string, pkg mailbox.Delivery) {
	cron.group.Add(1)
	defer cron.group.Done()
	for _, h := range lookupMX(host) {
		fmt.Printf("%d - %s\n", h.Pref, h.Host)
	}
}
