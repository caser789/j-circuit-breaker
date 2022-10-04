package circuitbreaker

import (
	"sync"
	"time"
)

// State models the different possible states of a CircuitBreaker
type State int32

const (
	// Closed state allows all calls to execute and transitions to Open
	// if error or timeout rate have exceeded the thresholds.
	Closed State = iota + 1
	// Open state blocks all calls from executing for a configurable amount of time.
	Open
	// SemiOpen state allows a number of requests through to probe if called service has recovered.
	SemiOpen
	// ForcedOpen is the same as Open but remains until reset() or another forced transition.
	ForcedOpen
	// ForcedClosed is the same as Closed but remains until reset() or another forced transition.
	ForcedClosed
)

func (state State) String() string {
	switch state {
	case Closed:
		return "CLOSED"
	case Open:
		return "OPEN"
	case SemiOpen:
		return "SEMI_OPEN"
	case ForcedOpen:
		return "FORCED_OPEN"
	case ForcedClosed:
		return "FORCED_CLOSED"
	default:
		return "INVALID_STATE"
	}
}

type transitionType int32

const (
	onRequestStart = iota + 1
	onRequestEnd
)

type iState interface {
	acquirePermission() error
	releasePermission()
	onError(isTimeout bool) WindowSnapshot
	onSuccess(isTimeout bool) WindowSnapshot
	getStateType() State
	getWindowMetrics() iWindowMetric
	shouldCheckForMaxConcurrency() bool

	// shouldStateTransition checks the snapshot against the configured thresholds and returns true if cb should transition
	// or false otherwise. Also returns which State to transition to and the transitionReason.
	shouldStateTransition(snapshot WindowSnapshot, transitionType transitionType) (State, transitionReason, bool)
}

type stateBase struct {
	windowMetrics iWindowMetric
	setting       setting
}

func newClosedState(setting setting) *closedState {
	return &closedState{
		stateBase{
			windowMetrics: newTimeWindowMetricWithTime(uint32(setting.SlidingWindowSize), time.Now().Unix()),
			setting:       setting,
		},
	}
}

type closedState struct {
	stateBase
}

func (c *closedState) acquirePermission() error { return nil }
func (c *closedState) releasePermission()       {}
func (c *closedState) onError(isTimeout bool) WindowSnapshot {
	var snapshot WindowSnapshot
	if isTimeout {
		snapshot = c.windowMetrics.update(timeoutErrorOutcome)
	} else {
		snapshot = c.windowMetrics.update(errorOutcome)
	}
	return snapshot
}
func (c *closedState) onSuccess(isTimeout bool) WindowSnapshot {
	var snapshot WindowSnapshot
	if isTimeout {
		snapshot = c.windowMetrics.update(timeoutSuccessOutcome)
	} else {
		snapshot = c.windowMetrics.update(successOutcome)
	}
	return snapshot
}
func (c *closedState) getStateType() State                { return Closed }
func (c *closedState) getWindowMetrics() iWindowMetric    { return c.windowMetrics }
func (c *closedState) shouldCheckForMaxConcurrency() bool { return true }
func (c *closedState) shouldStateTransition(snapshot WindowSnapshot, transitionType transitionType) (State, transitionReason, bool) {
	if transitionType != onRequestEnd {
		return Closed, invalidTransition, false
	}

	if int(snapshot.TotalCount) >= c.setting.MinimumNumberOfCalls {
		if exceeded, reason := c.setting.isThresholdsExceed(&snapshot); exceeded {
			return Open, reason, true
		}
	}

	return Closed, invalidTransition, false
}

func newOpenState(setting setting) *openState {
	return &openState{
		stateBase: stateBase{
			windowMetrics: newTimeWindowMetricWithTime(uint32(setting.SlidingWindowSize), time.Now().Unix()),
		},
		waitTimer: loadClock().timer(setting.WaitDurationInOpenState),
	}
}

type openState struct {
	stateBase
	waitTimer *timer
}

func (o *openState) acquirePermission() error {
	return ErrCircuitOpenState
}
func (o *openState) releasePermission() {}
func (o *openState) onError(isTimeout bool) WindowSnapshot {
	return o.windowMetrics.getSnapshot()
}
func (o *openState) onSuccess(isTimeout bool) WindowSnapshot {
	return o.windowMetrics.getSnapshot()
}
func (o *openState) getStateType() State {
	return Open
}
func (o *openState) getWindowMetrics() iWindowMetric {
	return o.windowMetrics
}
func (o *openState) shouldCheckForMaxConcurrency() bool { return true }
func (o *openState) shouldStateTransition(snapshot WindowSnapshot, transitionType transitionType) (State, transitionReason, bool) {
	if transitionType != onRequestStart {
		return Open, invalidTransition, false
	}

	select {
	case <-o.waitTimer.C:
		return SemiOpen, waitInOpenElapsed, true
	default:
		return Open, invalidTransition, false
	}
}

func newSemiOpenState(setting setting) *semiOpenState {
	numCalls := uint32(setting.NumberOfCallsInSemiOpenState)
	return &semiOpenState{
		stateBase: stateBase{
			windowMetrics: newCountWindowMetric(numCalls),
			setting:       setting,
		},
		callThreshold: numCalls,
		callCount:     0,
	}
}

