package circuitbreaker

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"
)

//nolint:funlen
func TestGet_newBreaker(t *testing.T) {
	resetBreakers()

	type args struct {
		name   string
		config *Config
	}
	type want struct {
		config          Config
		err             bool
		nilBreaker      bool
		disabledLimiter bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "nil config, gives default",
			args: args{
				name:   "cb",
				config: nil,
			},
			want: want{
				config: NewDefaultConfig(),
				err:    false,
			},
		},
		{
			name: "empty name",
			args: args{
				name:   "",
				config: nil,
			},
			want: want{
				config:     NewDefaultConfig(),
				err:        false,
				nilBreaker: false,
			},
		},
		{
			name: "valid name, valid config",
			args: args{
				name: "cb",
				config: &Config{
					ErrorPercentThreshold:        30,
					TimeoutPercentThreshold:      30,
					TimeoutThreshold:             1 * time.Second,
					MinimumNumberOfCalls:         1,
					NumberOfCallsInSemiOpenState: 3,
					WaitDurationInOpenState:      1 * time.Second,
					MaxConcurrentCalls:           123,
					SlidingWindowSize:            123,
				},
			},
			want: want{
				config: Config{
					ErrorPercentThreshold:        30,
					TimeoutPercentThreshold:      30,
					TimeoutThreshold:             1 * time.Second,
					MinimumNumberOfCalls:         1,
					NumberOfCallsInSemiOpenState: 3,
					WaitDurationInOpenState:      1 * time.Second,
					MaxConcurrentCalls:           123,
					SlidingWindowSize:            123,
				},
			},
		},
		{
			name: "valid name, valid config with missing values",
			args: args{
				name: "cb",
				config: &Config{
					ErrorPercentThreshold:        30,
					TimeoutPercentThreshold:      30,
					TimeoutThreshold:             1 * time.Second,
					MinimumNumberOfCalls:         1,
					NumberOfCallsInSemiOpenState: 3,
					WaitDurationInOpenState:      1 * time.Second,
				},
			},
			want: want{
				config: Config{
					ErrorPercentThreshold:        30,
					TimeoutPercentThreshold:      30,
					TimeoutThreshold:             1 * time.Second,
					MinimumNumberOfCalls:         1,
					NumberOfCallsInSemiOpenState: 3,
					WaitDurationInOpenState:      1 * time.Second,
					MaxConcurrentCalls:           NewDefaultConfig().MaxConcurrentCalls,
					SlidingWindowSize:            NewDefaultConfig().SlidingWindowSize,
				},
			},
		},
		{
			name: "valid breaker with disabled concurrency limiter",
			args: args{
				name: "cb",
				config: &Config{
					MaxConcurrentCalls: -1,
				},
			},
			want: want{
				config: Config{
					ErrorPercentThreshold:        50,
					TimeoutPercentThreshold:      0,
					TimeoutThreshold:             0,
					MinimumNumberOfCalls:         100,
					NumberOfCallsInSemiOpenState: 100,
					WaitDurationInOpenState:      10 * time.Second,
					MaxConcurrentCalls:           -1,
					SlidingWindowSize:            10,
					UseCachedTime:                false,
				},
				disabledLimiter: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetBreakers()
			breaker, err := Get(tt.args.name, tt.args.config)
			if (err != nil) != tt.want.err {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.want.err)
				return
			}
			if (breaker == nil) != tt.want.nilBreaker {
				t.Errorf("got = %v, wantNilBreaker %v", breaker, tt.want.nilBreaker)
			}
			if breaker != nil {
				if breaker.setting.Config != tt.want.config {
					t.Errorf("config = %v, want %v", breaker.setting.Config, tt.want.config)
				}

				if tt.want.disabledLimiter {
					_, ok := breaker.concurrencyLimiter.(*noOpLimiter)
					if !ok {
						t.Errorf("limiter = %v, expected = %v", breaker.concurrencyLimiter, &noOpLimiter{})
					}

				} else {
					limiter, _ := breaker.concurrencyLimiter.(*weightedSemaphore)
					if limiter.size != int64(tt.want.config.MaxConcurrentCalls) {
						t.Errorf("pool size = %v, want %v", limiter.size, tt.want.config.MaxConcurrentCalls)
					}
				}
			}
		})
	}
}

