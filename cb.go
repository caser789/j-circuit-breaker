package circuitbreaker

import (
	"context"
	"sync"
	"time"
)

var (
	breakersMutex   sync.RWMutex
	breakersSyncMap sync.Map
)

// CircuitBreaker models the structure of a CircuitBreaker state machine
//
// The CircuitBreaker maintains a finite state machine with five states:
// three normal states: CLOSED, OPEN and SEMI_OPEN; and two special state: FORCED_CLOSED and FORCED_OPEN.
//
// In CLOSED state, the CircuitBreaker tracks the number of errors and timeouts in a sliding window.
// When the error or timeout percentages exceeds the configurable thresholds, the CircuitBreaker transitions from CLOSED to OPEN state,
// and short-circuits further calls to the protected function.
//
// After a specified duration, the CircuitBreaker changes from OPEN to SEMI_OPEN state, allowing a configurable number of calls
// and checks the error/timeout rate of these calls against the thresholds to decide whether to transition to CLOSED or OPEN state.
// Sliding window statistics are reset when transitioning back to CLOSED state.
//
// The special states (FORCED_CLOSED and FORCED_OPEN) can only be reached using state transition function calls.
// The only way to exit from these states is through another force transition or by calling the Reset() method.
// Request outcomes and metrics are still recorded while in FORCED_CLOSED state.
type CircuitBreaker struct {
	name string

	// updateLock guards below fields
	updateLock         sync.RWMutex
	setting            setting
	state              iState
	concurrencyLimiter concurrencyLimiter
	listener           EventListener
}

func newCircuitBreaker(name string, config *Config) (*CircuitBreaker, error) {
	newCb := &CircuitBreaker{
		name: name,
	}

	if config == nil {
		defaultConfig := NewDefaultConfig()
		config = &defaultConfig
	}

	if err := IsValidConfig(config); err != nil {
		return nil, err
	}

	newCb.updateSetting(config)

	return newCb, nil
}

// updateSetting fills default values and sets the setting for this breaker instance.
// The concurrencyLimiter and state of this CircuitBreaker is also re-created.
// Config validation should be done before calling this method.
func (cb *CircuitBreaker) updateSetting(config *Config) {
	cb.updateLock.Lock()
	defer cb.updateLock.Unlock()

	newSetting := fillDefaultsAndGetSetting(*config)
	if newSetting.Config.isEqual(&cb.setting.Config) {
		return
	}

	cb.setting = newSetting
	waitDuration := time.Duration(newSetting.MaxWaitDurationMillis) * time.Millisecond
	cb.concurrencyLimiter = newConcurrencyLimiter(int64(newSetting.MaxConcurrentCalls), waitDuration)
	cb.state = newClosedState(cb.setting)

	if !cb.setting.MetricConfig.IsDisabled {
		setBreakerStateMetric(cb.name, Closed)
		setThresholdsMetrics(cb.name, newSetting.Config)
	}
}

// UpdateConfig updates the config of this CircuitBreaker.
// This will also reset the CircuitBreaker i.e. changes state to closed state and resets the window.
// If the existing config is exactly the same as the new config, no change will be made.
func (cb *CircuitBreaker) UpdateConfig(config *Config) error {
	breakersMutex.Lock()
	defer breakersMutex.Unlock()

	if err := IsValidConfig(config); err != nil {
		return err
	}

	cb.updateSetting(config)
	return nil
}

