package circuitbreaker

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

type Config struct {
	FailureThreshold    int           `json:"failure_threshold"`      // Başarısızlık eşiği
	SuccessThreshold    int           `json:"success_threshold"`      // Başarı eşiği
	Timeout             time.Duration `json:"timeout"`                // Açık durumda kalma süresi
	HalfOpenMaxRequests int           `json:"half_open_max_requests"` // Half-open durumunda maksimum istek
	WindowSize          time.Duration `json:"window_size"`            // Sliding window boyutu
	MinRequestCount     int           `json:"min_request_count"`      // Minimum istek sayısı
}

type CircuitBreaker struct {
	name            string
	config          Config
	state           State
	counts          *Counts
	lastError       error
	lastStateChange time.Time
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
}

type Counts struct {
	Requests             int64        `json:"requests"`
	TotalErrors          int64        `json:"total_errors"`
	ConsecutiveErrors    int64        `json:"consecutive_errors"`
	ConsecutiveSuccesses int64        `json:"consecutive_successes"`
	LastErrorTime        time.Time    `json:"last_error_time"`
	mu                   sync.RWMutex `json:"-"`
}

type Result struct {
	Allowed bool          `json:"allowed"`
	State   State         `json:"state"`
	Error   error         `json:"error,omitempty"`
	Latency time.Duration `json:"latency,omitempty"`
}

func NewCircuitBreaker(name string, config Config) *CircuitBreaker {
	ctx, cancel := context.WithCancel(context.Background())

	cb := &CircuitBreaker{
		name:            name,
		config:          config,
		state:           StateClosed,
		counts:          &Counts{},
		lastStateChange: time.Now(),
		ctx:             ctx,
		cancel:          cancel,
	}

	go cb.monitorState()

	return cb
}

func (cb *CircuitBreaker) Execute(fn func() error) error {
	if !cb.Ready() {
		return fmt.Errorf("circuit breaker %s is %s", cb.name, cb.state)
	}

	cb.counts.mu.Lock()
	cb.counts.Requests++
	cb.counts.mu.Unlock()

	start := time.Now()
	err := fn()
	latency := time.Since(start)

	cb.recordResult(err, latency)

	return err
}

func (cb *CircuitBreaker) ExecuteWithContext(ctx context.Context, fn func() error) error {
	if !cb.Ready() {
		return fmt.Errorf("circuit breaker %s is %s", cb.name, cb.state)
	}

	cb.counts.mu.Lock()
	cb.counts.Requests++
	cb.counts.mu.Unlock()

	start := time.Now()

	resultChan := make(chan error, 1)
	go func() {
		resultChan <- fn()
	}()

	select {
	case err := <-resultChan:
		latency := time.Since(start)
		cb.recordResult(err, latency)
		return err
	case <-ctx.Done():
		latency := time.Since(start)
		cb.recordResult(ctx.Err(), latency)
		return ctx.Err()
	}
}

func (cb *CircuitBreaker) Ready() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(cb.lastStateChange) >= cb.config.Timeout {
			cb.transitionToHalfOpen()
			return true
		}
		return false
	case StateHalfOpen:
		cb.counts.mu.RLock()
		requests := cb.counts.Requests
		cb.counts.mu.RUnlock()

		return requests < int64(cb.config.HalfOpenMaxRequests)
	default:
		return false
	}
}

func (cb *CircuitBreaker) recordResult(err error, latency time.Duration) {
	cb.counts.mu.Lock()
	defer cb.counts.mu.Unlock()

	if err != nil {
		cb.counts.TotalErrors++
		cb.counts.ConsecutiveErrors++
		cb.counts.ConsecutiveSuccesses = 0
		cb.counts.LastErrorTime = time.Now()
		cb.lastError = err

		if cb.shouldOpen() {
			cb.transitionToOpen()
		}
	} else {
		cb.counts.ConsecutiveSuccesses++
		cb.counts.ConsecutiveErrors = 0

		if cb.shouldClose() {
			cb.transitionToClosed()
		}
	}
}

func (cb *CircuitBreaker) shouldOpen() bool {
	if cb.counts.Requests < int64(cb.config.MinRequestCount) {
		return false
	}

	return cb.counts.ConsecutiveErrors >= int64(cb.config.FailureThreshold)
}

func (cb *CircuitBreaker) shouldClose() bool {
	if cb.state != StateHalfOpen {
		return false
	}

	return cb.counts.ConsecutiveSuccesses >= int64(cb.config.SuccessThreshold)
}

func (cb *CircuitBreaker) transitionToOpen() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state != StateOpen {
		cb.state = StateOpen
		cb.lastStateChange = time.Now()
		fmt.Printf("Circuit breaker %s: CLOSED -> OPEN\n", cb.name)
	}
}

