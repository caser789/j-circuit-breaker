package circuitbreaker

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

// MetricConfig configures metrics for a CircuitBreaker.
type MetricConfig struct {
	// IsDisabled disables metric reporting for a CircuitBreaker.
	// Defaults to false.
	IsDisabled bool `json:"is_disabled"`
}

// Config models the behavior of a circuitbreaker instance.
type Config struct {
	// When window error failure rate is equal or greater than ErrorPercentThreshold, CircuitBreaker transitions from CLOSED to OPEN.
	// If ErrorPercentThreshold and TimeoutPercentThreshold are both 0, this defaults to 50.
	// Otherwise, if ErrorPercentThreshold == 0 and TimeoutPercentThreshold > 0, then ErrorPercentThreshold checking is disabled.
	// Negative value disables ErrorPercentThreshold checking.
	ErrorPercentThreshold int `json:"error_percent_threshold"`

	// When window timeout failure rate is equal or greater than TimeoutPercentThreshold, CircuitBreaker transitions from CLOSED to OPEN.
	// The CircuitBreaker considers a call as timeout when the call duration is greater than TimeoutThreshold.
	// A value of 0 or negative disables TimeoutPercentThreshold checking. TimeoutPercentThreshold is disabled by default.
	TimeoutPercentThreshold int `json:"timeout_percent_threshold"`

	// The duration threshold above which calls are considered as timeout.
	// If TimeoutPercentThreshold checking is disabled and TimeoutThreshold is enabled, timeouts will still be reported to the metrics.
	// Defaults to 0. A value of 0 disables TimeoutThreshold checking.
	TimeoutThreshold time.Duration `json:"timeout_threshold"`

	// The minimum number of calls in the window required before the CircuitBreaker calculates the error rate and timeout rate.
	// If the number of calls in the window is less than minimumNumberOfCalls, the CircuitBreaker
	// will not transition to OPEN even if the error rate of those calls exceed the threshold.
	// Defaults to 100.
	MinimumNumberOfCalls int `json:"minimum_number_of_calls"`

	// The number of calls that is allowed while in SEMI_OPEN state. The error rate is calculated for these
	// calls and compared with the thresholds before transitioning to CLOSED or OPEN state.
	// Defaults to 100.
	NumberOfCallsInSemiOpenState int `json:"number_of_calls_in_semi_open_state"`

	// The duration to wait before transitioning from OPEN to SEMI_OPEN.
	// Defaults to 10s.
	WaitDurationInOpenState time.Duration `json:"wait_duration_in_open_state"`

	// Maximum number of concurrent calls permitted.
	// Intends to prevent the service from suffering due to high latency of dependencies.
	// The concurrency limiter can be disabled by setting this field to any negative value.
	// Defaults to 10000.
	MaxConcurrentCalls int `json:"max_concurrent_calls"`

	// MaxWaitDurationMillis is the maximum time a call will block while waiting to acquire permission from the concurrency limiter.
	// Defaults to 0 i.e. if concurrency limiter is full, then new requests will immediately be rejected and not wait.
	MaxWaitDurationMillis int `json:"max_wait_duration_millis"`

	// Size of the time-based sliding window where the calls of the last slidingWindowSize seconds are recorded.
	// Defaults to 10 (seconds), means the window has 10 buckets (each bucket last for one second).
	SlidingWindowSize int `json:"sliding_window_size"`

	// UseCachedTime indicates whether to use cached time for this circuitbreaker. Time is cached in millisecond intervals.
	// Cached time is disabled by default.
	UseCachedTime bool `json:"use_cached_time"`

	// MetricConfig configures metrics for this CircuitBreaker instance.
	MetricConfig MetricConfig `json:"metric_config"`
}

func (c *Config) isEqual(config *Config) bool {
	return reflect.DeepEqual(c, config)
}

func (c *Config) GetErrorPercentThreshold() int {
	return c.ErrorPercentThreshold
}

func (c *Config) GetTimeoutPercentThreshold() int {
	return c.TimeoutPercentThreshold
}

