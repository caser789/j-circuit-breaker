package circuitbreaker

import "sync"

// FallbackFunc is ran when runFunc returns an error
// or when CircuitBreaker blocks runFunc due to being in OPEN state or max concurrency level has been reached.
type FallbackFunc func(error) error

// EntryOption is an option for CircuitBreaker.EnterWithContext.
type EntryOption interface {
	applyEntryOption(*option)
}

// ProtectOption is an option for CircuitBreaker.ProtectWithContext.
type ProtectOption interface {
	applyProtectOption(*option)
}

// ProtectEntryOption is an option that can be used for either CircuitBreaker.EnterWithContext or CircuitBreaker.ProtectWithContext.
type ProtectEntryOption interface {
	EntryOption
	ProtectOption
}

// WithAmplifyingFactor will modify the call's weight when being recorded in the Concurrency Limiter.
// This option can be used in both Protect and Enter but only applies for the Concurrency Limiter.
// If set to 0, then the concurrency count will not increase.
func WithAmplifyingFactor(factor uint32) ProtectEntryOption {
	return amplifyingFactorOption{
		amplifyFactor: factor,
	}
}

type amplifyingFactorOption struct {
	amplifyFactor uint32
}

func (a amplifyingFactorOption) applyEntryOption(o *option) {
	o.amplifyFactor = a.amplifyFactor
}

func (a amplifyingFactorOption) applyProtectOption(o *option) {
	o.amplifyFactor = a.amplifyFactor
}

// WithFallbackFunction sets the FallbackFunc for this request.
// FallbackFunc is ran when runFunc returns an error
// or when CircuitBreaker blocks runFunc due to being in OPEN state or max concurrency level has been reached.
func WithFallbackFunction(fallbackFunc FallbackFunc) ProtectOption {
	return fallbackFuncOption{fallbackFunc: fallbackFunc}
}

type fallbackFuncOption struct {
	fallbackFunc FallbackFunc
}

func (f fallbackFuncOption) applyProtectOption(o *option) {
	o.fallbackFunc = f.fallbackFunc
}

var optionPool = sync.Pool{
	New: func() interface{} {
		return &option{
			amplifyFactor: 1,
			fallbackFunc:  nil,
		}
	},
}

type option struct {
	amplifyFactor uint32
	fallbackFunc  FallbackFunc
}

func (o *option) reset() {
	o.amplifyFactor = 1
	o.fallbackFunc = nil
}

func newOption() *option {
	o, _ := optionPool.Get().(*option)
	o.reset()
	return o
}

func recycleOption(option *option) {
	optionPool.Put(option)
}
