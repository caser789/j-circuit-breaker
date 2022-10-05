//nolint:funlen
package circuitbreaker

import (
	"context"
	"errors"
	"math/rand"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// We define 1 as an error outcome and 0 for success
type outcomes []int

var (
	e = 1
	s = 0
)

func Test_circuitBreaker_updateSetting(t *testing.T) {
	type args struct {
		config *Config
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "update all fields",
			args: args{
				config: &Config{
					ErrorPercentThreshold:        60,
					TimeoutPercentThreshold:      60,
					TimeoutThreshold:             123 * time.Second,
					MinimumNumberOfCalls:         123,
					NumberOfCallsInSemiOpenState: 123,
					WaitDurationInOpenState:      123 * time.Second,
					MaxConcurrentCalls:           123,
					SlidingWindowSize:            123,
				},
			},
			wantErr: false,
		},
	}
	breaker := &CircuitBreaker{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			breaker.updateSetting(tt.args.config)
			if !reflect.DeepEqual(breaker.setting.Config, *tt.args.config) {
				t.Errorf("updateSetting() breakerConfig = %v, wantConfig %v", breaker.setting.Config, tt.args.config)
			}
		})
	}
}

func Test_fillDefaultsAndGetSetting(t *testing.T) {
	type args struct {
		config Config
	}
	defaultConfig := NewDefaultConfig()
	tests := []struct {
		name string
		args args
		want setting
	}{
		{
			name: "empty error and timeout thresholds, should default to only error percent checker",
			args: args{
				config: Config{},
			},
			want: setting{
				Config:   defaultConfig,
				checkers: []healthChecker{errorPercentChecker},
			},
		},
		{
			name: "only timeout threshold",
			args: args{
				config: Config{
					TimeoutPercentThreshold: 50,
				},
			},
			want: setting{
				Config: Config{
					ErrorPercentThreshold:        defaultConfig.ErrorPercentThreshold,
					TimeoutPercentThreshold:      50,
					TimeoutThreshold:             defaultConfig.TimeoutThreshold,
					MinimumNumberOfCalls:         defaultConfig.MinimumNumberOfCalls,
					NumberOfCallsInSemiOpenState: defaultConfig.NumberOfCallsInSemiOpenState,
					WaitDurationInOpenState:      defaultConfig.WaitDurationInOpenState,
					MaxConcurrentCalls:           defaultConfig.MaxConcurrentCalls,
					SlidingWindowSize:            defaultConfig.SlidingWindowSize,
				},
				checkers: []healthChecker{timeoutPercentChecker},
			},
		},
		{
			name: "only error threshold",
			args: args{
				config: Config{
					ErrorPercentThreshold: 60,
				},
			},
			want: setting{
				Config: Config{
					ErrorPercentThreshold:        60,
					TimeoutPercentThreshold:      defaultConfig.TimeoutPercentThreshold,
					TimeoutThreshold:             defaultConfig.TimeoutThreshold,
					MinimumNumberOfCalls:         defaultConfig.MinimumNumberOfCalls,
					NumberOfCallsInSemiOpenState: defaultConfig.NumberOfCallsInSemiOpenState,
					WaitDurationInOpenState:      defaultConfig.WaitDurationInOpenState,
					MaxConcurrentCalls:           defaultConfig.MaxConcurrentCalls,
					SlidingWindowSize:            defaultConfig.SlidingWindowSize,
				},
				checkers: []healthChecker{errorPercentChecker},
			},
		},
		{
			name: "only timeout threshold disabled",
			args: args{
				config: Config{
					TimeoutPercentThreshold: -1,
				},
			},
			want: setting{
				Config: Config{
					ErrorPercentThreshold:        defaultConfig.ErrorPercentThreshold,
					TimeoutPercentThreshold:      defaultConfig.TimeoutPercentThreshold,
					TimeoutThreshold:             defaultConfig.TimeoutThreshold,
					MinimumNumberOfCalls:         defaultConfig.MinimumNumberOfCalls,
					NumberOfCallsInSemiOpenState: defaultConfig.NumberOfCallsInSemiOpenState,
					WaitDurationInOpenState:      defaultConfig.WaitDurationInOpenState,
					MaxConcurrentCalls:           defaultConfig.MaxConcurrentCalls,
					SlidingWindowSize:            defaultConfig.SlidingWindowSize,
				},
				checkers: []healthChecker{errorPercentChecker},
			},
		},
		{
			name: "only error threshold disabled",
			args: args{
				config: Config{
					ErrorPercentThreshold: -1,
				},
			},
			want: setting{
				Config: Config{
					ErrorPercentThreshold:        defaultConfig.ErrorPercentThreshold,
					TimeoutPercentThreshold:      defaultConfig.TimeoutPercentThreshold,
					TimeoutThreshold:             defaultConfig.TimeoutThreshold,
					MinimumNumberOfCalls:         defaultConfig.MinimumNumberOfCalls,
					NumberOfCallsInSemiOpenState: defaultConfig.NumberOfCallsInSemiOpenState,
					WaitDurationInOpenState:      defaultConfig.WaitDurationInOpenState,
					MaxConcurrentCalls:           defaultConfig.MaxConcurrentCalls,
					SlidingWindowSize:            defaultConfig.SlidingWindowSize,
				},
				checkers: []healthChecker{},
			},
		},
		{
			name: "both error threshold and timeout threshold disabled",
			args: args{
				config: Config{
					ErrorPercentThreshold:   -1,
					TimeoutPercentThreshold: -1,
				},
			},
			want: setting{
				Config: Config{
					ErrorPercentThreshold:        defaultConfig.ErrorPercentThreshold,
					TimeoutPercentThreshold:      defaultConfig.TimeoutPercentThreshold,
					TimeoutThreshold:             defaultConfig.TimeoutThreshold,
					MinimumNumberOfCalls:         defaultConfig.MinimumNumberOfCalls,
					NumberOfCallsInSemiOpenState: defaultConfig.NumberOfCallsInSemiOpenState,
					WaitDurationInOpenState:      defaultConfig.WaitDurationInOpenState,
					MaxConcurrentCalls:           defaultConfig.MaxConcurrentCalls,
					SlidingWindowSize:            defaultConfig.SlidingWindowSize,
				},
				checkers: []healthChecker{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fillDefaultsAndGetSetting(tt.args.config); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fillDefaultsAndGetSetting() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_circuitBreaker_stateTransition(t *testing.T) {
	type args struct {
		to State
	}

	cb := mustInitCB(t, nil)

	tests := []struct {
		name string
		args args
		want State
		from iState
	}{
		{
			name: "closed to closed",
			args: args{
				to: Closed,
			},
			from: newClosedState(cb.setting),
			want: Closed,
		},
		{
			name: "closed to open",
			args: args{
				to: Open,
			},
			from: newClosedState(cb.setting),
			want: Open,
		},
		{
			name: "open to semi_open",
			args: args{
				to: SemiOpen,
			},
			from: newOpenState(cb.setting),
			want: SemiOpen,
		},
		{
			name: "open to closed, possible through Reset()",
			args: args{
				to: Closed,
			},
			from: newOpenState(cb.setting),
			want: Closed,
		},
		{
			name: "open to forced_closed",
			args: args{
				to: ForcedClosed,
			},
			from: newOpenState(cb.setting),
			want: ForcedClosed,
		},
		{
			name: "semi-open to forced_closed",
			args: args{
				to: ForcedClosed,
			},
			from: newSemiOpenState(cb.setting),
			want: ForcedClosed,
		},
		{
			name: "closed to forced_open",
			args: args{
				to: ForcedOpen,
			},
			from: newClosedState(cb.setting),
			want: ForcedOpen,
		},
		{
			name: "semi-open to forced_open",
			args: args{
				to: ForcedOpen,
			},
			from: newSemiOpenState(cb.setting),
			want: ForcedOpen,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb.state = tt.from

			cb.StateTransition(context.Background(), cb.state, tt.args.to, manualTransition)
			currState := cb.state.getStateType()
			if currState != tt.want {
				t.Errorf("stateTransition actual %v, want = %v", currState, tt.want)
			}
		})
	}
}

func Test_circuitBreaker_Reset(t *testing.T) {
	breaker := mustInitCB(t, nil)

	tests := []struct {
		name string
		from iState
	}{
		{
			name: "from closed",
			from: newClosedState(breaker.setting),
		},
		{
			name: "from open",
			from: newOpenState(breaker.setting),
		},
		{
			name: "from semi-open",
			from: newSemiOpenState(breaker.setting),
		},
		{
			name: "from forced-closed",
			from: newForcedClosedState(breaker.setting),
		},
		{
			name: "from forced-open",
			from: newForcedOpenState(breaker.setting),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			breaker.state = tt.from

			// Add a single record in the window
			breaker.state.getWindowMetrics().update(successOutcome)

			breaker.Reset()

			snapshot := getStateSnapshot(breaker.state, breaker.concurrencyLimiter)
			if snapshot.GetTotalCount() != 0 {
				t.Errorf("window should be reset")
			}
			currState := breaker.state.getStateType()
			if currState != Closed {
				t.Errorf("Reset() to closed actual %v, want = %v", currState, Closed)
			}
		})
	}
}

func Test_circuitBreaker_Protect_EnterExit_extraCases(t *testing.T) {
	tests := []struct {
		name   string
		testFn func(*testing.T, cbAPIWrapper)
	}{
		{
			name:   "max concurrency",
			testFn: testMaxConcurrency,
		},
		{
			name:   "err semi open max permitted calls",
			testFn: testErrMaxPermittedCallsInSemiOpen,
		},
		{
			name:   "max concurrency; release permission",
			testFn: testMaxConcurrencyReleasePermissionInSemiOpen,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, wrapper := range apiWrappers {
				t.Run(wrapper.name, func(t *testing.T) {
					tt.testFn(t, wrapper.fn)
				})
			}
		})
	}
}

func testMaxConcurrency(t *testing.T, cbWrapperFunc cbAPIWrapper) {
	breaker := mustInitCB(t, &Config{MaxConcurrentCalls: 1})

	listenerCheck := testEventListener(t, breaker, &RequestStats{
		RequestOutcome: MaxConcurrencyRejectedOutcome,
		RunError:       ErrMaxConcurrency,
	}, nil)
	defer listenerCheck()

	_, _, err := runBlockingAndNonBlockingRequest(breaker, cbWrapperFunc, nil, nil)

	if err != ErrMaxConcurrency {
		t.Errorf("Protect() error = %v, wantErr %v", err, ErrMaxConcurrency)
	}
}

func testErrMaxPermittedCallsInSemiOpen(t *testing.T, cbAPIWrapperFunc cbAPIWrapper) {
	breaker := mustInitCB(t, &Config{NumberOfCallsInSemiOpenState: 1})

	breaker.StateTransition(context.Background(), breaker.state, SemiOpen, manualTransition)

	listenerCheck := testEventListener(t, breaker, &RequestStats{
		RequestOutcome: MaxPermittedCallsInSemiOpenRejectedOutcome,
		RunError:       ErrMaxPermittedCallsInSemiOpenState,
	}, nil)
	defer listenerCheck()

	_, _, err := runBlockingAndNonBlockingRequest(breaker, cbAPIWrapperFunc, nil, nil)

	if err != ErrMaxPermittedCallsInSemiOpenState {
		t.Errorf("error = %v, expected %v", err, ErrMaxPermittedCallsInSemiOpenState)
	}

	if breaker.state.getStateType() != SemiOpen {
		t.Errorf("currState = %v, expected %v", breaker.state.getStateType(), SemiOpen)
	}
}

// This tests the case where in SEMI_OPEN state, a call has passed the state check, but failed concurrency check.
// The call should release permission, so that it can decrement the NumberOfCallsInSemiOpenState counter
// to allow other requests to acquire the unused permission.
func testMaxConcurrencyReleasePermissionInSemiOpen(t *testing.T, cbAPIWrapper cbAPIWrapper) {
	breaker := mustInitCB(t, &Config{NumberOfCallsInSemiOpenState: 2, MaxConcurrentCalls: 1})

	breaker.StateTransition(context.Background(), breaker.state, SemiOpen, manualTransition)
	resume, done, err := runBlockingAndNonBlockingRequest(breaker, cbAPIWrapper, nil, nil)

	if err != ErrMaxConcurrency {
		t.Errorf("err = %v, expected %v", err, ErrMaxConcurrency)
	}

	resume <- struct{}{}
	<-done

	if breaker.state.getStateType() != SemiOpen {
		t.Errorf("currState = %v, expected %v", breaker.state.getStateType(), SemiOpen)
	}

	// This call should not be blocked and will cause a state transition to Closed state.
	err = cbAPIWrapper(breaker, func() error {
		return nil
	}, nil)

	if err != nil {
		t.Errorf("error = %v, expected %v", err, nil)
	}

	if breaker.state.getStateType() != Closed {
		t.Errorf("currState = %v, expected %v", breaker.state.getStateType(), Closed)
	}

}

func Test_circuitBreaker_automatic_stateTransitions(t *testing.T) {
	tests := []struct {
		name             string
		config           *Config
		outcomes         outcomes
		startState       State
		expectedNewState State
	}{
		{
			name: "closed to open on error",
			config: &Config{
				ErrorPercentThreshold: 1,
				MinimumNumberOfCalls:  1,
			},
			outcomes:         outcomes{e},
			startState:       Closed,
			expectedNewState: Open,
		},
		{
			name: "closed to open on success",
			config: &Config{
				ErrorPercentThreshold: 1,
				MinimumNumberOfCalls:  2,
			},
			outcomes:         outcomes{e, s},
			startState:       Closed,
			expectedNewState: Open,
		},
		{
			name: "semi_open to open on error MinimumNumberOfCalls < NumberOfCallsInSemiOpenState",
			config: &Config{
				ErrorPercentThreshold:        1,
				MinimumNumberOfCalls:         1,
				NumberOfCallsInSemiOpenState: 2,
			},
			outcomes:         outcomes{e},
			startState:       SemiOpen,
			expectedNewState: Open,
		},
		{
			name: "semi_open to open on error MinimumNumberOfCalls > NumberOfCallsInSemiOpenState",
			config: &Config{
				ErrorPercentThreshold:        1,
				MinimumNumberOfCalls:         2,
				NumberOfCallsInSemiOpenState: 1,
			},
			outcomes:         outcomes{e},
			startState:       SemiOpen,
			expectedNewState: Open,
		},
		{
			name: "semi_open to open on success",
			config: &Config{
				ErrorPercentThreshold:        1,
				MinimumNumberOfCalls:         2,
				NumberOfCallsInSemiOpenState: 2,
			},
			outcomes:         outcomes{e, s},
			startState:       SemiOpen,
			expectedNewState: Open,
		},
		{
			name: "semi_open to closed on success",
			config: &Config{
				ErrorPercentThreshold:        1,
				MinimumNumberOfCalls:         1,
				NumberOfCallsInSemiOpenState: 1,
			},
			outcomes:         outcomes{s},
			startState:       SemiOpen,
			expectedNewState: Closed,
		},
		{
			name: "semi_open to closed on error",
			config: &Config{
				ErrorPercentThreshold:        60,
				MinimumNumberOfCalls:         2,
				NumberOfCallsInSemiOpenState: 2,
			},
			outcomes:         outcomes{s, e},
			startState:       SemiOpen,
			expectedNewState: Closed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, apiWrapper := range apiWrappers {
				t.Run(apiWrapper.name, func(t *testing.T) {
					breaker := mustInitCB(t, tt.config)

					breaker.StateTransition(context.Background(), breaker.state, tt.startState, manualTransition)

					check := testEventListener(t, breaker, nil, &StateTransitionEvent{
						PrevState:    tt.startState,
						CurrentState: tt.expectedNewState,
					})
					defer check()

					for _, i := range tt.outcomes {
						_ = apiWrapper.fn(breaker, func() error {
							if i == e {
								return errors.New("runErr")
							}
							return nil
						}, nil)
					}

					if breaker.state.getStateType() != tt.expectedNewState {
						t.Errorf("currState = %v, expected %v", breaker.state.getStateType(), tt.expectedNewState)
					}
				})

			}
		})
	}

}

func Test_circuitBreaker_withAmplifyingFactor(t *testing.T) {
	config := &Config{
		ErrorPercentThreshold:        20,
		MinimumNumberOfCalls:         1,
		NumberOfCallsInSemiOpenState: 5,
		WaitDurationInOpenState:      10 * time.Second,
		MaxConcurrentCalls:           10, // MaxConcurrentCalls set to 10
		SlidingWindowSize:            10,
	}
	runnerFunc := func(wrapper cbAPIWrapperWithContext) {
		p := mustInitCB(t, config)

		callOneUnblock := make(chan struct{})
		callOneStarted := make(chan struct{})
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			_ = wrapper(p, func() error {
				callOneStarted <- struct{}{}
				<-callOneUnblock
				return nil
			}, WithAmplifyingFactor(10)) // Pass in factor of 10. This single request should saturate the concurrency limiter.
			wg.Done()
		}()

		<-callOneStarted // continue only after call 1 has begun.

		var finalErr error
		wg.Add(1)
		go func() {
			// Second call should be rejected due to max concurrency.
			finalErr = wrapper(p, func() error {
				return nil
			}, WithAmplifyingFactor(10))

			callOneUnblock <- struct{}{} // resume call 1
			wg.Done()
		}()
		wg.Wait()
		assert.NotNil(t, finalErr)
	}
	runProtectorAPIs(t, runnerFunc)
}