func (cb *CircuitBreaker) enterWithFallback(ctx context.Context, fb FallbackFunc, factor uint32) (entry *Entry, fbErr error, enterErr error) {
	var requestStats RequestStats
	var windowSnap WindowSnapshot

	cb.updateLock.RLock()
	currState := cb.state
	currSetting := cb.setting
	currLimiter := cb.concurrencyLimiter
	cb.updateLock.RUnlock()

	useCachedTime := currSetting.UseCachedTime

	if toState, reason, shouldTransition := currState.shouldStateTransition(windowSnap, onRequestStart); shouldTransition {
		currState = cb.StateTransition(ctx, currState, toState, reason)
	}

	if stateError := currState.acquirePermission(); stateError != nil {
		fbErr := cb.ErrorAndFallback(ctx, fb, stateError, &requestStats)
		cb.OnRequestComplete(false, requestStats, getStateSnapshot(currState, currLimiter), currentTime(useCachedTime))
		return nil, fbErr, stateError
	}

	err := currLimiter.acquire(ctx, int64(factor))
	if currState.shouldCheckForMaxConcurrency() && err != nil {
		fbErr := cb.ErrorAndFallback(ctx, fb, ErrMaxConcurrency, &requestStats)
		currState.releasePermission()
		cb.OnRequestComplete(false, requestStats, getStateSnapshot(currState, currLimiter), currentTime(useCachedTime))
		return nil, fbErr, ErrMaxConcurrency
	}

	return &Entry{
		cb:                 cb,
		entryTime:          currentTime(useCachedTime),
		setting:            currSetting,
		state:              currState,
		concurrencyLimiter: currLimiter,
		factor:             factor,
		context:            ctx,
	}, nil, nil
}

// ProtectWithContext is the same as Protect but with context and options params
func (cb *CircuitBreaker) ProtectWithContext(ctx context.Context, run RunFunc, opts ...ProtectOption) (err error) {
	opt := newOption()
	defer recycleOption(opt)
	for _, o := range opts {
		o.applyProtectOption(opt)
	}

	entry, fbErr, enterErr := cb.enterWithFallback(ctx, opt.fallbackFunc, opt.amplifyFactor)
	if enterErr != nil {
		return fbErr
	}

	var runErr error

	// We defer here so that the "exit" step will still run even if run() has panicked
	defer func() {
		err = entry.exitWithFallback(ctx, runErr, opt.fallbackFunc, opt.amplifyFactor)
	}()

	runErr = run()
	return runErr
}

// Protect runs the run function if CircuitBreaker is in CLOSED or SEMI_OPEN state.
// In OPEN state, Protect fails fast, runs the fallback function and returns ErrOpenState.
func (cb *CircuitBreaker) Protect(run RunFunc, fallback FallbackFunc) (err error) {
	return cb.ProtectWithContext(context.Background(), run, WithFallbackFunction(fallback))
}

// ForceClose changes the current state of the CircuitBreaker to FORCED_CLOSED.
// MaxConcurrency checking is also disabled.
// Request outcomes (success, error, timeout) continue to be recorded.
func (cb *CircuitBreaker) ForceClose() {
	cb.StateTransition(context.Background(), cb.state, ForcedClosed, manualTransition)
}

// ForceOpen changes the current state of the CircuitBreaker to FORCED_OPEN.
func (cb *CircuitBreaker) ForceOpen() {
	cb.StateTransition(context.Background(), cb.state, ForcedOpen, manualTransition)
}

// Reset resets the CircuitBreaker's sliding window statistics and changes the state to CLOSED.
func (cb *CircuitBreaker) Reset() {
	cb.StateTransition(context.Background(), cb.state, Closed, reset)
}

// EnterWithContext is the same as Enter but with Context and options parameters.
func (cb *CircuitBreaker) EnterWithContext(ctx context.Context, opts ...EntryOption) (*Entry, error) {
	opt := newOption()
	defer recycleOption(opt)
	for _, o := range opts {
		o.applyEntryOption(opt)
	}
	entry, _, enterErr := cb.enterWithFallback(ctx, nil, opt.amplifyFactor)
	return entry, enterErr
}

