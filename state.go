package circuitbreaker

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