// Three calls are executed, with the second call panicking.
// Protect should be able to recover in the case of panicked runFunc.
func Test_circuitBreaker_Protect_panic_on_semiOpen(t *testing.T) {
	config := &Config{
		ErrorPercentThreshold:        80,
		MinimumNumberOfCalls:         2,
		NumberOfCallsInSemiOpenState: 2,
	}
	breaker := mustInitCB(t, config)

	breaker.StateTransition(context.Background(), breaker.state, SemiOpen, manualTransition)

	// We'll run three calls
	// 1st call returns nil
	// 2nd call panics, recovers and returns error. CB update state to CLOSED since error rate is below threshold.
	// 3rd call passes.
	for i := 0; i < 3; i++ {
		i := i
		panicI := 1
		err := func() error {
			defer func() {
				_ = recover()
			}()
			err := breaker.Protect(func() error {
				if i == panicI {
					panic("panic!")
				}
				return nil
			}, nil)
			return err
		}()

		if i != panicI && err != nil {
			t.Errorf("unexpected error = %v", err)
		}
	}

	if breaker.state.getStateType() != Closed {
		t.Errorf("currState = %v, expected %v", breaker.state.getStateType(), Closed)
	}

}

func Test_circuitBreaker_Protect_normalCases(t *testing.T) {
	runErr := errors.New("runErr")
	fbErr := errors.New("fbErr")

	c := newMockClock()
	setClock(c)

	type args struct {
		run      RunFunc
		fallback FallbackFunc
	}
	tests := []struct {
		name                   string
		config                 *Config
		args                   args
		currState              State
		expectedPrevState      State
		expectedCurrState      State
		expectedProtectErr     error
		expectedRunErr         error
		expectedFbErr          error
		expectedWindowSnapshot WindowSnapshot
		expectedOutcome        Outcome
	}{
		{
			name:   "success",
			config: nil,
			args: args{
				run: func() error {
					return nil
				},
				fallback: nil,
			},
			currState:          Closed,
			expectedProtectErr: nil,
			expectedFbErr:      nil,
			expectedWindowSnapshot: WindowSnapshot{
				TotalCount:   1,
				SuccessCount: 1,
			},
			expectedOutcome: SuccessOutcome,
		},
		{
			name:   "runErr, nil fallback",
			config: nil,
			args: args{
				run: func() error {
					return runErr
				},
				fallback: nil,
			},
			currState:          Closed,
			expectedProtectErr: runErr,
			expectedRunErr:     runErr,
			expectedFbErr:      nil,
			expectedWindowSnapshot: WindowSnapshot{
				TotalCount:   1,
				ErrorCount:   1,
				ErrorPercent: 100,
			},
			expectedOutcome: ErrorOutcome,
		},
		{
			name:   "runErr, fallback return nil",
			config: nil,
			args: args{
				run: func() error {
					return runErr
				},
				fallback: nil,
			},
			currState:          Closed,
			expectedProtectErr: runErr,
			expectedRunErr:     runErr,
			expectedFbErr:      nil,
			expectedWindowSnapshot: WindowSnapshot{
				TotalCount:   1,
				ErrorCount:   1,
				ErrorPercent: 100,
			},
			expectedOutcome: ErrorOutcome,
		},
		{
			name:      "open state rejected",
			config:    nil,
			currState: Open,
			args: args{
				run: func() error {
					return runErr
				},
				fallback: nil,
			},
			expectedProtectErr: ErrCircuitOpenState,
			expectedRunErr:     ErrCircuitOpenState,
			expectedFbErr:      nil,
			expectedOutcome:    OpenStateRejectedOutcome,
		},
		{
			name:      "forced open state rejected",
			config:    nil,
			currState: ForcedOpen,
			args: args{
				run: func() error {
					return runErr
				},
				fallback: nil,
			},
			expectedProtectErr: ErrCircuitForcedOpenState,
			expectedRunErr:     ErrCircuitForcedOpenState,
			expectedFbErr:      nil,
			expectedOutcome:    ForcedOpenStateRejectedOutcome,
		},
		{
			name:      "forced closed success",
			config:    nil,
			currState: ForcedClosed,
			args: args{
				run: func() error {
					return nil
				},
				fallback: nil,
			},
			expectedWindowSnapshot: WindowSnapshot{
				TotalCount:   1,
				SuccessCount: 1,
			},
			expectedProtectErr: nil,
			expectedRunErr:     nil,
			expectedFbErr:      nil,
			expectedOutcome:    SuccessOutcome,
		},
		{
			name:      "forced closed error",
			config:    nil,
			currState: ForcedClosed,
			args: args{
				run: func() error {
					return runErr
				},
				fallback: nil,
			},
			expectedWindowSnapshot: WindowSnapshot{
				TotalCount:   1,
				ErrorCount:   1,
				ErrorPercent: 100,
			},
			expectedProtectErr: runErr,
			expectedRunErr:     runErr,
			expectedFbErr:      nil,
			expectedOutcome:    ErrorOutcome,
		},
		{
			name:      "run error, fallback error",
			config:    nil,
			currState: Closed,
			args: args{
				run: func() error {
					return runErr
				},
				fallback: func(err error) error {
					return fbErr
				},
			},
			expectedWindowSnapshot: WindowSnapshot{
				TotalCount:   1,
				ErrorCount:   1,
				ErrorPercent: 100,
			},
			expectedRunErr:     runErr,
			expectedProtectErr: fbErr,
			expectedFbErr:      fbErr,
			expectedOutcome:    ErrorOutcome,
		},
		{
			name:      "open state rejected, fallback error",
			config:    nil,
			currState: Open,
			args: args{
				run: func() error {
					return runErr
				},
				fallback: func(err error) error {
					return fbErr
				},
			},
			expectedRunErr:     ErrCircuitOpenState,
			expectedProtectErr: fbErr,
			expectedFbErr:      fbErr,
			expectedOutcome:    OpenStateRejectedOutcome,
		},
		{
			name:      "open state rejected, nil fallback error",
			config:    nil,
			currState: Open,
			args: args{
				run: func() error {
					return runErr
				},
				fallback: func(err error) error {
					return nil
				},
			},
			expectedRunErr:     ErrCircuitOpenState,
			expectedProtectErr: nil,
			expectedFbErr:      nil,
			expectedOutcome:    OpenStateRejectedOutcome,
		},
		{
			name: "closed success timeout",
			config: &Config{
				TimeoutThreshold: 1 * time.Millisecond,
			},
			args: args{
				run: func() error {
					c.advance(2 * time.Millisecond)
					return nil
				},
				fallback: nil,
			},
			currState:          Closed,
			expectedProtectErr: nil,
			expectedFbErr:      nil,
			expectedWindowSnapshot: WindowSnapshot{
				TotalCount:     1,
				SuccessCount:   1,
				TimeoutCount:   1,
				TimeoutPercent: 100,
			},
			expectedOutcome: SuccessTimeoutOutcome,
		},
		{
			name: "closed error timeout",
			config: &Config{
				TimeoutThreshold: 1 * time.Millisecond,
			},
			args: args{
				run: func() error {
					c.advance(2 * time.Millisecond)
					return runErr
				},
				fallback: nil,
			},
			currState:          Closed,
			expectedRunErr:     runErr,
			expectedProtectErr: runErr,
			expectedFbErr:      nil,
			expectedWindowSnapshot: WindowSnapshot{
				TotalCount:     1,
				ErrorCount:     1,
				ErrorPercent:   100,
				TimeoutCount:   1,
				TimeoutPercent: 100,
			},
			expectedOutcome: ErrorTimeoutOutcome,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, wrapper := range apiWrappers {
				t.Run(wrapper.name, func(t *testing.T) {
					cb := mustInitCB(t, tt.config)

					cb.StateTransition(context.Background(), cb.state, tt.currState, manualTransition)

					// Fallback error will always be nil when using enter/exit API
					if wrapper.name == enterExitAPIName {
						tt.expectedFbErr = nil
					}
					check := testEventListener(t, cb, &RequestStats{
						RequestOutcome: tt.expectedOutcome,
						RunError:       tt.expectedRunErr,
						FallbackError:  tt.expectedFbErr,
					}, nil)
					defer check()

					if err := wrapper.fn(cb, tt.args.run, tt.args.fallback); err != tt.expectedProtectErr {
						t.Errorf("Protect() error = %v, expectedProtectErr %v", err, tt.expectedProtectErr)
					}

					snapshot := cb.state.getWindowMetrics().getSnapshot()
					if !reflect.DeepEqual(snapshot, tt.expectedWindowSnapshot) {
						t.Errorf("Snapshot = %v, expected %v", snapshot, tt.expectedWindowSnapshot)
					}
				})
			}
		})
	}
}

