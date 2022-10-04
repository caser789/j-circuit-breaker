package circuitbreaker

import (
	"reflect"
	"testing"
	"time"
)

//nolint:funlen
func Test_state_onError(t *testing.T) {
	newSetting := fillDefaultsAndGetSetting(Config{})

	type want struct {
		snapshot WindowSnapshot
	}
	type args struct {
		isTimeout bool
	}
	tests := []struct {
		name  string
		want  want
		state iState
		args  args
	}{
		{
			name: "closed error no timeout",
			args: args{
				isTimeout: false,
			},
			want: want{
				snapshot: WindowSnapshot{
					TotalCount:   1,
					ErrorCount:   1,
					ErrorPercent: 100,
				},
			},
			state: newClosedState(newSetting),
		},
		{
			name: "closed error timeout",
			args: args{
				isTimeout: true,
			},
			want: want{
				snapshot: WindowSnapshot{
					TotalCount:     1,
					ErrorCount:     1,
					ErrorPercent:   100,
					TimeoutCount:   1,
					TimeoutPercent: 100,
				},
			},
			state: newClosedState(newSetting),
		},
		{
			name: "open error no timeout",
			args: args{
				isTimeout: false,
			},
			want: want{
				snapshot: WindowSnapshot{},
			},
			state: newOpenState(newSetting),
		},
		{
			name: "open error timeout",
			args: args{
				isTimeout: true,
			},
			want: want{
				snapshot: WindowSnapshot{},
			},
			state: newOpenState(newSetting),
		},
		{
			name: "semi open error no timeout",
			args: args{
				isTimeout: false,
			},
			want: want{
				snapshot: WindowSnapshot{
					TotalCount:   1,
					ErrorCount:   1,
					ErrorPercent: 100,
				},
			},
			state: newSemiOpenState(newSetting),
		},
		{
			name: "semi open error timeout",
			args: args{
				isTimeout: true,
			},
			want: want{
				snapshot: WindowSnapshot{
					TotalCount:     1,
					ErrorCount:     1,
					ErrorPercent:   100,
					TimeoutCount:   1,
					TimeoutPercent: 100,
				},
			},
			state: newSemiOpenState(newSetting),
		},
		{
			name: "forced open error no timeout",
			args: args{
				isTimeout: false,
			},
			want: want{
				snapshot: WindowSnapshot{},
			},
			state: newForcedOpenState(newSetting),
		},
		{
			name: "forced open error timeout",
			args: args{
				isTimeout: true,
			},
			want: want{
				snapshot: WindowSnapshot{},
			},
			state: newForcedOpenState(newSetting),
		},
		{
			name: "forced closed error no timeout",
			args: args{
				isTimeout: false,
			},
			want: want{
				snapshot: WindowSnapshot{
					TotalCount:   1,
					ErrorCount:   1,
					ErrorPercent: 100,
				},
			},
			state: newForcedClosedState(newSetting),
		},
		{
			name: "forced closed error timeout",
			args: args{
				isTimeout: true,
			},
			want: want{
				snapshot: WindowSnapshot{
					TotalCount:     1,
					ErrorCount:     1,
					ErrorPercent:   100,
					TimeoutCount:   1,
					TimeoutPercent: 100,
				},
			},
			state: newForcedClosedState(newSetting),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := tt.state.onError(tt.args.isTimeout)
			if !reflect.DeepEqual(snapshot, tt.want.snapshot) {
				t.Errorf("snapshot = %v, expected %v", snapshot, tt.want.snapshot)
			}
		})
	}
}

