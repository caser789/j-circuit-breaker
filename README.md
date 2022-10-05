circuit breaker
====================

The Circuit Breaker library is a fault tolerance library implementing the circuit breaker design pattern.

![xx](./img.png)

## Quick Start
A `CircuitBreaker`  instance with a given name can be created using the `Get` function. Further calls to `Get()`  using the same provided name will return the same instance of a `CircuitBreaker`.
Define the logic which calls a dependency and pass this function to `Protect()`. If the system is healthy i.e. `CLOSED` or `SEMI_OPEN` state, the passed function will be executed.

### Initialising a CircuitBreaker instance
```go

var cb *circuitbreaker.CircuitBreaker
 
func init() {
    var err error
 
    // Initialise a Config with default values
    config := circuitbreaker.NewDefaultConfig()
 
    // Alter the values of some fields
    config.SlidingWindowSize = 20
    config.ErrorPercentThreshold = 40
 
    // Initialise a CircuitBreaker with the created Config
    cb, err = circuitbreaker.Get("rpc_cb", &config)
    if err != nil {
        panic(err)
    }
}
```