func Test_circuitBreaker_ForceClose(t *testing.T) {
	cb := mustInitCB(t, nil)

	check := testEventListener(t, cb, nil, &StateTransitionEvent{
		PrevState:    Closed,
		CurrentState: ForcedClosed,
	})
	defer check()

	cb.ForceClose()

	if cb.state.getStateType() != ForcedClosed {
		t.Errorf("state = %v, expected %v", cb.state.getStateType().String(), ForcedClosed.String())
	}
}

func Test_circuitBreaker_ForceOpen(t *testing.T) {
	cb := mustInitCB(t, nil)

	check := testEventListener(t, cb, nil, &StateTransitionEvent{
		PrevState:    Closed,
		CurrentState: ForcedOpen,
	})
	defer check()

	cb.ForceOpen()

	if cb.state.getStateType() != ForcedOpen {
		t.Errorf("state = %v, expected %v", cb.state.getStateType().String(), ForcedOpen.String())
	}
}

func Test_circuitBreaker_EnterExit_shouldExitOnce(t *testing.T) {
	cb := mustInitCB(t, &Config{MaxConcurrentCalls: -1})
	entry, _ := cb.Enter()

	entry.Exit(nil)
	entry.Exit(nil)
	snapshot := cb.state.getWindowMetrics().getSnapshot()

	if snapshot.TotalCount > 1 {
		t.Errorf("snapshot total count = %v, expected <= %v", snapshot.TotalCount, 1)
	}
}

