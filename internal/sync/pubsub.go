package sync

import (
	"context"
	internal "migpt-go/internal/log"
	"sync"
)

type f func()

type Subscriber interface {
	OnMessage(msg any)
}

type PubSub struct {
	cs *CallbackSerializer
	mu *sync.Mutex
	//最近一条消息
	msg         any
	subscribers map[Subscriber]bool
}

func NewPubSub(ctx context.Context) *PubSub {
	return &PubSub{
		cs:          NewCallbackSerializer(ctx),
		subscribers: make(map[Subscriber]bool),
		mu:          &sync.Mutex{},
	}
}

func (ps *PubSub) Subseribe(sub Subscriber, consumeLast bool) (cancel f) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.subscribers[sub] = true

	if msg := ps.msg; consumeLast && msg != nil {
		ps.cs.TrySchedule(func(context.Context) {
			ps.mu.Lock()
			defer ps.mu.Unlock()

			if !ps.subscribers[sub] {
				return
			}

			sub.OnMessage(msg)
		})
	}

	return func() {
		ps.mu.Lock()
		defer ps.mu.Unlock()
		delete(ps.subscribers, sub)
	}
}

func (ps *PubSub) Publish(msg any) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.msg = msg

	for sub := range ps.subscribers {
		ps.cs.ScheduleOr(func(ctx context.Context) {
			ps.mu.Lock()
			defer ps.mu.Unlock()

			if !ps.subscribers[sub] {
				return
			}

			sub.OnMessage(msg)
		}, func() {
			internal.GetLogger().Warnf(ps.cs.ctx, "%#v schedule failed", sub)
		})
	}
}

func (ps *PubSub) Done() <-chan struct{} {
	return ps.cs.Done()
}
