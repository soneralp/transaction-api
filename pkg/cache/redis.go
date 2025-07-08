package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"transaction-api-w-go/pkg/domain"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

type RedisCache struct {
	client *redis.Client
	logger domain.Logger
}

type CacheConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
	PoolSize int
}

func NewRedisCache(config CacheConfig, logger domain.Logger) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password: config.Password,
		DB:       config.DB,
		PoolSize: config.PoolSize,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisCache{
		client: client,
		logger: logger,
	}, nil
}

func (c *RedisCache) Close() error {
	return c.client.Close()
}

func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	err = c.client.Set(ctx, key, data, expiration).Err()
	if err != nil {
		return fmt.Errorf("failed to set cache key %s: %w", key, err)
	}

	c.logger.Debug("Cache set", "key", key, "expiration", expiration)
	return nil
}

func (c *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return domain.ErrCacheMiss
		}
		return fmt.Errorf("failed to get cache key %s: %w", key, err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("failed to unmarshal cached value: %w", err)
	}

	c.logger.Debug("Cache hit", "key", key)
	return nil
}

func (c *RedisCache) Delete(ctx context.Context, key string) error {
	err := c.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete cache key %s: %w", key, err)
	}

	c.logger.Debug("Cache delete", "key", key)
	return nil
}

func (c *RedisCache) DeletePattern(ctx context.Context, pattern string) error {
	iter := c.client.Scan(ctx, 0, pattern, 0).Iterator()
	var keys []string

	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if len(keys) > 0 {
		err := c.client.Del(ctx, keys...).Err()
		if err != nil {
			return fmt.Errorf("failed to delete cache pattern %s: %w", pattern, err)
		}
	}

	c.logger.Debug("Cache delete pattern", "pattern", pattern, "keys_count", len(keys))
	return nil
}

func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	result, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check cache key existence %s: %w", key, err)
	}

	return result > 0, nil
}

func (c *RedisCache) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return false, fmt.Errorf("failed to marshal value: %w", err)
	}

	result, err := c.client.SetNX(ctx, key, data, expiration).Result()
	if err != nil {
		return false, fmt.Errorf("failed to set NX cache key %s: %w", key, err)
	}

	c.logger.Debug("Cache set NX", "key", key, "result", result)
	return result, nil
}

func (c *RedisCache) Increment(ctx context.Context, key string, value int64) (int64, error) {
	result, err := c.client.IncrBy(ctx, key, value).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment cache key %s: %w", key, err)
	}

	return result, nil
}

func (c *RedisCache) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := c.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get TTL for cache key %s: %w", key, err)
	}

	return ttl, nil
}

func (c *RedisCache) FlushAll(ctx context.Context) error {
	err := c.client.FlushAll(ctx).Err()
	if err != nil {
		return fmt.Errorf("failed to flush all cache: %w", err)
	}

	c.logger.Info("Cache flushed all")
	return nil
}

func (c *RedisCache) GetStats(ctx context.Context) (*CacheStats, error) {
	info, err := c.client.Info(ctx, "stats").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis stats: %w", err)
	}

	stats := &CacheStats{
		Info: info,
	}

	dbSize, err := c.client.DBSize(ctx).Result()
	if err == nil {
		stats.DBSize = dbSize
	}

	return stats, nil
}

type CacheStats struct {
	Info   string `json:"info"`
	DBSize int64  `json:"db_size"`
}

type CacheKeyGenerator struct{}

func NewCacheKeyGenerator() *CacheKeyGenerator {
	return &CacheKeyGenerator{}
}

func (g *CacheKeyGenerator) UserKey(userID uuid.UUID) string {
	return fmt.Sprintf("user:%s", userID.String())
}

func (g *CacheKeyGenerator) TransactionKey(transactionID uuid.UUID) string {
	return fmt.Sprintf("transaction:%s", transactionID.String())
}

func (g *CacheKeyGenerator) UserTransactionsKey(userID uuid.UUID, limit, offset int) string {
	return fmt.Sprintf("user_transactions:%s:%d:%d", userID.String(), limit, offset)
}

func (g *CacheKeyGenerator) BalanceKey(userID uuid.UUID) string {
	return fmt.Sprintf("balance:%s", userID.String())
}

func (g *CacheKeyGenerator) EventKey(eventID uuid.UUID) string {
	return fmt.Sprintf("event:%s", eventID.String())
}

func (g *CacheKeyGenerator) AggregateEventsKey(aggregateID uuid.UUID) string {
	return fmt.Sprintf("aggregate_events:%s", aggregateID.String())
}

func (g *CacheKeyGenerator) EventTypeKey(eventType domain.EventType, limit, offset int) string {
	return fmt.Sprintf("event_type:%s:%d:%d", eventType, limit, offset)
}

func (g *CacheKeyGenerator) EventTimeRangeKey(startTime, endTime time.Time) string {
	return fmt.Sprintf("event_time_range:%d:%d", startTime.Unix(), endTime.Unix())
}

func (g *CacheKeyGenerator) EventStatisticsKey() string {
	return "event_statistics"
}

type CachePatternGenerator struct{}

func NewCachePatternGenerator() *CachePatternGenerator {
	return &CachePatternGenerator{}
}

func (g *CachePatternGenerator) UserPattern(userID uuid.UUID) string {
	return fmt.Sprintf("user:%s*", userID.String())
}

func (g *CachePatternGenerator) TransactionPattern(transactionID uuid.UUID) string {
	return fmt.Sprintf("transaction:%s*", transactionID.String())
}

func (g *CachePatternGenerator) UserTransactionsPattern(userID uuid.UUID) string {
	return fmt.Sprintf("user_transactions:%s*", userID.String())
}

func (g *CachePatternGenerator) BalancePattern(userID uuid.UUID) string {
	return fmt.Sprintf("balance:%s*", userID.String())
}

func (g *CachePatternGenerator) EventPattern(eventID uuid.UUID) string {
	return fmt.Sprintf("event:%s*", eventID.String())
}

func (g *CachePatternGenerator) AggregateEventsPattern(aggregateID uuid.UUID) string {
	return fmt.Sprintf("aggregate_events:%s*", aggregateID.String())
}

func (g *CachePatternGenerator) EventTypePattern(eventType domain.EventType) string {
	return fmt.Sprintf("event_type:%s*", eventType)
}

func (g *CachePatternGenerator) AllEventsPattern() string {
	return "event:*"
}

func (g *CachePatternGenerator) AllTransactionsPattern() string {
	return "transaction:*"
}

func (g *CachePatternGenerator) AllUsersPattern() string {
	return "user:*"
}

func (g *CachePatternGenerator) AllBalancesPattern() string {
	return "balance:*"
}