func TestCircuitBreaker_UpdateConfig(t *testing.T) {
	breakerName := "TestCircuitBreaker_UpdateConfig"

	// Create default breaker
	cb, err := Get(breakerName, nil)
	if err != nil {
		t.Fatalf("%v", err)
	}

	// Should be same as Defaults
	assertBreakerConfig(t, cb, NewDefaultConfig())

	newConfig := &Config{
		ErrorPercentThreshold:        14,
		TimeoutPercentThreshold:      78,
		TimeoutThreshold:             10 * time.Second,
		MinimumNumberOfCalls:         150,
		NumberOfCallsInSemiOpenState: 200,
		WaitDurationInOpenState:      10 * time.Second,
		MaxConcurrentCalls:           2000,
		SlidingWindowSize:            20,
	}
	err = UpdateConfig(breakerName, newConfig)
	if err != nil {
		t.Errorf("err = %v, wantErr nil", err)
	}

	// Updated configs
	assertBreakerConfig(t, cb, *newConfig)
}

// ================================ Race condition tests ================================

// To test race conditions in state transition from closed to open. Two error requests start in CLOSED state. Request 1 completes first
// and causes a state transition to OPEN state while Request 2 is still in-flight. When Request 2 completes, it should report to the
// CLOSED state instance but this outcome is essentially ignored since the cb is already using a new OPEN state instance.
// Request 2 should not cause another transition despite exceeding the thresholds.
func Test_circuitBreaker_Protect_closedToOpen(t *testing.T) {
	for _, wrapper := range apiWrappers {
		t.Run(wrapper.name, func(t *testing.T) {
			breaker := mustInitCB(t, &Config{
				ErrorPercentThreshold: 1,
				TimeoutThreshold:      1,
				MinimumNumberOfCalls:  1,
			})

			numOfStateTransitions := 0
			expectedNumOfStateTransitions := 1
			breaker.RegisterEventListener(func(event iEvent) {
				if _, ok := event.(StateTransitionEvent); ok {
					numOfStateTransitions++
				}
			})

			wg := &sync.WaitGroup{}

			ch, _, _ := runBlockingAndNonBlockingRequest(breaker, wrapper.fn, errors.New("err"), errors.New("err"))

			ch <- struct{}{}
			wg.Wait()

			if numOfStateTransitions != expectedNumOfStateTransitions {
				t.Errorf("numOfStateTransitions = %d, expected %d", numOfStateTransitions, expectedNumOfStateTransitions)
			}
		})
	}
}

