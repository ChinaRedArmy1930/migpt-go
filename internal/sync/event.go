package sync

import (
	"sync"
	"sync/atomic"
)

type emtpych chan struct{}

/*
确保某件事情只做一次
*/
type Event struct {
	done atomic.Bool
	ch   emtpych
	one  sync.Once
}

func (e *Event) Do() bool {
	ret := false

	e.one.Do(func() {
		e.done.Store(true)
		close(e.ch)
		ret = true
	})

	return ret
}

func (e *Event) Done() <-chan struct{} {
	return e.ch
}

func (e *Event) HasDone() bool {
	return e.done.Load()
}

func NewEvent() *Event {
	return &Event{ch: make(emtpych)}
}
