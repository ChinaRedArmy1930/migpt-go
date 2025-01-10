package buffer

import (
	"errors"
	"sync"
)

/*
通过 backlog 这个数组, 可以实现无限数量(理论)的缓冲区
*/
type Unbounded struct {
	c       chan any
	closed  bool
	closing bool
	mu      sync.Mutex
	backlog []any
}

var errBufferClosed = errors.New("error:buffer closed")

func NewUnbounded() *Unbounded {
	return &Unbounded{c: make(chan any, 1)}
}

func (b *Unbounded) Put(t any) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closing {
		return errBufferClosed
	}

	if len(b.backlog) == 0 {
		select {
		case b.c <- t:
			return nil
		default:
		}
	}

	b.backlog = append(b.backlog, t)
	return nil
}

func (b *Unbounded) Load() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.backlog) > 0 {
		select {
		case b.c <- b.backlog[0]:
			b.backlog[0] = nil
			b.backlog = b.backlog[1:]
		default:
		}
	} else if b.closing && !b.closed {
		close(b.c)
	}
}

func (b *Unbounded) Get() <-chan any {
	return b.c
}

func (b *Unbounded) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closing {
		return
	}

	b.closing = true

	if len(b.backlog) == 0 {
		b.closed = true
		close(b.c)
	}
}
