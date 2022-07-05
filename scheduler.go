package main

import (
	"time"
)

type Scheduler struct {
	triggerC   chan struct{}
	triggerTime time.Time
	lastChange time.Time
	ticker time.Ticker
	unusedChan bool
}

func NewScheduler() *Scheduler {
	s := &Scheduler {
		triggerC: make(chan struct{}, 1),
		lastChange: time.Now(),
		ticker: *time.NewTicker(time.Millisecond * 100),
	}

	s.triggerTime = s.lastChange

	go func() {
		for range s.ticker.C {
			if s.triggerTime == s.lastChange {
				continue
			}

			if s.triggerTime.Sub(s.lastChange) < (time.Second * 2) {
				time.Sleep(time.Duration(2 - s.triggerTime.Sub(s.lastChange).Seconds()))
			}

			if s.unusedChan {
				<-s.triggerC
			}

			s.triggerC <- struct{}{}
			s.unusedChan = true
		}
	}()

	return s
}

func (s *Scheduler) TriggerChange() {
	s.triggerTime = time.Now()
}

func (s *Scheduler) WaitForChange() {
	<-s.triggerC

	s.unusedChan = false
	s.lastChange = time.Now()
	s.triggerTime = s.lastChange
}