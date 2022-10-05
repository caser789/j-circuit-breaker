package circuitbreaker

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
)

type windowOutcome uint32

const (
	successOutcome windowOutcome = iota
	errorOutcome
	timeoutSuccessOutcome
	timeoutErrorOutcome
)

type counter struct {
	TotalCount   int64 `json:"total_count"`
	ErrorCount   int64 `json:"error_count"`
	TimeoutCount int64 `json:"timeout_count"`
}

func (c *counter) add(outcome windowOutcome) {
	c.TotalCount++
	switch outcome {
	case errorOutcome:
		c.ErrorCount++
	case timeoutErrorOutcome:
		c.ErrorCount++
		c.TimeoutCount++
	case timeoutSuccessOutcome:
		c.TimeoutCount++
	}
}

func (c *counter) subtract(a *counter) {
	c.ErrorCount -= a.ErrorCount
	c.TimeoutCount -= a.TimeoutCount
	c.TotalCount -= a.TotalCount
}

func (c *counter) reset() {
	c.TotalCount = 0
	c.ErrorCount = 0
	c.TimeoutCount = 0
}

// WindowSnapshot models a snapshot of a sliding window
type WindowSnapshot struct {
	TotalCount   int64
	SuccessCount int64

	ErrorCount     int64
	ErrorPercent   int
	TimeoutCount   int64
	TimeoutPercent int
}

func (w WindowSnapshot) GetTimeoutPercent() int {
	return w.TimeoutPercent
}

func (w WindowSnapshot) GetErrorPercent() int {
	return w.ErrorPercent
}

func (w WindowSnapshot) GetTotalCount() int64 {
	return w.TotalCount
}

func newSnapshot(c *counter) WindowSnapshot {
	snapshot := WindowSnapshot{
		TotalCount:   c.TotalCount,
		SuccessCount: c.TotalCount - c.ErrorCount,
		ErrorCount:   c.ErrorCount,
		TimeoutCount: c.TimeoutCount,
	}

	if snapshot.TotalCount > 0 {
		snapshot.ErrorPercent = int(float64(c.ErrorCount) / float64(c.TotalCount) * 100)
		snapshot.TimeoutPercent = int(float64(c.TimeoutCount) / float64(c.TotalCount) * 100)
	}
	return snapshot
}

type countWindowMetric struct {
	capacity uint32
	endIndex int

	array []*counter
	total *counter

	updateLock sync.Mutex
}

func (c *countWindowMetric) moveForwardOneStep() *counter {
	c.endIndex = (c.endIndex + 1) % int(c.capacity)
	latest := c.array[c.endIndex]
	c.total.subtract(latest)
	latest.reset()
	return latest
}

func (c *countWindowMetric) update(outcome windowOutcome) WindowSnapshot {
	c.updateLock.Lock()
	defer c.updateLock.Unlock()

	c.moveForwardOneStep().add(outcome)
	c.total.add(outcome)
	return newSnapshot(c.total)
}

func (c *countWindowMetric) getSnapshot() WindowSnapshot {
	c.updateLock.Lock()
	defer c.updateLock.Unlock()
	return newSnapshot(c.total)
}

func newCountWindowMetric(windowSize uint32) *countWindowMetric {
	ca := &countWindowMetric{
		capacity: windowSize,
		array:    make([]*counter, windowSize),
		total:    &counter{},
	}
	for i := 0; i < int(windowSize); i++ {
		ca.array[i] = &counter{}
	}
	ca.endIndex = 0
	return ca
}

type bucket struct {
	counter
	startTime int64
}

func (b *bucket) reset(currEpochSecond int64) {
	b.startTime = currEpochSecond
	b.TotalCount = 0
	b.ErrorCount = 0
	b.TimeoutCount = 0
}

type buckets []*bucket

func (bs buckets) String() string {
	s := make([]string, 0, len(bs))
	for _, b := range bs {
		s = append(s, fmt.Sprintf("%v", b))
	}
	return strings.Join(s, ",")
}

type timeWindowMetric struct {
	capacity uint32
	endIndex int

	buckets buckets
	total   *counter

	updateLock sync.Mutex
}

func newTimeWindowMetricWithTime(windowSize uint32, now int64) *timeWindowMetric {
	ca := &timeWindowMetric{
		capacity: windowSize,
		buckets:  make([]*bucket, windowSize),
		total:    &counter{},
	}

	startTime := now
	for i := int(windowSize) - 1; i >= 0; i-- {
		b := &bucket{
			startTime: startTime,
			counter:   counter{},
		}
		ca.buckets[i%int(windowSize)] = b
		startTime--

	}
	ca.endIndex = int(windowSize) - 1
	return ca
}

func (t *timeWindowMetric) String() string {
	return fmt.Sprintf("cap:%v,buckets:%v,endIndex:%v,total:%v", t.capacity, t.buckets, t.endIndex, t.total)
}

func (t *timeWindowMetric) moveWindowToCurrentSecond(now int64) *bucket {
	currEpochSecond := now
	latestBucket := t.buckets[t.endIndex]
	differenceInSeconds := currEpochSecond - atomic.LoadInt64(&latestBucket.startTime)
	if differenceInSeconds == 0 {
		return latestBucket
	}

	secondsToMoveWindow := int64min(differenceInSeconds, int64(t.capacity))
	var currentBucket *bucket
	for {
		secondsToMoveWindow--
		t.endIndex = (t.endIndex + 1) % int(t.capacity)
		currentBucket = t.buckets[t.endIndex]
		t.total.subtract(&currentBucket.counter)
		currentBucket.reset(currEpochSecond - secondsToMoveWindow)
		if secondsToMoveWindow <= 0 {
			if secondsToMoveWindow < 0 {
				log.Printf("secondsToMoveWindow_below_0")
			}
			break
		}
	}
	return currentBucket
}

func (t *timeWindowMetric) update(outcome windowOutcome) WindowSnapshot {
	t.updateLock.Lock()
	defer t.updateLock.Unlock()

	now := currentTimeInSecond()
	t.moveWindowToCurrentSecond(now).add(outcome)
	t.total.add(outcome)
	return newSnapshot(t.total)
}

func (t *timeWindowMetric) getSnapshot() WindowSnapshot {
	t.updateLock.Lock()
	defer t.updateLock.Unlock()

	now := currentTimeInSecond()
	t.moveWindowToCurrentSecond(now)
	return newSnapshot(t.total)
}

func int64min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

type iWindowMetric interface {
	update(outcome windowOutcome) WindowSnapshot
	getSnapshot() WindowSnapshot
}
