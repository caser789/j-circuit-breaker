package circuitbreaker

import (
	"fmt"
	"strings"
)

// Get returns a CircuitBreaker if it already exists in the breaker registry, ignoring the provided config.
// If breaker with given name does not exist in the breaker registry, Get creates a new breaker with the provided config
// and saves it in the registry and finally returns the new breaker.
// If provided config is nil, default config values are used.
// If provided name is empty, returns a new circuitbreaker that is not saved in the global registry.
// Returns a pointer to a CircuitBreaker instance.
func Get(name string, config *Config) (*CircuitBreaker, error) {
	if len(strings.TrimSpace(name)) == 0 {
		cb, err := newCircuitBreaker(name, config)
		return cb, err
	}

	cb, ok := breakersSyncMap.Load(name)

	if !ok {
		newCb, err := newCircuitBreaker(name, config)
		if err != nil {
			return nil, err
		}

		// We use LoadOrStore here in the case that another thread has beat this in
		// storing a new breaker
		loadedCb, _ := breakersSyncMap.LoadOrStore(name, newCb)
		return loadedCb.(*CircuitBreaker), nil
	}

	return cb.(*CircuitBreaker), nil
}

// GetExisting returns a CircuitBreaker if it already exists in the breaker registry.
// If breaker with given name does not exist in the breaker registry, GetExisting will return nil.
func GetExisting(name string) *CircuitBreaker {
	breaker, ok := breakersSyncMap.Load(name)

	if !ok {
		return nil
	}

	return breaker.(*CircuitBreaker)
}

// ForceOpen changes the current state of the CircuitBreaker to FORCED_OPEN.
// If breaker with given name does not exist in the breaker registry, ForceOpen will return ErrBreakerNotFound.
func ForceOpen(name string) error {
	breaker, ok := breakersSyncMap.Load(name)

	if !ok {
		return ErrBreakerNotFound
	}

	cb, _ := breaker.(*CircuitBreaker)
	cb.ForceOpen()
	return nil
}

// ForceClose changes the current state of the CircuitBreaker to FORCED_CLOSED.
// If breaker with given name does not exist in the breaker registry, ForceClose will return ErrBreakerNotFound.
func ForceClose(name string) error {
	breaker, ok := breakersSyncMap.Load(name)

	if !ok {
		return ErrBreakerNotFound
	}

	cb, _ := breaker.(*CircuitBreaker)
	cb.ForceClose()
	return nil
}

// Reset resets the CircuitBreaker's sliding window statistics and changes the state to CLOSED.
// If breaker with given name does not exist in the breaker registry, ForceClose will return ErrBreakerNotFound.
func Reset(name string) error {
	breaker, ok := breakersSyncMap.Load(name)

	if !ok {
		return ErrBreakerNotFound
	}

	cb, _ := breaker.(*CircuitBreaker)
	cb.Reset()
	return nil
}

// UpdateConfig updates the config of a single circuit breaker.
// This will also reset the CircuitBreaker i.e. changes state to closed state and resets the window.
// CircuitBreaker must exist in the global registry.
// If the existing config is exactly the same as the new config, no change will be made.
func UpdateConfig(name string, config *Config) error {
	breaker, ok := breakersSyncMap.Load(name)

	if !ok {
		return ErrBreakerNotFound
	}

	cb, _ := breaker.(*CircuitBreaker)

	return cb.UpdateConfig(config)
}

// UpdateConfigs updates the config of multiple circuit breakers.
// All entries must be valid otherwise no update will be executed for any CircuitBreaker.
func UpdateConfigs(configmap map[string]*Config) error {
	breakersMutex.Lock()
	defer breakersMutex.Unlock()

	// First, ensure all are entries are valid before updating configs of any breaker
	for k, v := range configmap {
		_, ok := breakersSyncMap.Load(k)

		if !ok {
			return fmt.Errorf("%s:breaker_not_found", k)
		}

		if configErr := IsValidConfig(v); configErr != nil {
			return fmt.Errorf("%s:%v", k, configErr)
		}
	}

	for k, v := range configmap {
		breaker, _ := breakersSyncMap.Load(k)

		cb, _ := breaker.(*CircuitBreaker)
		// oldConfig := cb.setting.Config
		cb.updateSetting(v)
	}

	return nil
}

// resetBreakers deletes all breakers from the global map. Mainly used for testing.
func resetBreakers() {
	breakersMutex.Lock()
	defer breakersMutex.Unlock()

	breakersSyncMap.Range(func(key, breaker interface{}) bool {
		breaker.(*CircuitBreaker).Reset()
		breakersSyncMap.Delete(key)
		return true
	})
}
