package circuitbreaker

import (
	"sync"
	"sync/atomic"
	"time"
)

const (
	nanosPerMillis  = int64(time.Millisecond / time.Nanosecond)
	nanosPerSeconds = int64(time.Second / time.Nanosecond)
)

var (
	currentClock *atomic.Value
	nowInMs      = int64(0)
)

type timer struct {
	C        <-chan time.Time
	c        chan time.Time
	duration time.Duration
}

type iClock interface {
	timer(d time.Duration) *timer
	currentTime() time.Time
	currentCachedTimeMillis() int64
	currentCachedTimeSeconds() int64
}

type clockWrapper struct {
	clock iClock
}

func setClock(c iClock) {
	currentClock.Store(&clockWrapper{c})
}

func loadClock() iClock {
	return currentClock.Load().(*clockWrapper).clock
}

func init() {
	c := newRealClock()
	currentClock = new(atomic.Value)
	setClock(c)

	initTimeCaching()
}

// initTimeCaching starts a background task that caches current timestamp per ms,
// which may provide better performance in high-concurrency scenarios.
func initTimeCaching() {
	atomic.StoreInt64(&nowInMs, time.Now().UnixNano()/nanosPerMillis)
	go func() {
		ticker := time.NewTicker(time.Millisecond)
		for range ticker.C {
			now := time.Now().UnixNano() / nanosPerMillis
			atomic.StoreInt64(&nowInMs, now)
		}
	}()
}

type realClock struct{}

func newRealClock() *realClock {
	return &realClock{}
}
func (r *realClock) timer(d time.Duration) *timer {
	t := time.NewTimer(d)
	return &timer{C: t.C}
}

func (r *realClock) currentTime() time.Time {
	return time.Now()
}

func (r *realClock) currentCachedTimeMillis() int64 {
	nowMs := atomic.LoadInt64(&nowInMs)
	if nowMs == 0 {
		nowMs = time.Now().UnixNano() / nanosPerMillis
	}
	return nowMs
}

func (r *realClock) currentCachedTimeSeconds() int64 {
	nowMs := atomic.LoadInt64(&nowInMs)
	if nowMs == 0 {
		return time.Now().UnixNano() / nanosPerSeconds
	}
	return nowMs * nanosPerMillis / nanosPerSeconds
}

// currentTime() returns the current time from the current clock.
// If useCachedTime is set to true, this returns the cached time which has ms precision.
func currentTime(useCachedTime bool) time.Time {
	if useCachedTime {
		return time.Unix(0, loadClock().currentCachedTimeMillis()*nanosPerMillis)
	}

	return loadClock().currentTime()
}

func currentTimeInSecond() int64 {
	return loadClock().currentCachedTimeSeconds()
}

// mockClock implements clock and is used to replace the globalClock for testing.
type mockClock struct {
	currentMockTime time.Time
	timers          []*timer
	updateLock      sync.RWMutex
}

func newMockClock() *mockClock {
	return &mockClock{
		currentMockTime: time.Time{},
		timers:          nil,
		updateLock:      sync.RWMutex{},
	}
}
func (m *mockClock) currentCachedTimeMillis() int64 {
	return m.currentMockTime.UnixNano() / nanosPerMillis
}
func (m *mockClock) currentCachedTimeSeconds() int64 {
	return m.currentMockTime.UnixNano() / nanosPerSeconds
}
func (m *mockClock) currentTime() time.Time {
	return m.currentMockTime
}
func (m *mockClock) timer(d time.Duration) *timer {
	m.updateLock.Lock()
	defer m.updateLock.Unlock()

	ch := make(chan time.Time, 1)
	t := &timer{
		C:        ch,
		c:        ch,
		duration: d,
	}
	m.timers = append(m.timers, t)
	return t
}

func (m *mockClock) advance(duration time.Duration) {
	m.updateLock.Lock()
	defer m.updateLock.Unlock()

	oldTime := m.currentMockTime
	m.currentMockTime = m.currentMockTime.Add(duration)

	for _, t := range m.timers {
		if m.currentMockTime.After(oldTime.Add(t.duration)) {
			t.c <- m.currentMockTime
		}
	}
}