// An empty breaker name will always return a new breaker and
// will not be saved in the global registry.
func TestGet_emptyName(t *testing.T) {
	resetBreakers()

	breaker1, err := Get("  ", nil)
	if err != nil {
		t.Fatal(err)
	}

	breaker2, err := Get("  ", nil)
	if err != nil {
		t.Fatal(err)
	}

	if breaker1 == breaker2 {
		t.Errorf("expected different breaker instances")
	}
}

func TestGet_existingBreaker_nilConfig(t *testing.T) {
	resetBreakers()

	cbName := "cb"
	breaker1, err := Get(cbName, nil)
	if err != nil {
		t.Fatal()
	}
	breaker2, err := Get(cbName, nil)
	if !reflect.DeepEqual(breaker1, breaker2) {
		t.Errorf("breaker1 %v, breaker2 %v", breaker1, breaker2)
	}
	if err != nil {
		t.Fatal()
	}
}

func TestGet_existingBreaker_shouldIgnoreNewConfig(t *testing.T) {
	resetBreakers()

	cbName := "cb"

	breaker1, err := Get(cbName, nil)
	if err != nil {
		t.Fatal()
	}

	customConfig := Config{
		ErrorPercentThreshold:        50,
		TimeoutPercentThreshold:      50,
		TimeoutThreshold:             10 * time.Second,
		MinimumNumberOfCalls:         150,
		NumberOfCallsInSemiOpenState: 200,
		WaitDurationInOpenState:      10 * time.Second,
		MaxConcurrentCalls:           2000,
		SlidingWindowSize:            20,
	}

	// Get with custom config should ignore custom config
	breaker2, err := Get(cbName, &customConfig)

	if !reflect.DeepEqual(breaker1, breaker2) {
		t.Errorf("breaker1 %v, breaker2 %v", breaker1, breaker2)
	}
	if err != nil {
		t.Fatal()
	}

	assertBreakerConfig(t, breaker2, NewDefaultConfig())
}

