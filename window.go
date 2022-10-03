package circuitbreaker

import "sync"

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
