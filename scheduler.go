package main

import "time"

type Scheduler struct {
	triggerC   chan struct{}
	triggerTime time.Time
	lastChange time.Time
	ticker time.Ticker
}

func NewScheduler() *Scheduler {
	s := &Scheduler {
		triggerC: make(chan struct{}, 1),
		lastChange: time.Now(),
		ticker: *time.NewTicker(time.Millisecond * 500),
	}

	s.triggerTime = s.lastChange

	go func() {
		for range s.ticker.C {
			if s.triggerTime == s.lastChange {
				continue
			}

			if time.Since(s.triggerTime) < (time.Millisecond * 1500) {
				continue
			}

			s.triggerC <- struct{}{}
		}
	}()

	return s
}

func (s *Scheduler) TriggerChange() {
	s.triggerTime = time.Now()
}

func (s *Scheduler) WaitForChange() {
	<-s.triggerC
	s.lastChange = time.Now()
	s.triggerTime = s.lastChange
}