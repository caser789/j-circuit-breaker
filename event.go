package circuitbreaker

import "time"

// iEvent is the basic interface for the events.
type iEvent interface {
	EventType() EventType
	CircuitBreakerName() string
	CreationTime() time.Time
	StateSnapshot() iConcurrentCallGetter
}

// EventType are the different types of event to be emitted through the event listener.
type EventType uint32

const (
	// StateTransitionEventType is the type of StateTransitionEvent
	StateTransitionEventType EventType = iota + 1
	// OnRequestCompleteEventType is the type of OnRequestCompleteEvent
	OnRequestCompleteEventType
)

func (e EventType) String() string {
	switch e {
	case StateTransitionEventType:
		return "state_transition"
	case OnRequestCompleteEventType:
		return "on_request_complete"
	default:
		return "unknown_event"
	}
}

// StateTransitionEvent is emitted after a state transition
type StateTransitionEvent struct {
	PrevState     State
	CurrentState  State
	name          string
	creationTime  time.Time
	stateSnapshot iConcurrentCallGetter
}

// EventType returns the event type.
func (s StateTransitionEvent) EventType() EventType {
	return StateTransitionEventType
}

// CircuitBreakerName returns the circuitbreaker name.
func (s StateTransitionEvent) CircuitBreakerName() string {
	return s.name
}

// CreationTime returns the creation time of the event.
func (s StateTransitionEvent) CreationTime() time.Time {
	return s.creationTime
}

// StateSnapshot returns the state snapshot of the circuitbreaker.
func (s StateTransitionEvent) StateSnapshot() iConcurrentCallGetter {
	return s.stateSnapshot
}

// OnRequestCompleteEvent is emitted after every request is completed (regardless whether successful or not).
type OnRequestCompleteEvent struct {
	Stats         RequestStats
	name          string
	creationTime  time.Time
	stateSnapshot iConcurrentCallGetter
}

// EventType returns the event type.
func (o OnRequestCompleteEvent) EventType() EventType {
	return OnRequestCompleteEventType
}

// CircuitBreakerName returns the circuitbreaker name.
func (o OnRequestCompleteEvent) CircuitBreakerName() string {
	return o.name
}

// CreationTime returns the creation time of the event.
func (o OnRequestCompleteEvent) CreationTime() time.Time {
	return o.creationTime
}

// StateSnapshot returns the state snapshot of the circuitbreaker.
func (o OnRequestCompleteEvent) StateSnapshot() iConcurrentCallGetter {
	return o.stateSnapshot
}

// RequestStats models the stats for a completed request.
type RequestStats struct {
	Elapsed        time.Duration
	RequestOutcome Outcome
	RunError       error
	FallbackError  error
}

// Outcome is the outcome of a request.
type Outcome uint32

const (
	// SuccessOutcome when call was successful with no errors but the duration exceeded the configured TimeoutThreshold.
	SuccessOutcome Outcome = iota + 1
	// SuccessTimeoutOutcome when call was successful with no errors but the duration exceeded the configured TimeoutThreshold.
	SuccessTimeoutOutcome
	// ErrorOutcome when call completed with an error returned.
	ErrorOutcome
	// ErrorTimeoutOutcome when call completed with an error returned and duration exceeded the configued TimeoutThreshold.
	ErrorTimeoutOutcome
	// MaxConcurrencyRejectedOutcome when call was rejected due to the number of concurrent calls
	// has already reached the configured MaxConcurrentCalls for this CircuitBreaker.
	MaxConcurrencyRejectedOutcome
	// MaxPermittedCallsInSemiOpenRejectedOutcome when call was rejected due to the number of calls
	// has already reached the configured NumberOfCallsInSemiOpenState while in SEMI_OPEN state.
	MaxPermittedCallsInSemiOpenRejectedOutcome
	// OpenStateRejectedOutcome when call was rejected due to the CircuitBreaker being in OPEN state.
	OpenStateRejectedOutcome
	// ForcedOpenStateRejectedOutcome when call was rejected due to the CircuitBreaker being in OPEN state.
	ForcedOpenStateRejectedOutcome
)

func (o Outcome) String() string {
	switch o {
	case SuccessOutcome:
		return "success"
	case SuccessTimeoutOutcome:
		return "success_timeout"
	case ErrorOutcome:
		return "error"
	case ErrorTimeoutOutcome:
		return "error_timeout"
	case MaxConcurrencyRejectedOutcome:
		return "max_concurrency_rejected"
	case MaxPermittedCallsInSemiOpenRejectedOutcome:
		return "max_permitted_in_semi_open_rejected"
	case OpenStateRejectedOutcome:
		return "open_state_rejected"
	case ForcedOpenStateRejectedOutcome:
		return "forced_open_state_rejected"
	default:
		return "unknown_outcome"
	}
}
