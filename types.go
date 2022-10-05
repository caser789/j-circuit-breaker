package circuitbreaker

// EventListener is called when events are emitted. Currently, only supports StateTransitionEvents
type EventListener func(event iEvent)

// RunFunc is ran when current state of CircuitBreaker is in CLOSED or SEMI_OPEN state.
type RunFunc func() error
