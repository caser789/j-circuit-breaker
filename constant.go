package circuitbreaker

// transitionReason is the reason for state transition
type transitionReason uint32

const (
	// invalidTransition means the state transition should not have happened.
	invalidTransition transitionReason = iota
	// errorThresholdExceeded is when the snapshot error percent exceeds the configured ErrorPercentThreshold.
	errorThresholdExceeded
	// timeoutThresholdExceeded is when the snapshot timeout percent exceeds the configured TimeoutPercentThreshold.
	timeoutThresholdExceeded
	// waitInOpenElapsed is when WaitDurationInOpenState has elapsed since start time in OPEN state.
	waitInOpenElapsed
	// failurePercentsBelowThresholds is when both error and timeout percents are below the configured thresholds so circuitbreaker can transition from SEMI-OPEN to CLOSED.
	failurePercentsBelowThresholds
	// manualTransition is when the state transition is triggered manually e.g. through ForceClose.
	manualTransition
	// reset is when transition is done through the breaker reset() method
	reset
)