func (cb *CircuitBreaker) transitionToHalfOpen() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateOpen {
		cb.state = StateHalfOpen
		cb.lastStateChange = time.Now()

		cb.counts.mu.Lock()
		cb.counts.Requests = 0
		cb.counts.ConsecutiveErrors = 0
		cb.counts.ConsecutiveSuccesses = 0
		cb.counts.mu.Unlock()

		fmt.Printf("Circuit breaker %s: OPEN -> HALF_OPEN\n", cb.name)
	}
}

func (cb *CircuitBreaker) transitionToClosed() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateHalfOpen {
		cb.state = StateClosed
		cb.lastStateChange = time.Now()

		cb.counts.mu.Lock()
		cb.counts.Requests = 0
		cb.counts.TotalErrors = 0
		cb.counts.ConsecutiveErrors = 0
		cb.counts.ConsecutiveSuccesses = 0
		cb.counts.mu.Unlock()

		fmt.Printf("Circuit breaker %s: HALF_OPEN -> CLOSED\n", cb.name)
	}
}

func (cb *CircuitBreaker) monitorState() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-cb.ctx.Done():
			return
		case <-ticker.C:
			cb.checkStateTransition()
		}
	}
}

func (cb *CircuitBreaker) checkStateTransition() {
	cb.mu.RLock()
	state := cb.state
	lastChange := cb.lastStateChange
	cb.mu.RUnlock()

	if state == StateOpen && time.Since(lastChange) >= cb.config.Timeout {
		cb.transitionToHalfOpen()
	}
}

func (cb *CircuitBreaker) GetState() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

func (cb *CircuitBreaker) GetCounts() Counts {
	cb.counts.mu.RLock()
	defer cb.counts.mu.RUnlock()

	return Counts{
		Requests:             cb.counts.Requests,
		TotalErrors:          cb.counts.TotalErrors,
		ConsecutiveErrors:    cb.counts.ConsecutiveErrors,
		ConsecutiveSuccesses: cb.counts.ConsecutiveSuccesses,
		LastErrorTime:        cb.counts.LastErrorTime,
	}
}

func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	cb.mu.RLock()
	state := cb.state
	lastStateChange := cb.lastStateChange
	lastError := cb.lastError
	cb.mu.RUnlock()

	counts := cb.GetCounts()

	stats := map[string]interface{}{
		"name":                  cb.name,
		"state":                 state.String(),
		"last_state_change":     lastStateChange,
		"last_error":            lastError,
		"requests":              counts.Requests,
		"total_errors":          counts.TotalErrors,
		"consecutive_errors":    counts.ConsecutiveErrors,
		"consecutive_successes": counts.ConsecutiveSuccesses,
		"error_rate":            0.0,
		"last_error_time":       counts.LastErrorTime,
	}

	if counts.Requests > 0 {
		stats["error_rate"] = float64(counts.TotalErrors) / float64(counts.Requests)
	}

	return stats
}

func (cb *CircuitBreaker) ForceOpen() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateOpen
	cb.lastStateChange = time.Now()
	fmt.Printf("Circuit breaker %s: FORCED OPEN\n", cb.name)
}

func (cb *CircuitBreaker) ForceClose() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.lastStateChange = time.Now()

	cb.counts.mu.Lock()
	cb.counts.Requests = 0
	cb.counts.TotalErrors = 0
	cb.counts.ConsecutiveErrors = 0
	cb.counts.ConsecutiveSuccesses = 0
	cb.counts.mu.Unlock()

	fmt.Printf("Circuit breaker %s: FORCED CLOSED\n", cb.name)
}

func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.lastStateChange = time.Now()
	cb.lastError = nil

	cb.counts.mu.Lock()
	cb.counts.Requests = 0
	cb.counts.TotalErrors = 0
	cb.counts.ConsecutiveErrors = 0
	cb.counts.ConsecutiveSuccesses = 0
	cb.counts.mu.Unlock()

	fmt.Printf("Circuit breaker %s: RESET\n", cb.name)
}

func (cb *CircuitBreaker) Close() {
	cb.cancel()
}

func DefaultConfig() Config {
	return Config{
		FailureThreshold:    5,
		SuccessThreshold:    3,
		Timeout:             60 * time.Second,
		HalfOpenMaxRequests: 3,
		WindowSize:          10 * time.Second,
		MinRequestCount:     10,
	}
}

func StrictConfig() Config {
	return Config{
		FailureThreshold:    3,
		SuccessThreshold:    5,
		Timeout:             30 * time.Second,
		HalfOpenMaxRequests: 2,
		WindowSize:          5 * time.Second,
		MinRequestCount:     5,
	}
}

func LenientConfig() Config {
	return Config{
		FailureThreshold:    10,
		SuccessThreshold:    2,
		Timeout:             120 * time.Second,
		HalfOpenMaxRequests: 5,
		WindowSize:          30 * time.Second,
		MinRequestCount:     20,
	}
}
