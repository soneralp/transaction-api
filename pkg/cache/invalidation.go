package cache

import (
	"context"
	"fmt"
	"sync"

	"transaction-api-w-go/pkg/domain"

	"github.com/google/uuid"
)

type InvalidationStrategy interface {
	Invalidate(ctx context.Context, keys ...string) error
	InvalidatePattern(ctx context.Context, patterns ...string) error
	InvalidateUser(ctx context.Context, userID uuid.UUID) error
	InvalidateTransaction(ctx context.Context, transactionID uuid.UUID) error
	InvalidateBalance(ctx context.Context, userID uuid.UUID) error
	InvalidateEvent(ctx context.Context, eventID uuid.UUID) error
	InvalidateAggregateEvents(ctx context.Context, aggregateID uuid.UUID) error
}

type CacheInvalidator struct {
	cache      *RedisCache
	keyGen     *CacheKeyGenerator
	patternGen *CachePatternGenerator
	logger     domain.Logger
	mu         sync.RWMutex
}

func NewCacheInvalidator(cache *RedisCache, logger domain.Logger) *CacheInvalidator {
	return &CacheInvalidator{
		cache:      cache,
		keyGen:     NewCacheKeyGenerator(),
		patternGen: NewCachePatternGenerator(),
		logger:     logger,
	}
}

func (i *CacheInvalidator) Invalidate(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	for _, key := range keys {
		if err := i.cache.Delete(ctx, key); err != nil {
			i.logger.Error("Failed to invalidate cache key", "key", key, "error", err)
			continue
		}
	}

	i.logger.Info("Cache invalidated", "keys_count", len(keys))
	return nil
}

func (i *CacheInvalidator) InvalidatePattern(ctx context.Context, patterns ...string) error {
	if len(patterns) == 0 {
		return nil
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	for _, pattern := range patterns {
		if err := i.cache.DeletePattern(ctx, pattern); err != nil {
			i.logger.Error("Failed to invalidate cache pattern", "pattern", pattern, "error", err)
			continue
		}
	}

	i.logger.Info("Cache pattern invalidated", "patterns_count", len(patterns))
	return nil
}

func (i *CacheInvalidator) InvalidateUser(ctx context.Context, userID uuid.UUID) error {
	patterns := []string{
		i.patternGen.UserPattern(userID),
		i.patternGen.UserTransactionsPattern(userID),
		i.patternGen.BalancePattern(userID),
	}

	return i.InvalidatePattern(ctx, patterns...)
}

func (i *CacheInvalidator) InvalidateTransaction(ctx context.Context, transactionID uuid.UUID) error {
	patterns := []string{
		i.patternGen.TransactionPattern(transactionID),
	}

	return i.InvalidatePattern(ctx, patterns...)
}

func (i *CacheInvalidator) InvalidateBalance(ctx context.Context, userID uuid.UUID) error {
	patterns := []string{
		i.patternGen.BalancePattern(userID),
	}

	return i.InvalidatePattern(ctx, patterns...)
}

func (i *CacheInvalidator) InvalidateEvent(ctx context.Context, eventID uuid.UUID) error {
	patterns := []string{
		i.patternGen.EventPattern(eventID),
	}

	return i.InvalidatePattern(ctx, patterns...)
}

func (i *CacheInvalidator) InvalidateAggregateEvents(ctx context.Context, aggregateID uuid.UUID) error {
	patterns := []string{
		i.patternGen.AggregateEventsPattern(aggregateID),
	}

	return i.InvalidatePattern(ctx, patterns...)
}

func (i *CacheInvalidator) InvalidateAllEvents(ctx context.Context) error {
	patterns := []string{
		i.patternGen.AllEventsPattern(),
	}

	return i.InvalidatePattern(ctx, patterns...)
}

func (i *CacheInvalidator) InvalidateAllTransactions(ctx context.Context) error {
	patterns := []string{
		i.patternGen.AllTransactionsPattern(),
	}

	return i.InvalidatePattern(ctx, patterns...)
}

func (i *CacheInvalidator) InvalidateAllUsers(ctx context.Context) error {
	patterns := []string{
		i.patternGen.AllUsersPattern(),
	}

	return i.InvalidatePattern(ctx, patterns...)
}

func (i *CacheInvalidator) InvalidateAllBalances(ctx context.Context) error {
	patterns := []string{
		i.patternGen.AllBalancesPattern(),
	}

	return i.InvalidatePattern(ctx, patterns...)
}

func (i *CacheInvalidator) InvalidateAll(ctx context.Context) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if err := i.cache.FlushAll(ctx); err != nil {
		return fmt.Errorf("failed to invalidate all cache: %w", err)
	}

	i.logger.Info("All cache invalidated")
	return nil
}

type InvalidationRule struct {
	EntityType string
	EntityID   uuid.UUID
	Patterns   []string
	Keys       []string
}

type InvalidationRuleBuilder struct {
	rules []InvalidationRule
}