func TestGet_invalidConfig(t *testing.T) {
	resetBreakers()

	invalidConfig := Config{
		ErrorPercentThreshold: 101,
	}
	cbName := "cb"
	_, err := Get(cbName, &invalidConfig)

	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

func TestGetExisting(t *testing.T) {
	resetBreakers()

	cbName := "cb"
	breaker := GetExisting(cbName)
	if breaker != nil {
		t.Errorf("expected nil, got non-nil")
	}

	breaker1, err := Get(cbName, nil)
	if err != nil {
		t.Fatal(err)
	}
	breaker2 := GetExisting(cbName)
	if !reflect.DeepEqual(breaker1, breaker2) {
		t.Errorf("breaker1 %v, breaker2 %v", breaker1, breaker2)
	}
}

func TestForceOpen(t *testing.T) {
	resetBreakers()

	cbName := "cb"
	err := ForceOpen(cbName)
	if err != ErrBreakerNotFound {
		t.Errorf("expected ErrBreakerNotFound, get err: %v", err)
	}

	breaker, err := Get(cbName, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = ForceOpen(cbName)
	if err != nil {
		t.Fatal(err)
	}

	if breaker.state.getStateType() != ForcedOpen {
		t.Errorf("state = %v, expected %v", breaker.state.getStateType().String(), ForcedOpen.String())
	}
}

func TestForceClose(t *testing.T) {
	resetBreakers()

	cbName := "cb"
	err := ForceOpen(cbName)
	if err != ErrBreakerNotFound {
		t.Errorf("expected ErrBreakerNotFound, get err: %v", err)
	}

	breaker, err := Get(cbName, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = ForceClose(cbName)
	if err != nil {
		t.Fatal(err)
	}

	if breaker.state.getStateType() != ForcedClosed {
		t.Errorf("state = %v, expected %v", breaker.state.getStateType().String(), ForcedOpen.String())
	}
}

func TestReset(t *testing.T) {
	resetBreakers()

	cbName := "cb"
	err := ForceOpen(cbName)
	if err != ErrBreakerNotFound {
		t.Errorf("expected ErrBreakerNotFound, get err: %v", err)
	}

	breaker, err := Get(cbName, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = Reset(cbName)
	if err != nil {
		t.Fatal(err)
	}

	snapshot := getStateSnapshot(breaker.state, breaker.concurrencyLimiter)
	if snapshot.GetTotalCount() != 0 {
		t.Errorf("window should be reset")
	}
	currState := breaker.state.getStateType()
	if currState != Closed {
		t.Errorf("Reset() to closed actual %v, want = %v", currState, Closed)
	}
}

//nolint:funlen
func TestUpdateConfig_existingBreaker(t *testing.T) {
	type args struct {
		config Config
	}
	type want struct {
		wantErr bool
		config  Config
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "existing breaker, valid config",
			args: args{
				config: Config{
					ErrorPercentThreshold:        14,
					TimeoutPercentThreshold:      78,
					TimeoutThreshold:             10 * time.Second,
					MinimumNumberOfCalls:         150,
					NumberOfCallsInSemiOpenState: 200,
					WaitDurationInOpenState:      10 * time.Second,
					MaxConcurrentCalls:           2000,
					SlidingWindowSize:            20,
				},
			},
			want: want{
				wantErr: false,
				config: Config{
					ErrorPercentThreshold:        14,
					TimeoutPercentThreshold:      78,
					TimeoutThreshold:             10 * time.Second,
					MinimumNumberOfCalls:         150,
					NumberOfCallsInSemiOpenState: 200,
					WaitDurationInOpenState:      10 * time.Second,
					MaxConcurrentCalls:           2000,
					SlidingWindowSize:            20,
				},
			},
		},
		{
			name: "existing breaker, invalid config",
			args: args{
				config: Config{
					ErrorPercentThreshold: 101,
				},
			},
			want: want{
				wantErr: true,
				config:  NewDefaultConfig(),
			},
		},
		{
			name: "existing breaker, disable max concurrent calls",
			args: args{
				config: Config{
					MaxConcurrentCalls: -1,
				},
			},
			want: want{
				wantErr: false,
				config: Config{
					ErrorPercentThreshold:        50,
					TimeoutPercentThreshold:      0,
					TimeoutThreshold:             0,
					MinimumNumberOfCalls:         100,
					NumberOfCallsInSemiOpenState: 100,
					WaitDurationInOpenState:      10 * time.Second,
					MaxConcurrentCalls:           -1,
					SlidingWindowSize:            10,
					UseCachedTime:                false,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetBreakers()

			breakerName := "existingBreakerUpdateConfig"
			// Create breaker
			cb, err := Get(breakerName, nil)
			if err != nil {
				t.Fatalf("%v", err)
			}

			// Should be same as Defaults
			assertBreakerConfig(t, cb, NewDefaultConfig())

			err = UpdateConfig(breakerName, &tt.args.config)
			if (err != nil) != tt.want.wantErr {
				t.Errorf("err = %v, wantErr %v ", err, tt.want.wantErr)
			}

			// Updated configs
			assertBreakerConfig(t, cb, tt.want.config)
		})
	}
}

func TestUpdateConfig_nonExistentBreaker(t *testing.T) {
	resetBreakers()

	type args struct {
		name   string
		config Config
	}
	type want struct {
		wantErr bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "nonexistent breaker",
			args: args{
				name: "nonexistentBreaker",
				config: Config{
					ErrorPercentThreshold:   13,
					TimeoutPercentThreshold: 15,
				},
			},
			want: want{
				wantErr: true,
			},
		},
		{
			name: "empty name",
			args: args{
				name: "",
				config: Config{
					ErrorPercentThreshold:   13,
					TimeoutPercentThreshold: 15,
				},
			},
			want: want{
				wantErr: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UpdateConfig(tt.args.name, &tt.args.config)
			if (err != nil) != tt.want.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.want.wantErr)
			}
		})
	}
}

func TestUpdateConfig_existingBreaker_sameConfig(t *testing.T) {
	resetBreakers()

	customConfig := Config{
		ErrorPercentThreshold:        50,
		TimeoutPercentThreshold:      50,
		TimeoutThreshold:             10 * time.Second,
		MinimumNumberOfCalls:         150,
		NumberOfCallsInSemiOpenState: 200,
		WaitDurationInOpenState:      10 * time.Second,
		MaxConcurrentCalls:           2000,
		SlidingWindowSize:            20,
	}
	cbName := "cb"

	cb, err := Get(cbName, &customConfig)
	if err != nil {
		t.Fatalf("%v", err)
	}

	cb.state.onSuccess(false)

	// Update config will not reset breaker
	err = UpdateConfig(cbName, &customConfig)
	if err != nil {
		t.Fatalf("%v", err)
	}

	snapshot := cb.state.getWindowMetrics().getSnapshot()
	if snapshot.TotalCount != 1 {
		t.Errorf("expected window to be not reset")
	}

	// Updated configs
	assertBreakerConfig(t, cb, customConfig)
}

func TestUpdateConfigs(t *testing.T) {
	resetBreakers()

	customConfig := Config{
		ErrorPercentThreshold:        50,
		TimeoutPercentThreshold:      50,
		TimeoutThreshold:             10 * time.Second,
		MinimumNumberOfCalls:         150,
		NumberOfCallsInSemiOpenState: 200,
		WaitDurationInOpenState:      10 * time.Second,
		MaxConcurrentCalls:           2000,
		SlidingWindowSize:            20,
	}

	existingCb := "cb"

	cb, err := Get(existingCb, nil)
	if err != nil {
		t.Fatalf("%v", err)
	}

	configMap := make(map[string]*Config)
	configMap[existingCb] = &customConfig

	err = UpdateConfigs(configMap)
	if err != nil {
		t.Fatalf("%v", err)
	}

	// Check updated configs
	assertBreakerConfig(t, cb, customConfig)
}

func TestUpdateConfigs_invalidConfig(t *testing.T) {
	resetBreakers()

	validConfig := Config{
		ErrorPercentThreshold: 13,
	}

	invalidConfig := Config{
		ErrorPercentThreshold: 101,
	}

	existingCb := "cb"
	existingCb2 := "cb"

	cb1, err := Get(existingCb, nil)
	if err != nil {
		t.Fatalf("%v", err)
	}
	_, err2 := Get(existingCb2, nil)
	if err2 != nil {
		t.Fatalf("%v", err2)
	}

	configMap := make(map[string]*Config)
	configMap[existingCb] = &validConfig
	configMap[existingCb2] = &invalidConfig

	err = UpdateConfigs(configMap)
	if err == nil {
		t.Errorf("got nil error, expected invalid config")
	}

	// UpdateConfigs should be atomic and first breaker with valid config should not have been updated
	assertBreakerConfig(t, cb1, NewDefaultConfig())
}

func assertBreakerConfig(t *testing.T, breaker *CircuitBreaker, expectedConfig Config) {
	if !reflect.DeepEqual(expectedConfig, breaker.setting.Config) {
		t.Errorf("config  = %v , want %v", breaker.setting.Config, expectedConfig)
	}

}
func BenchmarkProtectParallel(b *testing.B) {
	cb := mustInitCB(b, nil)
	b.SetParallelism(16)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			protectWrapper(cb, func() error {
				return nil
			}, func(err error) error {
				return nil
			})
		}
	})
}