//nolint:funlen
func Test_state_onSuccess(t *testing.T) {
	newSetting := fillDefaultsAndGetSetting(Config{})

	type want struct {
		snapshot WindowSnapshot
	}
	type args struct {
		isTimeout bool
	}
	tests := []struct {
		name  string
		want  want
		state iState
		args  args
	}{
		{
			name: "closed error no timeout",
			args: args{
				isTimeout: false,
			},
			want: want{
				snapshot: WindowSnapshot{
					TotalCount:   1,
					SuccessCount: 1,
				},
			},
			state: newClosedState(newSetting),
		},
		{
			name: "closed error timeout",
			args: args{
				isTimeout: true,
			},
			want: want{
				snapshot: WindowSnapshot{
					TotalCount:     1,
					SuccessCount:   1,
					TimeoutCount:   1,
					TimeoutPercent: 100,
				},
			},
			state: newClosedState(newSetting),
		},
		{
			name: "open error no timeout",
			args: args{
				isTimeout: false,
			},
			want: want{
				snapshot: WindowSnapshot{},
			},
			state: newOpenState(newSetting),
		},
		{
			name: "open error timeout",
			args: args{
				isTimeout: true,
			},
			want: want{
				snapshot: WindowSnapshot{},
			},
			state: newOpenState(newSetting),
		},
		{
			name: "semi open error no timeout",
			args: args{
				isTimeout: false,
			},
			want: want{
				snapshot: WindowSnapshot{
					TotalCount:   1,
					SuccessCount: 1,
				},
			},
			state: newSemiOpenState(newSetting),
		},
		{
			name: "semi open error timeout",
			args: args{
				isTimeout: true,
			},
			want: want{
				snapshot: WindowSnapshot{
					TotalCount:     1,
					SuccessCount:   1,
					TimeoutCount:   1,
					TimeoutPercent: 100,
				},
			},
			state: newSemiOpenState(newSetting),
		},
		{
			name: "forced open error no timeout",
			args: args{
				isTimeout: false,
			},
			want: want{
				snapshot: WindowSnapshot{},
			},
			state: newForcedOpenState(newSetting),
		},
		{
			name: "forced open error timeout",
			args: args{
				isTimeout: true,
			},
			want: want{
				snapshot: WindowSnapshot{},
			},
			state: newForcedOpenState(newSetting),
		},
		{
			name: "forced closed error no timeout",
			args: args{
				isTimeout: false,
			},
			want: want{
				snapshot: WindowSnapshot{
					TotalCount:   1,
					SuccessCount: 1,
				},
			},
			state: newForcedClosedState(newSetting),
		},
		{
			name: "forced closed error timeout",
			args: args{
				isTimeout: true,
			},
			want: want{
				snapshot: WindowSnapshot{
					TotalCount:     1,
					SuccessCount:   1,
					TimeoutCount:   1,
					TimeoutPercent: 100,
				},
			},
			state: newForcedClosedState(newSetting),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := tt.state.onSuccess(tt.args.isTimeout)
			if !reflect.DeepEqual(snapshot, tt.want.snapshot) {
				t.Errorf("snapshot = %v, expected %v", snapshot, tt.want.snapshot)
			}
		})
	}
}

//nolint:funlen
func Test_state_shouldStateTransition(t *testing.T) {
	newSetting := fillDefaultsAndGetSetting(Config{
		ErrorPercentThreshold:        50,
		TimeoutPercentThreshold:      50,
		TimeoutThreshold:             1 * time.Millisecond,
		MinimumNumberOfCalls:         1,
		NumberOfCallsInSemiOpenState: 1,
		WaitDurationInOpenState:      1 * time.Millisecond,
		MaxConcurrentCalls:           5,
		SlidingWindowSize:            10,
	})

	type want struct {
		shouldStateTransition bool
		toState               State
		reason                transitionReason
	}
	type args struct {
		snapshot       WindowSnapshot
		transitionType transitionType
	}
	tests := []struct {
		name  string
		want  want
		state iState
		args  args
	}{
		{
			name: "closed to open, errors exceed threshold",
			args: args{
				snapshot: WindowSnapshot{
					TotalCount:   2,
					ErrorCount:   2,
					ErrorPercent: 100,
				},
				transitionType: onRequestEnd,
			},
			want: want{
				shouldStateTransition: true,
				toState:               Open,
				reason:                errorThresholdExceeded,
			},
			state: newClosedState(newSetting),
		},
		{
			name: "closed to open, timeouts exceed threshold",
			args: args{
				snapshot: WindowSnapshot{
					TotalCount:     2,
					TimeoutCount:   2,
					TimeoutPercent: 100,
				},
				transitionType: onRequestEnd,
			},
			want: want{
				shouldStateTransition: true,
				toState:               Open,
				reason:                timeoutThresholdExceeded,
			},
			state: newClosedState(newSetting),
		},
		{
			name: "closed below threshold",
			args: args{
				snapshot: WindowSnapshot{
					TotalCount:   2,
					SuccessCount: 2,
				},
				transitionType: onRequestEnd,
			},
			want: want{
				shouldStateTransition: false,
				toState:               Closed,
				reason:                invalidTransition,
			},
			state: newClosedState(newSetting),
		},
		{
			name: "semi-open to open, errors exceed threshold",
			args: args{
				snapshot: WindowSnapshot{
					TotalCount:   2,
					ErrorCount:   2,
					ErrorPercent: 100,
				},
				transitionType: onRequestEnd,
			},
			want: want{
				shouldStateTransition: true,
				toState:               Open,
				reason:                errorThresholdExceeded,
			},
			state: newSemiOpenState(newSetting),
		},
		{
			name: "semi-open to open, timeouts exceed thresholds",
			args: args{
				snapshot: WindowSnapshot{
					TotalCount:     2,
					TimeoutCount:   2,
					TimeoutPercent: 100,
				},
				transitionType: onRequestEnd,
			},
			want: want{
				shouldStateTransition: true,
				toState:               Open,
				reason:                timeoutThresholdExceeded,
			},
			state: newSemiOpenState(newSetting),
		},
		{
			name: "semi-open to closed",
			args: args{
				snapshot: WindowSnapshot{
					TotalCount:   2,
					SuccessCount: 2,
				},
				transitionType: onRequestEnd,
			},
			want: want{
				shouldStateTransition: true,
				toState:               Closed,
				reason:                failurePercentsBelowThresholds,
			},
			state: newSemiOpenState(newSetting),
		},
		{
			name: "forcedClosed",
			args: args{
				snapshot: WindowSnapshot{
					TotalCount:   2,
					SuccessCount: 2,
				},
				transitionType: onRequestEnd,
			},
			want: want{
				shouldStateTransition: false,
				toState:               ForcedClosed,
				reason:                invalidTransition,
			},
			state: newForcedClosedState(newSetting),
		},
		{
			name: "forcedOpen",
			args: args{
				snapshot: WindowSnapshot{
					TotalCount:   2,
					SuccessCount: 2,
				},
				transitionType: onRequestEnd,
			},
			want: want{
				shouldStateTransition: false,
				toState:               ForcedOpen,
				reason:                invalidTransition,
			},
			state: newForcedOpenState(newSetting),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toState, reason, shouldTransition := tt.state.shouldStateTransition(tt.args.snapshot, tt.args.transitionType)
			if shouldTransition != tt.want.shouldStateTransition {
				t.Errorf("shouldStateTransition = %v, expected %v", shouldTransition, tt.want.shouldStateTransition)
			}
			if toState != tt.want.toState {
				t.Errorf("toState = %v, expected %v", toState, tt.want.toState)
			}
			if reason != tt.want.reason {
				t.Errorf("reason = %v, expected %v", reason, tt.want.reason)
			}
		})
	}
}