// An error request is in-flight which should cause a state transition. ForceClose is done while the request is still in-flight.
// When the error request completes it should not cause a state transition.
func Test_circuitBreaker_race_forcedClosed(t *testing.T) {
	for _, wrapper := range apiWrappers {
		t.Run(wrapper.name, func(t *testing.T) {
			breaker := mustInitCB(t, &Config{
				ErrorPercentThreshold: 1,
				TimeoutThreshold:      1,
				MinimumNumberOfCalls:  1,
			})

			numOfStateTransitions := 0
			expectedNumOfStateTransitions := 1
			breaker.RegisterEventListener(func(event iEvent) {
				if _, ok := event.(StateTransitionEvent); ok {
					numOfStateTransitions++
				}
			})

			started := make(chan struct{})
			resume := make(chan struct{})
			done := make(chan struct{})
			go func() {
				_ = wrapper.fn(breaker, func() error {
					started <- struct{}{}
					<-resume // block until cb has been force closed
					return errors.New("error")
				}, nil)
				done <- struct{}{}
			}()

			<-started // block until the error request has started

			breaker.ForceClose() // ForceClose while runFunc is in-flight

			resume <- struct{}{}
			<-done

			if numOfStateTransitions != expectedNumOfStateTransitions {
				t.Errorf("numOfStateTransitions = %d, expected %d", numOfStateTransitions, expectedNumOfStateTransitions)
			}
			if breaker.state.getStateType() != ForcedClosed {
				t.Errorf("state = %v, expected %v", breaker.state.getStateType(), ForcedClosed)
			}
		})
	}
}