// IsValidConfig validates a provided config.
func IsValidConfig(config *Config) error {
	var errstrings []string

	if config == nil {
		return fmt.Errorf("%w:nil_config", ErrInvalidConfig)
	}

	if config.SlidingWindowSize < 0 {
		errstrings = append(errstrings, fmt.Sprintf("window_size:%d", config.SlidingWindowSize))
	}
	if config.TimeoutThreshold < 0 {
		errstrings = append(errstrings, fmt.Sprintf("timeout_threshold:%d", config.TimeoutThreshold))
	}
	if config.MinimumNumberOfCalls < 0 {
		errstrings = append(errstrings, fmt.Sprintf("minimum_num_calls:%d", config.MinimumNumberOfCalls))
	}
	if config.TimeoutPercentThreshold > 100 {
		errstrings = append(errstrings, fmt.Sprintf("timeout_threshold:%d", config.TimeoutPercentThreshold))
	}
	if config.ErrorPercentThreshold > 100 {
		errstrings = append(errstrings, fmt.Sprintf("error_threshold:%d", config.ErrorPercentThreshold))
	}
	if config.WaitDurationInOpenState < 0 {
		errstrings = append(errstrings, fmt.Sprintf("time_in_open:%d", config.WaitDurationInOpenState))
	}
	if config.NumberOfCallsInSemiOpenState < 0 {
		errstrings = append(errstrings, fmt.Sprintf("num_calls_in_semi_open:%d", config.NumberOfCallsInSemiOpenState))
	}

	if len(errstrings) > 0 {
		return fmt.Errorf("%w:%v", ErrInvalidConfig, strings.Join(errstrings, ","))
	}

	return nil
}

// setting is a wrapper of Config which also includes a slice of healthChecker.
// Each CircuitBreaker has a setting associated to it.
type setting struct {
	Config
	checkers []healthChecker
}

func (s setting) isThresholdsExceed(snapshot iSnapshot) (bool, transitionReason) {
	for _, checker := range s.checkers {
		if exceeded, reason := checker.isThresholdExceed(snapshot, &s); exceeded {
			return true, reason
		}
	}
	return false, invalidTransition
}

func (s setting) isTimeout(duration time.Duration) bool {
	if s.TimeoutThreshold > 0 {
		return duration >= s.TimeoutThreshold
	}
	return false
}

// NewDefaultConfig returns a Config with the default configurations.
func NewDefaultConfig() Config {
	return Config{
		ErrorPercentThreshold:        50,
		TimeoutPercentThreshold:      0,
		TimeoutThreshold:             0,
		MinimumNumberOfCalls:         100,
		NumberOfCallsInSemiOpenState: 100,
		WaitDurationInOpenState:      10 * time.Second,
		MaxConcurrentCalls:           10000,
		MaxWaitDurationMillis:        0,
		SlidingWindowSize:            10,
		UseCachedTime:                false,
	}
}

func fillDefaultsAndGetSetting(config Config) setting {
	newConfigFromDefaults := NewDefaultConfig()

	// No checks enabled by default
	checkers := make([]healthChecker, 0)

	if config.ErrorPercentThreshold > 0 {
		newConfigFromDefaults.ErrorPercentThreshold = config.ErrorPercentThreshold
		checkers = append(checkers, errorPercentChecker)
	}

	if config.TimeoutPercentThreshold > 0 {
		newConfigFromDefaults.TimeoutPercentThreshold = config.TimeoutPercentThreshold
		checkers = append(checkers, timeoutPercentChecker)
	}

	if config.ErrorPercentThreshold == 0 && config.TimeoutPercentThreshold <= 0 {
		// Enable error percent checker which will be set to default value
		checkers = append(checkers, errorPercentChecker)
	}

	if config.WaitDurationInOpenState != 0 {
		newConfigFromDefaults.WaitDurationInOpenState = config.WaitDurationInOpenState
	}
	if config.SlidingWindowSize != 0 {
		newConfigFromDefaults.SlidingWindowSize = config.SlidingWindowSize
	}
	if config.MinimumNumberOfCalls != 0 {
		newConfigFromDefaults.MinimumNumberOfCalls = config.MinimumNumberOfCalls
	}
	if config.TimeoutThreshold != 0 {
		newConfigFromDefaults.TimeoutThreshold = config.TimeoutThreshold
	}
	if config.NumberOfCallsInSemiOpenState != 0 {
		newConfigFromDefaults.NumberOfCallsInSemiOpenState = config.NumberOfCallsInSemiOpenState
	}
	if config.MaxConcurrentCalls != 0 {
		newConfigFromDefaults.MaxConcurrentCalls = config.MaxConcurrentCalls
	}
	if config.MaxWaitDurationMillis > 0 {
		newConfigFromDefaults.MaxWaitDurationMillis = config.MaxWaitDurationMillis
	}

	newConfigFromDefaults.MetricConfig = config.MetricConfig
	newConfigFromDefaults.UseCachedTime = config.UseCachedTime

	newSetting := setting{
		Config:   newConfigFromDefaults,
		checkers: checkers,
	}
	return newSetting
}