func BenchmarkProtect_Closed(b *testing.B) {
	b.Run("Protect in Closed state,without time caching", func(b *testing.B) {
		for _, wrapper := range apiWrappers {
			b.Run(wrapper.name, func(b *testing.B) {
				cb := mustInitCB(b, nil)
				benchmarkProtect(b, cb, wrapper.fn)
			})
		}
	})

	b.Run("Protect in Closed state,with time caching and disabled concurrency limiter", func(b *testing.B) {
		for _, wrapper := range apiWrappers {
			b.Run(wrapper.name, func(b *testing.B) {
				cb := mustInitCB(b, &Config{
					MaxConcurrentCalls: -1,
					UseCachedTime:      true,
				})
				benchmarkProtect(b, cb, wrapper.fn)
			})
		}
	})

	b.Run("Protect in Closed state,with time caching and disabled concurrency limiter; metrics disabled", func(b *testing.B) {
		for _, wrapper := range apiWrappers {
			b.Run(wrapper.name, func(b *testing.B) {
				cb := mustInitCB(b, &Config{
					MaxConcurrentCalls: -1,
					UseCachedTime:      true,
					MetricConfig:       MetricConfig{IsDisabled: true},
				})
				benchmarkProtect(b, cb, wrapper.fn)
			})
		}
	})
}