// openState shouldStateTransition method testing is separate as these are time sensitive
func Test_openState_shouldStateTransition(t *testing.T) {
	c := newMockClock()
	setClock(c)

	newSetting := fillDefaultsAndGetSetting(Config{
		WaitDurationInOpenState: 1 * time.Millisecond,
	})
	openState := newOpenState(newSetting)

	toState, _, shouldTransition := openState.shouldStateTransition(WindowSnapshot{}, onRequestStart)

	if shouldTransition != false {
		t.Errorf("shouldStateTransition = %v, expected %v", shouldTransition, false)
	}
	if toState != Open {
		t.Errorf("toState = %v, expected %v", toState, Open)
	}

	// Advance clock after wait time
	c.advance(2 * time.Millisecond)
	toState, _, shouldTransition = openState.shouldStateTransition(WindowSnapshot{}, onRequestStart)

	if shouldTransition != true {
		t.Errorf("shouldStateTransition = %v, expected %v", shouldTransition, true)
	}
	if toState != SemiOpen {
		t.Errorf("toState = %v, expected %v", toState, SemiOpen)
	}
}

// There are no tests for SemiOpen here because the methods are stateful for SemiOpen.
// SemiOpen is tested through the Protect() tests.
//nolint:funlen
func Test_state_otherMethods(t *testing.T) {
	newSetting := fillDefaultsAndGetSetting(Config{})
	type want struct {
		errPermission                error
		stateType                    State
		shouldCheckForMaxConcurrency bool
	}
	tests := []struct {
		name  string
		want  want
		state iState
	}{
		{
			name: "closed",
			want: want{
				errPermission:                nil,
				stateType:                    Closed,
				shouldCheckForMaxConcurrency: true,
			},
			state: newClosedState(newSetting),
		},
		{
			name: "open",
			want: want{
				errPermission:                ErrCircuitOpenState,
				stateType:                    Open,
				shouldCheckForMaxConcurrency: true,
			},
			state: newOpenState(newSetting),
		},
		{
			name: "force closed",
			want: want{
				errPermission:                nil,
				stateType:                    ForcedClosed,
				shouldCheckForMaxConcurrency: false,
			},
			state: newForcedClosedState(newSetting),
		},
		{
			name: "force open",
			want: want{
				errPermission:                ErrCircuitForcedOpenState,
				stateType:                    ForcedOpen,
				shouldCheckForMaxConcurrency: false,
			},
			state: newForcedOpenState(newSetting),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.state
			if got := s.getStateType(); got != tt.want.stateType {
				t.Errorf("getStateType() = %v, want %v", got, tt.want)
			}

			if err := s.acquirePermission(); err != tt.want.errPermission {
				t.Errorf("err = %v, expected nil", tt.want.errPermission)
			}

			if shouldCheckForMaxConcurrency := s.shouldCheckForMaxConcurrency(); shouldCheckForMaxConcurrency != tt.want.shouldCheckForMaxConcurrency {
				t.Errorf("shouldCheckForMaxConcurrency = %v, expected %v",
					shouldCheckForMaxConcurrency, tt.want.shouldCheckForMaxConcurrency)
			}

			s.releasePermission() // no op
		})
	}
}