func Test_circuitBreaker_race_updateSetting(t *testing.T) {
	for _, wrapper := range apiWrappers {
		t.Run(wrapper.name, func(t *testing.T) {
			breaker := mustInitCB(t, nil)
			var wg sync.WaitGroup
			for i := 0; i < 256; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					_ = wrapper.fn(breaker, func() error {
						return nil
					}, nil)
					breaker.updateSetting(&Config{ErrorPercentThreshold: rand.Intn(99) + 1})
				}()
			}
			wg.Wait()
		})
	}
}

// ============================= util functions ============================================
func mustInitCB(tb testing.TB, config *Config) *CircuitBreaker {
	breaker, err := newCircuitBreaker("cb", config)
	if err != nil {
		tb.Fatal(err)
	}
	return breaker
}

func testEventListener(t *testing.T, breaker *CircuitBreaker, expectedRequestStats *RequestStats, expectedStateEvent *StateTransitionEvent) (checkListenerCalled func()) {
	isListenerCalled := false
	breaker.RegisterEventListener(func(event iEvent) {
		if expectedRequestStats != nil {
			if req, ok := event.(OnRequestCompleteEvent); ok {
				isListenerCalled = true
				req.Stats.Elapsed = 0 // Skip check for elapsed time
				if req.Stats != *expectedRequestStats {
					t.Errorf("actual %v, expected %v", req.Stats, expectedRequestStats)
				}
			}
		}

		if expectedStateEvent != nil {
			if ste, ok := event.(StateTransitionEvent); ok {
				isListenerCalled = true
				if ste.PrevState != expectedStateEvent.PrevState {
					t.Errorf("current state = %v, expected %v", ste.PrevState, expectedStateEvent.PrevState)
				}
				if ste.CurrentState != expectedStateEvent.CurrentState {
					t.Errorf("current state = %v, expected %v", ste.CurrentState, expectedStateEvent.CurrentState)
				}
			}
		}
	})

	return func() {
		if !isListenerCalled {
			t.Errorf("listener not called")
		}
	}
}

