package loadbalancer

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type Backend struct {
	ID        string        `json:"id"`
	URL       string        `json:"url"`
	Weight    int           `json:"weight"`
	IsActive  bool          `json:"is_active"`
	Health    float64       `json:"health"` // 0.0 - 1.0
	Latency   time.Duration `json:"latency"`
	LastCheck time.Time     `json:"last_check"`
	mu        sync.RWMutex  `json:"-"`
}

type LoadBalancer struct {
	backends    []*Backend
	strategy    LoadBalancingStrategy
	healthCheck HealthChecker
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
}

type LoadBalancingStrategy interface {
	SelectBackend(backends []*Backend) *Backend
}

type HealthChecker interface {
	CheckHealth(backend *Backend) error
}

type RoundRobinStrategy struct {
	current int
	mu      sync.Mutex
}

type WeightedRoundRobinStrategy struct {
	current int
	mu      sync.Mutex
}

type LeastConnectionsStrategy struct {
	mu sync.Mutex
}

type HealthCheckerImpl struct {
	timeout time.Duration
}

func NewLoadBalancer(strategy LoadBalancingStrategy, healthCheck HealthChecker) *LoadBalancer {
	ctx, cancel := context.WithCancel(context.Background())

	lb := &LoadBalancer{
		strategy:    strategy,
		healthCheck: healthCheck,
		ctx:         ctx,
		cancel:      cancel,
	}

	go lb.startHealthMonitoring()

	return lb
}

func (lb *LoadBalancer) AddBackend(backend *Backend) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.backends = append(lb.backends, backend)
}

func (lb *LoadBalancer) RemoveBackend(backendID string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for i, backend := range lb.backends {
		if backend.ID == backendID {
			lb.backends = append(lb.backends[:i], lb.backends[i+1:]...)
			break
		}
	}
}

func (lb *LoadBalancer) GetBackend() (*Backend, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	activeBackends := make([]*Backend, 0)
	for _, backend := range lb.backends {
		if backend.IsActive {
			activeBackends = append(activeBackends, backend)
		}
	}

	if len(activeBackends) == 0 {
		return nil, fmt.Errorf("no active backends available")
	}

	return lb.strategy.SelectBackend(activeBackends), nil
}

func (lb *LoadBalancer) startHealthMonitoring() {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-lb.ctx.Done():
			return
		case <-ticker.C:
			lb.performHealthCheck()
		}
	}
}

func (lb *LoadBalancer) performHealthCheck() {
	lb.mu.RLock()
	backends := make([]*Backend, len(lb.backends))
	copy(backends, lb.backends)
	lb.mu.RUnlock()

	for _, backend := range backends {
		go lb.checkBackendHealth(backend)
	}
}

func (lb *LoadBalancer) checkBackendHealth(backend *Backend) {
	start := time.Now()

	err := lb.healthCheck.CheckHealth(backend)
	latency := time.Since(start)

	backend.mu.Lock()
	defer backend.mu.Unlock()

	backend.LastCheck = time.Now()
	backend.Latency = latency

	if err != nil {
		backend.IsActive = false
		backend.Health = 0.0
	} else {
		backend.IsActive = true
		backend.Health = 1.0
	}
}

func (lb *LoadBalancer) GetBackends() []*Backend {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	backends := make([]*Backend, len(lb.backends))
	copy(backends, lb.backends)
	return backends
}

func (lb *LoadBalancer) GetStats() map[string]interface{} {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	stats := map[string]interface{}{
		"total_backends":    len(lb.backends),
		"active_backends":   0,
		"inactive_backends": 0,
		"average_latency":   time.Duration(0),
		"average_health":    0.0,
	}

	totalLatency := time.Duration(0)
	totalHealth := 0.0

	for _, backend := range lb.backends {
		if backend.IsActive {
			stats["active_backends"] = stats["active_backends"].(int) + 1
		} else {
			stats["inactive_backends"] = stats["inactive_backends"].(int) + 1
		}

		totalLatency += backend.Latency
		totalHealth += backend.Health
	}

	if len(lb.backends) > 0 {
		stats["average_latency"] = totalLatency / time.Duration(len(lb.backends))
		stats["average_health"] = totalHealth / float64(len(lb.backends))
	}

	return stats
}

func (lb *LoadBalancer) Close() {
	lb.cancel()
}

func (rr *RoundRobinStrategy) SelectBackend(backends []*Backend) *Backend {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	if len(backends) == 0 {
		return nil
	}

	backend := backends[rr.current]
	rr.current = (rr.current + 1) % len(backends)

	return backend
}

func (wrr *WeightedRoundRobinStrategy) SelectBackend(backends []*Backend) *Backend {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()

	if len(backends) == 0 {
		return nil
	}

	totalWeight := 0
	for _, backend := range backends {
		totalWeight += backend.Weight
	}

	if totalWeight == 0 {
		backend := backends[wrr.current]
		wrr.current = (wrr.current + 1) % len(backends)
		return backend
	}

	backend := backends[wrr.current]
	wrr.current = (wrr.current + 1) % len(backends)

	return backend
}

func (lc *LeastConnectionsStrategy) SelectBackend(backends []*Backend) *Backend {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	if len(backends) == 0 {
		return nil
	}

	var bestBackend *Backend
	bestLatency := time.Duration(1<<63 - 1)

	for _, backend := range backends {
		if backend.Latency < bestLatency {
			bestLatency = backend.Latency
			bestBackend = backend
		}
	}

	return bestBackend
}

func NewHealthChecker(timeout time.Duration) *HealthCheckerImpl {
	return &HealthCheckerImpl{
		timeout: timeout,
	}
}

func (hc *HealthCheckerImpl) CheckHealth(backend *Backend) error {
	ctx, cancel := context.WithTimeout(context.Background(), hc.timeout)
	defer cancel()

	select {
	case <-ctx.Done():
		return fmt.Errorf("health check timeout")
	case <-time.After(time.Duration(rand.Intn(100)) * time.Millisecond):
		if rand.Float64() < 0.05 {
			return fmt.Errorf("simulated health check failure")
		}
		return nil
	}
}

func NewRoundRobinStrategy() *RoundRobinStrategy {
	return &RoundRobinStrategy{
		current: 0,
	}
}

func NewWeightedRoundRobinStrategy() *WeightedRoundRobinStrategy {
	return &WeightedRoundRobinStrategy{
		current: 0,
	}
}

func NewLeastConnectionsStrategy() *LeastConnectionsStrategy {
	return &LeastConnectionsStrategy{}
}