func NewInvalidationRuleBuilder() *InvalidationRuleBuilder {
	return &InvalidationRuleBuilder{
		rules: make([]InvalidationRule, 0),
	}
}

func (b *InvalidationRuleBuilder) AddUserRule(userID uuid.UUID) *InvalidationRuleBuilder {
	patternGen := NewCachePatternGenerator()

	b.rules = append(b.rules, InvalidationRule{
		EntityType: "user",
		EntityID:   userID,
		Patterns: []string{
			patternGen.UserPattern(userID),
			patternGen.UserTransactionsPattern(userID),
			patternGen.BalancePattern(userID),
		},
	})

	return b
}

func (b *InvalidationRuleBuilder) AddTransactionRule(transactionID uuid.UUID) *InvalidationRuleBuilder {
	patternGen := NewCachePatternGenerator()

	b.rules = append(b.rules, InvalidationRule{
		EntityType: "transaction",
		EntityID:   transactionID,
		Patterns: []string{
			patternGen.TransactionPattern(transactionID),
		},
	})

	return b
}

func (b *InvalidationRuleBuilder) AddBalanceRule(userID uuid.UUID) *InvalidationRuleBuilder {
	patternGen := NewCachePatternGenerator()

	b.rules = append(b.rules, InvalidationRule{
		EntityType: "balance",
		EntityID:   userID,
		Patterns: []string{
			patternGen.BalancePattern(userID),
		},
	})

	return b
}

func (b *InvalidationRuleBuilder) AddEventRule(eventID uuid.UUID) *InvalidationRuleBuilder {
	patternGen := NewCachePatternGenerator()

	b.rules = append(b.rules, InvalidationRule{
		EntityType: "event",
		EntityID:   eventID,
		Patterns: []string{
			patternGen.EventPattern(eventID),
		},
	})

	return b
}

func (b *InvalidationRuleBuilder) AddAggregateEventsRule(aggregateID uuid.UUID) *InvalidationRuleBuilder {
	patternGen := NewCachePatternGenerator()

	b.rules = append(b.rules, InvalidationRule{
		EntityType: "aggregate_events",
		EntityID:   aggregateID,
		Patterns: []string{
			patternGen.AggregateEventsPattern(aggregateID),
		},
	})

	return b
}

func (b *InvalidationRuleBuilder) AddCustomRule(entityType string, entityID uuid.UUID, patterns []string, keys []string) *InvalidationRuleBuilder {
	b.rules = append(b.rules, InvalidationRule{
		EntityType: entityType,
		EntityID:   entityID,
		Patterns:   patterns,
		Keys:       keys,
	})

	return b
}

func (b *InvalidationRuleBuilder) Build() []InvalidationRule {
	return b.rules
}

type BatchInvalidator struct {
	invalidator *CacheInvalidator
	logger      domain.Logger
}

func NewBatchInvalidator(invalidator *CacheInvalidator, logger domain.Logger) *BatchInvalidator {
	return &BatchInvalidator{
		invalidator: invalidator,
		logger:      logger,
	}
}

func (b *BatchInvalidator) InvalidateBatch(ctx context.Context, rules []InvalidationRule) error {
	if len(rules) == 0 {
		return nil
	}

	b.logger.Info("Starting batch cache invalidation", "rules_count", len(rules))

	var allPatterns []string
	var allKeys []string

	for _, rule := range rules {
		allPatterns = append(allPatterns, rule.Patterns...)
		allKeys = append(allKeys, rule.Keys...)
	}

	if len(allPatterns) > 0 {
		if err := b.invalidator.InvalidatePattern(ctx, allPatterns...); err != nil {
			b.logger.Error("Failed to invalidate patterns in batch", "error", err)
		}
	}

	if len(allKeys) > 0 {
		if err := b.invalidator.Invalidate(ctx, allKeys...); err != nil {
			b.logger.Error("Failed to invalidate keys in batch", "error", err)
		}
	}

	b.logger.Info("Batch cache invalidation completed",
		"rules_count", len(rules),
		"patterns_count", len(allPatterns),
		"keys_count", len(allKeys))

	return nil
}

func (b *BatchInvalidator) InvalidateByEntityType(ctx context.Context, entityType string, entityIDs []uuid.UUID) error {
	if len(entityIDs) == 0 {
		return nil
	}

	builder := NewInvalidationRuleBuilder()

	switch entityType {
	case "user":
		for _, userID := range entityIDs {
			builder.AddUserRule(userID)
		}
	case "transaction":
		for _, transactionID := range entityIDs {
			builder.AddTransactionRule(transactionID)
		}
	case "balance":
		for _, userID := range entityIDs {
			builder.AddBalanceRule(userID)
		}
	case "event":
		for _, eventID := range entityIDs {
			builder.AddEventRule(eventID)
		}
	case "aggregate_events":
		for _, aggregateID := range entityIDs {
			builder.AddAggregateEventsRule(aggregateID)
		}
	default:
		return fmt.Errorf("unknown entity type: %s", entityType)
	}

	rules := builder.Build()
	return b.InvalidateBatch(ctx, rules)
}
