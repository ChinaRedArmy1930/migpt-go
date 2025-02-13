package status

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrKeyNotExist = errors.New("key not exists")
	ErrKeyExist    = errors.New("key exists")
)

type statusEntry struct {
	onExit func()
	timer  *time.Timer
	ok     atomic.Bool
}

type TimeoutStatus struct {
	mu      sync.Mutex
	timeout time.Duration
	status  map[any]*statusEntry
}

func NewTimeoutStatus(timeout time.Duration) *TimeoutStatus {
	return &TimeoutStatus{
		timeout: timeout,
		status:  make(map[any]*statusEntry),
	}
}

// 判断状态是否 ok
func (s *TimeoutStatus) Ok(key any) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.status[key]; !ok {
		return false, ErrKeyNotExist
	}

	return s.status[key].ok.Load(), nil
}

func (s *TimeoutStatus) Add(key any, onExist func()) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.status[key]; ok {
		return false, ErrKeyExist
	}

	s.status[key] = &statusEntry{
		onExit: onExist,
		timer:  time.NewTimer(s.timeout),
	}

	return true, nil
}

func (s *TimeoutStatus) Start(key any) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.status[key]; !ok {
		return false, ErrKeyNotExist
	}

	s.status[key].ok.Store(true)
	s.status[key].timer = time.AfterFunc(s.timeout, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.status[key].ok.Store(false)
		if s.status[key].onExit != nil {
			s.status[key].onExit()
		}
	})

	return true, nil
}

// 状态续期
func (s *TimeoutStatus) Renewal(key any) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.status[key]; !ok {
		return false, ErrKeyNotExist
	}

	s.status[key].ok.Store(true)
	if !s.status[key].timer.Stop() {
		select {
		case <-s.status[key].timer.C:
		default:
		}
	}

	s.status[key].timer.Reset(s.timeout)

	return true, nil
}

// 退出状态
func (s *TimeoutStatus) Exit(key any) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.status[key]; !ok {
		return false, ErrKeyNotExist
	}

	if !s.status[key].timer.Stop() {
		select {
		case <-s.status[key].timer.C:
		default:
		}
	}

	s.status[key].ok.Store(false)

	if s.status[key].onExit != nil {
		s.status[key].onExit()
	}
	return true, nil
}

// 删除当前key
func (s *TimeoutStatus) Over(key any) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.status[key]; !ok {
		return false, ErrKeyNotExist
	}

	if s.status[key].onExit != nil {
		s.status[key].onExit()
	}

	if !s.status[key].timer.Stop() {
		<-s.status[key].timer.C
	}

	delete(s.status, key)

	return true, nil
}

func (s *TimeoutStatus) Exist(key any) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.status[key]
	return ok
}
