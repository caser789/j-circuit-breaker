package circuitbreaker

import (
	"context"
	"sync"
	"time"
)

type iBreaker interface {
	ErrorAndFallback(ctx context.Context, fallback FallbackFunc, err error, stats *RequestStats) error
	StateTransition(ctx context.Context, fromStateRef iState, to State, reason transitionReason) iState
	OnRequestComplete(isTimeout bool, requestStats RequestStats, snapshot iConcurrentCallGetter, endTime time.Time)
}

// Entry is returned by CircuitBreaker.Enter if the current CircuitBreaker state is healthy.
type Entry struct {
	cb        iBreaker
	entryTime time.Time
	setting   setting
	state     iState
	context   context.Context
	factor    uint32

	concurrencyLimiter concurrencyLimiter
	exitOnce           sync.Once
}

// Exit should be called to close an Entry.
// Exit can only be called once for each Entry.
// The runError parameter is the error returned by the dependency call.
func (e *Entry) Exit(runError error) {
	e.exitOnce.Do(func() {
		_ = e.exitWithFallback(e.context, runError, nil, e.factor)
	})
}

func (e *Entry) exitWithFallback(ctx context.Context, runErr error, fb FallbackFunc, factor uint32) error {
	var windowSnap WindowSnapshot
	var requestStats RequestStats
	var err error

	e.concurrencyLimiter.release(ctx, int64(factor))

	endTime := currentTime(e.setting.UseCachedTime)
	elapsed := endTime.Sub(e.entryTime)
	isTimeout := e.setting.isTimeout(elapsed)
	requestStats.Elapsed = elapsed

	if runErr != nil {
		windowSnap = e.state.onError(isTimeout)
		err = e.cb.ErrorAndFallback(ctx, fb, runErr, &requestStats)
	} else {
		windowSnap = e.state.onSuccess(isTimeout)
	}

	if toState, reason, shouldTransition := e.state.shouldStateTransition(windowSnap, onRequestEnd); shouldTransition {
		e.cb.StateTransition(ctx, e.state, toState, reason)
	}

	e.cb.OnRequestComplete(isTimeout, requestStats, stateSnapshot{
		WindowSnapshot:     windowSnap,
		NumConcurrentCalls: e.concurrencyLimiter.getCount(e.state.shouldCheckForMaxConcurrency()),
	}, endTime)
	return err
}