// Enter should be called before a dependency call to check if the call is allowed to execute or blocked by the CircuitBreaker.
//
// If the call is allowed, an Entry object and a nil error is returned.
// Entry.Exit must be called after the protected call has been executed,
// or after it has panicked.
//
// If the call is blocked, a nil Entry object
// and a CircuitError will be returned depending on the reason for the block.
// Entry.Exit does not have to be called for this case.
func (cb *CircuitBreaker) Enter() (*Entry, error) {
	entry, _, enterErr := cb.enterWithFallback(context.Background(), nil, 1)
	return entry, enterErr
}

func (cb *CircuitBreaker) OnRequestComplete(isTimeout bool, requestStats RequestStats, snapshot iConcurrentCallGetter, endTime time.Time) {
	cb.updateLock.RLock()
	defer cb.updateLock.RUnlock()

	var outcome Outcome
	switch requestStats.RunError {
	case ErrCircuitForcedOpenState:
		outcome = ForcedOpenStateRejectedOutcome
	case ErrMaxPermittedCallsInSemiOpenState:
		outcome = MaxPermittedCallsInSemiOpenRejectedOutcome
	case ErrCircuitOpenState:
		outcome = OpenStateRejectedOutcome
	case ErrMaxConcurrency:
		outcome = MaxConcurrencyRejectedOutcome
	case nil:
		if isTimeout {
			outcome = SuccessTimeoutOutcome
		} else {
			outcome = SuccessOutcome
		}
	default:
		if isTimeout {
			outcome = ErrorTimeoutOutcome
		} else {
			outcome = ErrorOutcome
		}
	}

	requestStats.RequestOutcome = outcome

	if !cb.setting.MetricConfig.IsDisabled {
		addRequestTotalMetric(cb.name, outcome.String())
		setWindowSnapshotMetrics(cb.name, snapshot)
	}

	if cb.listener == nil {
		return
	}

	cb.listener(OnRequestCompleteEvent{
		Stats:         requestStats,
		name:          cb.name,
		creationTime:  endTime,
		stateSnapshot: snapshot,
	})
}

// RegisterEventListener registers an event listener.
func (cb *CircuitBreaker) RegisterEventListener(callback EventListener) {
	cb.updateLock.Lock()
	defer cb.updateLock.Unlock()

	cb.listener = callback
}

func (cb *CircuitBreaker) StateTransition(ctx context.Context, fromStateRef iState, to State, reason transitionReason) iState {
	cb.updateLock.Lock()
	defer cb.updateLock.Unlock()

	if reason != manualTransition && reason != reset {
		if fromStateRef != cb.state {
			return fromStateRef
		}

		if !validAutomaticStateTransitions[fromStateRef.getStateType()][to] {
			return fromStateRef
		}
	}

	from := cb.state.getStateType()
	prevStateSnapshot := getStateSnapshot(cb.state, cb.concurrencyLimiter)

	newState := newStateInstance(to, cb.setting)

	if !cb.setting.MetricConfig.IsDisabled {
		setBreakerStateMetric(cb.name, to)
	}

	cb.state = newState

	if cb.listener != nil {
		cb.listener(StateTransitionEvent{
			PrevState:     from,
			CurrentState:  to,
			name:          cb.name,
			creationTime:  currentTime(cb.setting.UseCachedTime),
			stateSnapshot: prevStateSnapshot,
		})
	}

	return newState
}

// ErrorAndFallback runs the FallbackFunc(if non-nil) and populates RequestStats with RunError and FallbackError.
// If FallbackFunc is not nil, this returns the error from FallbackFunc. Otherwise, returns the original error.
func (cb *CircuitBreaker) ErrorAndFallback(ctx context.Context, fallback FallbackFunc, err error, stats *RequestStats) error {
	switch err {
	case ErrCircuitOpenState,
		ErrCircuitForcedOpenState,
		ErrMaxPermittedCallsInSemiOpenState,
		ErrMaxConcurrency:
	}

	stats.RunError = err

	if fallback == nil {
		stats.FallbackError = nil
		return err
	}

	fbErr := fallback(err)
	stats.FallbackError = fbErr
	return fbErr
}
