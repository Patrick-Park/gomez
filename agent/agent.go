package agent

import (
	"strconv"
	"time"

	"github.com/gbbr/gomez/mailbox"
	"github.com/gbbr/jamon"
)

type cronJob struct {
	lastRun time.Time
	running bool
}

func Run(dq mailbox.Dequeuer, conf jamon.Group) error {
	var cron cronJob
	s, err := strconv.Atoi(conf.Get("tick"))
	if err != nil {
		return err
	}
	tick := time.Duration(s) * time.Second
	for {
		cron.lastRun = <-time.After(tick)
		jobs, err := dq.Dequeue()
		if err != nil {
			return err
		}
		for host, pkg := range jobs {
			go deliverTo(host, pkg)
		}
	}
}

func deliverTo(host string, pkg mailbox.Delivery) {

}
