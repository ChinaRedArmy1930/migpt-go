package cache

import (
	"sync"
	"sync/atomic"
	"time"
)

type cacheEntry struct {
	item    any
	cb      func()
	timer   *time.Timer
	deleted atomic.Bool
}

type TimeoutCache struct {
	mu      sync.Mutex
	timeout time.Duration
	cache   map[any]*cacheEntry
}

func NewTimeoutCache(timeout time.Duration) *TimeoutCache {
	return &TimeoutCache{
		timeout: timeout,
		cache:   make(map[any]*cacheEntry),
	}
}

func (c *TimeoutCache) Add(key, item any, cb func()) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if e, ok := c.cache[key]; ok {
		return e, false
	}

	entry := &cacheEntry{
		item: item,
		cb:   cb,
	}

	entry.timer = time.AfterFunc(c.timeout, func() {
		c.mu.Lock()

		if entry.deleted.Load() {
			c.mu.Unlock()
			return
		}

		delete(c.cache, key)
		c.mu.Unlock()

		entry.cb()
	})

	c.cache[key] = entry
	return item, true
}

func (c *TimeoutCache) remove(key any) (*cacheEntry, bool) {
	e, ok := c.cache[key]
	if !ok {
		return nil, false
	}

	delete(c.cache, key)

	if !e.timer.Stop() {
		e.deleted.Store(true)
	}

	return e, true
}

func (c *TimeoutCache) Remove(key any) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.remove(key)
	if !ok {
		return nil, false
	}

	return e.item, true
}

func (c *TimeoutCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for k := range c.cache {
		c.remove(k)
	}
}

func (c *TimeoutCache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return len(c.cache)
}
