package fallback

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type FallbackStrategy interface {
	Execute(ctx context.Context, primary func() error, fallbacks []func() error) error
}

type FallbackConfig struct {
	MaxRetries        int           `json:"max_retries"`
	RetryDelay        time.Duration `json:"retry_delay"`
	Timeout           time.Duration `json:"timeout"`
	EnableCaching     bool          `json:"enable_caching"`
	CacheTTL          time.Duration `json:"cache_ttl"`
	EnableDegradation bool          `json:"enable_degradation"`
}

type FallbackManager struct {
	config   FallbackConfig
	strategy FallbackStrategy
	cache    *FallbackCache
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
}

type FallbackCache struct {
	data map[string]*CacheEntry
	mu   sync.RWMutex
}

type CacheEntry struct {
	Data      interface{}   `json:"data"`
	Timestamp time.Time     `json:"timestamp"`
	TTL       time.Duration `json:"ttl"`
}

type SequentialFallbackStrategy struct {
	config FallbackConfig
}

type ParallelFallbackStrategy struct {
	config FallbackConfig
}

type DegradationFallbackStrategy struct {
	config FallbackConfig
}

func NewFallbackManager(config FallbackConfig, strategy FallbackStrategy) *FallbackManager {
	ctx, cancel := context.WithCancel(context.Background())

	fm := &FallbackManager{
		config:   config,
		strategy: strategy,
		cache:    &FallbackCache{data: make(map[string]*CacheEntry)},
		ctx:      ctx,
		cancel:   cancel,
	}

	if config.EnableCaching {
		go fm.startCacheCleanup()
	}

	return fm
}

func (fm *FallbackManager) Execute(ctx context.Context, key string, primary func() (interface{}, error), fallbacks ...func() (interface{}, error)) (interface{}, error) {
	if fm.config.EnableCaching {
		if cached, found := fm.cache.Get(key); found {
			return cached, nil
		}
	}

	var result interface{}
	var err error

	primaryFn := func() error {
		var primaryErr error
		result, primaryErr = primary()
		return primaryErr
	}

	fallbackFns := make([]func() error, len(fallbacks))
	for i, fallback := range fallbacks {
		fallbackFns[i] = func() error {
			var fallbackErr error
			result, fallbackErr = fallback()
			return fallbackErr
		}
	}

	err = fm.strategy.Execute(ctx, primaryFn, fallbackFns)

	if err == nil && fm.config.EnableCaching {
		fm.cache.Set(key, result, fm.config.CacheTTL)
	}

	return result, err
}

func (fm *FallbackManager) ExecuteWithDegradation(ctx context.Context, key string, primary func() (interface{}, error), degraded func() (interface{}, error)) (interface{}, error) {
	if !fm.config.EnableDegradation {
		return fm.Execute(ctx, key, primary)
	}

	result, err := primary()
	if err == nil {
		return result, nil
	}

	degradedResult, degradedErr := degraded()
	if degradedErr != nil {
		return nil, fmt.Errorf("both primary and degraded functions failed: primary: %v, degraded: %v", err, degradedErr)
	}

	fmt.Printf("Degradation activated for key: %s, primary error: %v\n", key, err)

	return degradedResult, nil
}

func (fm *FallbackManager) startCacheCleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-fm.ctx.Done():
			return
		case <-ticker.C:
			fm.cache.Cleanup()
		}
	}
}

func (fc *FallbackCache) Get(key string) (interface{}, bool) {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	entry, exists := fc.data[key]
	if !exists {
		return nil, false
	}

	if time.Since(entry.Timestamp) > entry.TTL {
		delete(fc.data, key)
		return nil, false
	}

	return entry.Data, true
}

func (fc *FallbackCache) Set(key string, data interface{}, ttl time.Duration) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	fc.data[key] = &CacheEntry{
		Data:      data,
		Timestamp: time.Now(),
		TTL:       ttl,
	}
}

func (fc *FallbackCache) Cleanup() {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	now := time.Now()
	for key, entry := range fc.data {
		if now.Sub(entry.Timestamp) > entry.TTL {
			delete(fc.data, key)
		}
	}
}

