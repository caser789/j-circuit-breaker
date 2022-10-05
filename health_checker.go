package circuitbreaker

type healthChecker uint32

const (
	timeoutPercentChecker healthChecker = iota + 1
	errorPercentChecker
)

type iSnapshot interface {
	GetTimeoutPercent() int
	GetErrorPercent() int
	GetTotalCount() int64
}

type iConcurrentCallGetter interface {
	iSnapshot
	GetNumConcurrentCalls() int
}

type iSetting interface {
	GetTimeoutPercentThreshold() int
	GetErrorPercentThreshold() int
}

func (h healthChecker) isThresholdExceed(snapshot iSnapshot, setting iSetting) (bool, transitionReason) {
	switch h {
	case timeoutPercentChecker:
		if snapshot.GetTimeoutPercent() >= setting.GetTimeoutPercentThreshold() {
			return true, timeoutThresholdExceeded
		}
	case errorPercentChecker:
		if snapshot.GetErrorPercent() >= setting.GetErrorPercentThreshold() {
			return true, errorThresholdExceeded
		}
	default:
		return false, invalidTransition
	}
	return false, invalidTransition
}

type stateSnapshot struct {
	WindowSnapshot
	NumConcurrentCalls int
}

func (s stateSnapshot) GetNumConcurrentCalls() int {
	return s.NumConcurrentCalls
}

func getStateSnapshot(state iState, cl concurrencyLimiter) iConcurrentCallGetter {
	windowSnapshot := state.getWindowMetrics().getSnapshot()
	concurrencyCount := cl.getCount(state.shouldCheckForMaxConcurrency())
	return stateSnapshot{
		WindowSnapshot:     windowSnapshot,
		NumConcurrentCalls: concurrencyCount,
	}
}
