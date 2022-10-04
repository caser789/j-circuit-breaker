package circuitbreaker

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// StateCountGaugeMetric records the current state of each circuitbreaker instance.
	// Value is either 1 or 0.
	StateCountGaugeMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "circuitbreaker_state_count",
	}, []string{"name", "state"})
	// RequestTotalMetric records the outcome of all requests.
	RequestTotalMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "circuitbreaker_request_total",
	}, []string{"name", "type"})
	// ConcurrentCountGaugeMetric records the current concurrency count for a circuitbreaker instance.
	ConcurrentCountGaugeMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "circuitbreaker_concurrent_request_count",
	}, []string{"name"})
	// ErrorPercentGaugeMetric records the current error percent of the cb window.
	ErrorPercentGaugeMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "circuitbreaker_error_percent",
	}, []string{"name"})
	// TimeoutPercentGaugeMetric records the current error percent of the cb window.
	TimeoutPercentGaugeMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "circuitbreaker_timeout_percent",
	}, []string{"name"})
	// ConfigGaugeMetric exposes the configuration values of a circuitbreaker.
	ConfigGaugeMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "circuitbreaker_config",
	}, []string{"name", "type"})
)

func setBreakerStateMetric(name string, state State) {
	stateList := []State{
		Closed, Open, SemiOpen, ForcedOpen, ForcedClosed,
	}

	for _, st := range stateList {
		if st == state {
			StateCountGaugeMetric.WithLabelValues(name, state.String()).Set(1)
		} else {
			StateCountGaugeMetric.WithLabelValues(name, st.String()).Set(0)
		}
	}
}

func addRequestTotalMetric(name string, reqType string) {
	RequestTotalMetric.WithLabelValues(name, reqType).Inc()
}

func setWindowSnapshotMetrics(name string, snapshot iConcurrentCallGetter) {
	ConcurrentCountGaugeMetric.WithLabelValues(name).Set(float64(snapshot.GetNumConcurrentCalls()))
	ErrorPercentGaugeMetric.WithLabelValues(name).Set(float64(snapshot.GetErrorPercent()))
	TimeoutPercentGaugeMetric.WithLabelValues(name).Set(float64(snapshot.GetTimeoutPercent()))
}

func setThresholdsMetrics(name string, config Config) {
	ConfigGaugeMetric.WithLabelValues(name, "error_percent_threshold").Set(float64(config.ErrorPercentThreshold))
	ConfigGaugeMetric.WithLabelValues(name, "timeout_percent_threshold").Set(float64(config.TimeoutPercentThreshold))
	ConfigGaugeMetric.WithLabelValues(name, "timeout_threshold_millis").Set(float64(config.TimeoutThreshold / time.Millisecond))
	ConfigGaugeMetric.WithLabelValues(name, "max_concurrent_calls").Set(float64(config.MaxConcurrentCalls))
	ConfigGaugeMetric.WithLabelValues(name, "wait_duration_in_open_state_secs").Set(float64(config.WaitDurationInOpenState / time.Second))
}