func NewSequentialFallbackStrategy(config FallbackConfig) *SequentialFallbackStrategy {
	return &SequentialFallbackStrategy{config: config}
}

func (s *SequentialFallbackStrategy) Execute(ctx context.Context, primary func() error, fallbacks []func() error) error {
	err := primary()
	if err == nil {
		return nil
	}

	for i, fallback := range fallbacks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if i > 0 && s.config.RetryDelay > 0 {
			time.Sleep(s.config.RetryDelay)
		}

		err = fallback()
		if err == nil {
			return nil
		}
	}

	return fmt.Errorf("all fallback attempts failed")
}

func NewParallelFallbackStrategy(config FallbackConfig) *ParallelFallbackStrategy {
	return &ParallelFallbackStrategy{config: config}
}

func (p *ParallelFallbackStrategy) Execute(ctx context.Context, primary func() error, fallbacks []func() error) error {
	err := primary()
	if err == nil {
		return nil
	}

	if len(fallbacks) == 0 {
		return err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, p.config.Timeout)
	defer cancel()

	resultChan := make(chan error, len(fallbacks))

	for _, fallback := range fallbacks {
		go func(fn func() error) {
			resultChan <- fn()
		}(fallback)
	}

	successCount := 0
	for i := 0; i < len(fallbacks); i++ {
		select {
		case <-timeoutCtx.Done():
			return timeoutCtx.Err()
		case fallbackErr := <-resultChan:
			if fallbackErr == nil {
				successCount++
			}
		}
	}

	if successCount > 0 {
		return nil
	}

	return fmt.Errorf("all parallel fallback attempts failed")
}

func NewDegradationFallbackStrategy(config FallbackConfig) *DegradationFallbackStrategy {
	return &DegradationFallbackStrategy{config: config}
}

func (d *DegradationFallbackStrategy) Execute(ctx context.Context, primary func() error, fallbacks []func() error) error {
	err := primary()
	if err == nil {
		return nil
	}

	if !d.config.EnableDegradation {
		return err
	}

	for i, fallback := range fallbacks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if i > 0 && d.config.RetryDelay > 0 {
			time.Sleep(d.config.RetryDelay)
		}

		fallbackErr := fallback()
		if fallbackErr == nil {
			fmt.Printf("Degradation activated: fallback %d succeeded\n", i+1)
			return nil
		}
	}

	return fmt.Errorf("all fallback attempts failed, including degradation")
}

func DefaultConfig() FallbackConfig {
	return FallbackConfig{
		MaxRetries:        3,
		RetryDelay:        1 * time.Second,
		Timeout:           30 * time.Second,
		EnableCaching:     true,
		CacheTTL:          5 * time.Minute,
		EnableDegradation: true,
	}
}

func StrictConfig() FallbackConfig {
	return FallbackConfig{
		MaxRetries:        1,
		RetryDelay:        500 * time.Millisecond,
		Timeout:           10 * time.Second,
		EnableCaching:     false,
		CacheTTL:          1 * time.Minute,
		EnableDegradation: false,
	}
}

func LenientConfig() FallbackConfig {
	return FallbackConfig{
		MaxRetries:        5,
		RetryDelay:        2 * time.Second,
		Timeout:           60 * time.Second,
		EnableCaching:     true,
		CacheTTL:          10 * time.Minute,
		EnableDegradation: true,
	}
}

func (fm *FallbackManager) Close() {
	fm.cancel()
}

func (fm *FallbackManager) GetStats() map[string]interface{} {
	fm.cache.mu.RLock()
	cacheSize := len(fm.cache.data)
	fm.cache.mu.RUnlock()

	return map[string]interface{}{
		"cache_size":         cacheSize,
		"enable_caching":     fm.config.EnableCaching,
		"enable_degradation": fm.config.EnableDegradation,
		"max_retries":        fm.config.MaxRetries,
		"retry_delay":        fm.config.RetryDelay,
		"timeout":            fm.config.Timeout,
		"cache_ttl":          fm.config.CacheTTL,
	}
}