// runBlockingAndNonBlockingRequest runs two Protect() calls. First call will run and will be blocked.
// Second call will execute while the first is still running.
// resume channel resumes the first call, done channel unblocks after first call completes.
// err returned is from second call.
func runBlockingAndNonBlockingRequest(breaker *CircuitBreaker, cbAPI cbAPIWrapper, firstErr error, secondErr error) (
	resume chan<- struct{}, done <-chan struct{}, err error) {

	started := make(chan struct{})
	resumeChan := make(chan struct{})
	doneChan := make(chan struct{})

	go func() {
		_ = cbAPI(breaker, func() error {
			started <- struct{}{} // to ensure this goroutine acquires permission first
			<-resumeChan          // blocked
			return firstErr
		}, nil)
		doneChan <- struct{}{}
	}()

	<-started

	// this call will try to acquire permission but should be blocked
	err = cbAPI(breaker, func() error {
		return secondErr
	}, nil)

	return resumeChan, doneChan, err
}

const (
	protectAPIName   = "protect API"
	enterExitAPIName = "enter exit API"
)

var apiWrappers = []struct {
	name string
	fn   cbAPIWrapper
}{
	{
		name: protectAPIName,
		fn:   protectWrapper,
	},
	{
		name: enterExitAPIName,
		fn:   enterExitWrapper,
	},
}

