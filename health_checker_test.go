package circuitbreaker

import (
	"testing"
)

func Test_healthChecker_isThresholdExceed(t *testing.T) {
	tests := []struct {
		name string

		hc       healthChecker
		snapshot iSnapshot
		setting  iSetting

		resultWant           bool
		transitionReasonWant transitionReason
	}{
		{
			name:                 "test timeout checker exceeded",
			hc:                   timeoutPercentChecker,
			snapshot:             &dummySnapshot{timeoutPercent: 10},
			setting:              &dummySetting{timeoutPercentThreshold: 1},
			resultWant:           true,
			transitionReasonWant: timeoutThresholdExceeded,
		},
		{
			name:                 "test error checker exceeded",
			hc:                   errorPercentChecker,
			snapshot:             &dummySnapshot{errorPercent: 10},
			setting:              &dummySetting{errorPercentThreshold: 1},
			resultWant:           true,
			transitionReasonWant: errorThresholdExceeded,
		},
		{
			name:                 "test unknown checker",
			hc:                   3,
			snapshot:             &dummySnapshot{errorPercent: 10},
			setting:              &dummySetting{errorPercentThreshold: 1},
			resultWant:           false,
			transitionReasonWant: invalidTransition,
		},
		{
			name:                 "test timeout checker exceeded",
			hc:                   timeoutPercentChecker,
			snapshot:             &dummySnapshot{timeoutPercent: 1},
			setting:              &dummySetting{timeoutPercentThreshold: 10},
			resultWant:           false,
			transitionReasonWant: invalidTransition,
		},
		{
			name:                 "test error checker exceeded",
			hc:                   errorPercentChecker,
			snapshot:             &dummySnapshot{errorPercent: 1},
			setting:              &dummySetting{errorPercentThreshold: 10},
			resultWant:           false,
			transitionReasonWant: invalidTransition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultGot, transitionReasonGot := tt.hc.isThresholdExceed(tt.snapshot, tt.setting)

			if resultGot != tt.resultWant {
				t.Errorf("result got = %v, reesult want = %v", resultGot, tt.resultWant)
			}
			if transitionReasonGot != tt.transitionReasonWant {
				t.Errorf("transaction reason got = %v, transaction reason want = %v", transitionReasonGot, tt.transitionReasonWant)
			}
		})
	}
}

type dummySnapshot struct {
	timeoutPercent int
	errorPercent   int
}

func (d *dummySnapshot) GetTotalCount() int64 {
	//TODO implement me
	panic("implement me")
}

func (d *dummySnapshot) GetTimeoutPercent() int { return d.timeoutPercent }
func (d *dummySnapshot) GetErrorPercent() int   { return d.errorPercent }

type dummySetting struct {
	timeoutPercentThreshold int
	errorPercentThreshold   int
}

func (d *dummySetting) GetTimeoutPercentThreshold() int { return d.timeoutPercentThreshold }
func (d *dummySetting) GetErrorPercentThreshold() int   { return d.errorPercentThreshold }
