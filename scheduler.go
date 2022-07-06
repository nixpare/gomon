package main

import (
	"fmt"
	"sync"
	"time"
)

type Scheduler struct {
	triggerChange bool
	waiting bool
	triggerTime time.Time
	lastChangeTime time.Time
	RoutineWG *sync.WaitGroup
	waitChangesForRecompile bool
}

func NewScheduler() *Scheduler {
	s := new(Scheduler)
	s.RoutineWG = new(sync.WaitGroup)

	ticker := time.NewTicker(time.Millisecond * 100)

	go func() {
		for range ticker.C {
			if s.triggerTime.Before(s.lastChangeTime) || s.triggerTime.Equal(s.lastChangeTime) {
				continue
			}

			s.triggerChange = true
		}
	}()

	return s
}

func (s *Scheduler) TriggerChange() {
	s.triggerTime = time.Now()
}

func (s *Scheduler) WaitForChange(condition *bool) {
	if s.waiting {
		panic(fmt.Errorf("already waiting"))
	}
	s.waiting = true
	defer func() {
		s.waiting = false
	}()

	if condition != nil {
		for !s.triggerChange && !*condition {
			time.Sleep(time.Millisecond * 10)
		}

		if *condition {
			return
		}
	} else {
		for !s.triggerChange {
			time.Sleep(time.Millisecond * 10)
		}
	}

	s.lastChangeTime = time.Now()
	s.triggerChange = false
}