type semiOpenState struct {
	stateBase
	callThreshold uint32
	callCount     uint32
	callCountLock sync.Mutex
}

func (s *semiOpenState) acquirePermission() error {
	s.callCountLock.Lock()
	defer s.callCountLock.Unlock()

	if s.callCount < s.callThreshold {
		s.callCount++
		return nil
	}

	return ErrMaxPermittedCallsInSemiOpenState
}
func (s *semiOpenState) releasePermission() {
	s.callCountLock.Lock()
	defer s.callCountLock.Unlock()
	s.callCount--
}
func (s *semiOpenState) onError(isTimeout bool) WindowSnapshot {
	var snapshot WindowSnapshot
	if isTimeout {
		snapshot = s.windowMetrics.update(timeoutErrorOutcome)
	} else {
		snapshot = s.windowMetrics.update(errorOutcome)
	}
	return snapshot
}
func (s *semiOpenState) onSuccess(isTimeout bool) WindowSnapshot {
	var snapshot WindowSnapshot
	if isTimeout {
		snapshot = s.windowMetrics.update(timeoutSuccessOutcome)
	} else {
		snapshot = s.windowMetrics.update(successOutcome)
	}
	return snapshot
}
func (s *semiOpenState) getStateType() State                { return SemiOpen }
func (s *semiOpenState) getWindowMetrics() iWindowMetric    { return s.windowMetrics }
func (s *semiOpenState) shouldCheckForMaxConcurrency() bool { return true }
func (s *semiOpenState) shouldStateTransition(snapshot WindowSnapshot, transitionType transitionType) (State, transitionReason, bool) {
	if transitionType != onRequestEnd {
		return SemiOpen, invalidTransition, false
	}

	if int(snapshot.TotalCount) >= min(s.setting.MinimumNumberOfCalls, s.setting.NumberOfCallsInSemiOpenState) {
		if exceeded, reason := s.setting.isThresholdsExceed(snapshot); exceeded {
			return Open, reason, true
		}

		return Closed, failurePercentsBelowThresholds, true
	}

	return SemiOpen, invalidTransition, false
}

func newForcedClosedState(setting setting) *forcedClosedState {
	return &forcedClosedState{
		stateBase: stateBase{
			windowMetrics: newTimeWindowMetricWithTime(uint32(setting.SlidingWindowSize), time.Now().Unix()),
			setting:       setting,
		},
	}
}

type forcedClosedState struct {
	stateBase
}

func (f *forcedClosedState) acquirePermission() error { return nil }
func (f *forcedClosedState) releasePermission()       {}
func (f *forcedClosedState) onError(isTimeout bool) WindowSnapshot {
	var snapshot WindowSnapshot
	if isTimeout {
		snapshot = f.windowMetrics.update(timeoutErrorOutcome)
	} else {
		snapshot = f.windowMetrics.update(errorOutcome)
	}
	return snapshot
}
func (f *forcedClosedState) onSuccess(isTimeout bool) WindowSnapshot {
	var snapshot WindowSnapshot
	if isTimeout {
		snapshot = f.windowMetrics.update(timeoutSuccessOutcome)
	} else {
		snapshot = f.windowMetrics.update(successOutcome)
	}
	return snapshot
}
func (f *forcedClosedState) getStateType() State                { return ForcedClosed }
func (f *forcedClosedState) getWindowMetrics() iWindowMetric    { return f.windowMetrics }
func (f *forcedClosedState) shouldCheckForMaxConcurrency() bool { return false }
func (f *forcedClosedState) shouldStateTransition(snapshot WindowSnapshot, transitionType transitionType) (State, transitionReason, bool) {
	return ForcedClosed, invalidTransition, false
}

func newForcedOpenState(setting setting) *forcedOpenState {
	return &forcedOpenState{
		stateBase: stateBase{
			windowMetrics: newTimeWindowMetricWithTime(uint32(setting.SlidingWindowSize), time.Now().Unix()),
			setting:       setting,
		},
	}
}

type forcedOpenState struct {
	stateBase
}

func (f *forcedOpenState) acquirePermission() error {
	return ErrCircuitForcedOpenState
}
func (f *forcedOpenState) releasePermission() {}
func (f *forcedOpenState) onError(isTimeout bool) WindowSnapshot {
	return f.windowMetrics.getSnapshot()
}
func (f *forcedOpenState) onSuccess(isTimeout bool) WindowSnapshot {
	return f.windowMetrics.getSnapshot()
}
func (f *forcedOpenState) getStateType() State                { return ForcedOpen }
func (f *forcedOpenState) getWindowMetrics() iWindowMetric    { return f.windowMetrics }
func (f *forcedOpenState) shouldCheckForMaxConcurrency() bool { return false }
func (f *forcedOpenState) shouldStateTransition(snapshot WindowSnapshot, transitionType transitionType) (State, transitionReason, bool) {
	return ForcedOpen, invalidTransition, false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
