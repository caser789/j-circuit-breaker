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

// CircuitError models the different types of failures in execution due to the current CircuitBreaker state.
// CircuitError is returned by the Protect() function when the RunFunc is blocked from executing.
type CircuitError struct {
	isCircuitOpen bool
	message       string
}

func (ce CircuitError) Error() string {
	return "circuitbreaker:" + ce.message
}

// IsCircuitOpen returns true for OPEN and FORCED_OPEN state produced errors. This is a way to generalise the
// specific error types
func (ce CircuitError) IsCircuitOpen() bool {
	return ce.isCircuitOpen
}

var (
	// ErrMaxConcurrency is returned when the number of concurrent calls has already reached MaxConcurrentCalls for this CircuitBreaker.
	ErrMaxConcurrency = CircuitError{message: "max_concurrency"}
	// ErrCircuitOpenState is returned when CircuitBreaker is in OPEN state.
	ErrCircuitOpenState = CircuitError{message: "circuit_open", isCircuitOpen: true}
	// ErrCircuitForcedOpenState is returned when CircuitBreaker is in FORCED_OPEN state.
	ErrCircuitForcedOpenState = CircuitError{message: "circuit_forced_open", isCircuitOpen: true}
	// ErrMaxPermittedCallsInSemiOpenState is returned when max number of permitted calls is reached while in SEMI_OPEN state
	ErrMaxPermittedCallsInSemiOpenState = CircuitError{message: "max_permitted_in_semi_open"}

	// ErrInvalidConfig is returned when creating or updating a CircuitBreaker using invalid configurations.
	ErrInvalidConfig = CircuitError{message: "invalid_circuitbreaker_config"}

	// ErrBreakerNotFound is returned when CircuitBreaker with the given name does not exist.
	ErrBreakerNotFound = CircuitError{message: "breaker_not_found"}
)