func runProtectorAPIs(t *testing.T, f func(wrapper cbAPIWrapperWithContext)) {
	t.Run("protect API", func(t *testing.T) {
		f(protectWrapperWithContext)
	})
	t.Run("enter exit API", func(t *testing.T) {
		f(enterExitWrapperWithContext)
	})
}

// cbAPIWrapperWithContext is a generic function which wraps the two CircuitBreaker APIs.
// cbAPIWrapperWithContext is used so that we can use the two APIs interchangeably for the unit tests.
// The two APIs for CB are ProtectWithContext and EnterWithContext.
type cbAPIWrapperWithContext func(breaker *CircuitBreaker, runFunc RunFunc, opts ...ProtectEntryOption) error

func protectWrapperWithContext(cb *CircuitBreaker, runFunc RunFunc, opts ...ProtectEntryOption) error {
	pOpts := make([]ProtectOption, 0, len(opts))
	for _, opt := range opts {
		pOpts = append(pOpts, opt)
	}
	return cb.ProtectWithContext(context.Background(), runFunc, pOpts...)
}

func enterExitWrapperWithContext(cb *CircuitBreaker, runFunc RunFunc, opts ...ProtectEntryOption) error {
	eOpts := make([]EntryOption, 0, len(opts))
	for _, opt := range opts {
		eOpts = append(eOpts, opt)
	}

	entry, err := cb.EnterWithContext(context.Background(), eOpts...)
	if err != nil {
		return err
	}

	runErr := runFunc()
	fbErr := runErr

	entry.Exit(runErr)
	return fbErr
}

// cbAPIWrapper is a generic function which wraps the two CircuitBreaker APIs.
// cbAPIWrapper is used so that we can use the two APIs interchangeably for the unit tests.
// The two APIs for CB are Protect and Enter/Exit
type cbAPIWrapper func(breaker *CircuitBreaker, runFunc RunFunc, fallbackFunc FallbackFunc) error

func protectWrapper(cb *CircuitBreaker, runFunc RunFunc, fallbackFunc FallbackFunc) error {
	return cb.Protect(runFunc, fallbackFunc)
}

func enterExitWrapper(cb *CircuitBreaker, runFunc RunFunc, fallbackFunc FallbackFunc) error {
	entry, err := cb.Enter()
	if err != nil {
		if fallbackFunc != nil {
			return fallbackFunc(err)
		}
		return err
	}

	runErr := runFunc()
	fbErr := runErr

	if runErr != nil && fallbackFunc != nil {
		fbErr = fallbackFunc(runErr)
	}
	entry.Exit(runErr)
	return fbErr
}