func BenchmarkProtected_Closed_DisableErrorPercentTimeoutPercentThreshold(b *testing.B) {
	b.Run("disable error percent & timeout percent threshold", func(b *testing.B) {
		for _, wrapper := range apiWrappers {
			b.Run(wrapper.name, func(b *testing.B) {
				cb := mustInitCB(b, &Config{
					ErrorPercentThreshold:   -1,
					TimeoutPercentThreshold: -1,
					TimeoutThreshold:        1000,
				})
				benchmarkProtect(b, cb, wrapper.fn)
			})
		}
	})

	b.Run("enable error percent & timeout percent threshold", func(b *testing.B) {
		for _, wrapper := range apiWrappers {
			b.Run(wrapper.name, func(b *testing.B) {
				cb := mustInitCB(b, &Config{
					ErrorPercentThreshold:   50,
					TimeoutPercentThreshold: 90,
					TimeoutThreshold:        1000,
				})
				benchmarkProtect(b, cb, wrapper.fn)
			})
		}
	})
}

func benchmarkProtect(b *testing.B, cb *CircuitBreaker, cbAPIWrapper cbAPIWrapper) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cbAPIWrapper(cb, func() error {
			return nil
		}, nil)
	}
}

func GetWithMap(parallelism int, b *testing.B) {
	mapMutex := sync.RWMutex{}
	breakers := make(map[string]*CircuitBreaker)
	cb, _ := newCircuitBreaker("cb", nil)
	breakers["cb"] = cb

	b.SetParallelism(parallelism)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			get := func() {
				mapMutex.RLock()
				defer mapMutex.RUnlock()
				_ = breakers["cb"]
			}
			get()
		}
	})

}

func GetWithSyncMap(parallelism int, b *testing.B) {
	cbKey := "cb"
	syncMap := sync.Map{}

	cb, _ := newCircuitBreaker(cbKey, nil)
	syncMap.Store(cbKey, cb)

	b.SetParallelism(parallelism)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			get := func() {
				_, _ = syncMap.Load("cb")
			}
			get()
		}
	})

}

func BenchmarkGetWithMapAndSyncMap(b *testing.B) {
	maxThreads := 500000
	numCPU := runtime.NumCPU()
	for i := numCPU; i < maxThreads; i *= 2 {
		parallelism := i / numCPU
		b.Run(fmt.Sprintf("GetWithMap_%d", parallelism), func(b *testing.B) { GetWithMap(parallelism, b) })
		b.Run(fmt.Sprintf("GetWithSyncMap_%d", parallelism), func(b *testing.B) { GetWithSyncMap(parallelism, b) })
	}
}

func BenchmarkProtect_Open(b *testing.B) {
	for _, wrapper := range apiWrappers {
		b.ResetTimer()
		cb := mustInitCB(b, nil)
		cb.StateTransition(context.Background(), cb.state, Open, manualTransition)
		b.Run(wrapper.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				err := wrapper.fn(cb, func() error {
					return nil
				}, nil)
				if err != ErrCircuitOpenState {
					b.Fatal()
				}
			}
		})
	}
}

func BenchmarkProtect_SemiOpen(b *testing.B) {
	for _, wrapper := range apiWrappers {
		resetBreakers()
		cb := mustInitCB(b, &Config{
			MinimumNumberOfCalls:         100000000,
			NumberOfCallsInSemiOpenState: 100000000,
		})
		cb.StateTransition(context.Background(), cb.state, SemiOpen, manualTransition)
		b.ResetTimer()
		b.Run(wrapper.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = wrapper.fn(cb, func() error {
					return nil
				}, nil)
			}
		})
	}
}
