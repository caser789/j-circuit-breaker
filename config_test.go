package circuitbreaker

import (
	"testing"
	"time"
)

//nolint:funlen
func TestIsValidConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid",
			config: &Config{
				ErrorPercentThreshold:        50,
				TimeoutPercentThreshold:      50,
				TimeoutThreshold:             10 * time.Second,
				MinimumNumberOfCalls:         10,
				NumberOfCallsInSemiOpenState: 3,
				WaitDurationInOpenState:      10 * time.Second,
				MaxConcurrentCalls:           100,
				SlidingWindowSize:            10,
			},
			wantErr: false,
		},
		{
			name:    "nil setting",
			config:  nil,
			wantErr: true,
		},
		{
			name: "disable error percent",
			config: &Config{
				ErrorPercentThreshold: -1,
			},
			wantErr: false,
		},
		{
			name: "invalid error percent",
			config: &Config{
				ErrorPercentThreshold: 101,
			},
			wantErr: true,
		},
		{
			name: "disable timeout percent",
			config: &Config{
				TimeoutPercentThreshold: -1,
			},
			wantErr: false,
		},
		{
			name: "invalid timeout percent",
			config: &Config{
				TimeoutPercentThreshold: 101,
			},
			wantErr: true,
		},
		{
			name: "disable both error and timeout percent threshold",
			config: &Config{
				ErrorPercentThreshold:   -1,
				TimeoutPercentThreshold: -1,
			},
			wantErr: false,
		},
		{
			name: "multi error",
			config: &Config{
				ErrorPercentThreshold:   101,
				TimeoutPercentThreshold: 101,
			},
			wantErr: true,
		},
		{
			name: "invalid SlidingWindowSize",
			config: &Config{
				SlidingWindowSize: -1,
			},
			wantErr: true,
		},
		{
			name: "invalid TimeoutThreshold",
			config: &Config{
				TimeoutThreshold: -1,
			},
			wantErr: true,
		},
		{
			name: "invalid MinimumNumberOfCalls",
			config: &Config{
				MinimumNumberOfCalls: -1,
			},
			wantErr: true,
		},
		{
			name: "invalid WaitDurationInOpenState",
			config: &Config{
				WaitDurationInOpenState: -1,
			},
			wantErr: true,
		},
		{
			name: "invalid NumberOfCallsInSemiOpenState",
			config: &Config{
				NumberOfCallsInSemiOpenState: -1,
			},
			wantErr: true,
		},
		{
			name: "valid disabled MaxConcurrentCalls",
			config: &Config{
				MaxConcurrentCalls: -1,
			},
			wantErr: false,
		},
		{
			name: "valid MaxConcurrentCalls",
			config: &Config{
				MaxConcurrentCalls: -100,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := IsValidConfig(tt.config); (err != nil) != tt.wantErr {
				t.Errorf("IsValidConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
