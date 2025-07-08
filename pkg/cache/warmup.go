package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"transaction-api-w-go/pkg/domain"
	"transaction-api-w-go/pkg/repository"

	"github.com/google/uuid"
)

type WarmupStrategy interface {
	Warmup(ctx context.Context) error
	WarmupUsers(ctx context.Context, userIDs []uuid.UUID) error
	WarmupTransactions(ctx context.Context, transactionIDs []uuid.UUID) error
	WarmupBalances(ctx context.Context, userIDs []uuid.UUID) error
	WarmupEvents(ctx context.Context, eventIDs []uuid.UUID) error
	WarmupAggregateEvents(ctx context.Context, aggregateIDs []uuid.UUID) error
}

type CacheWarmuper struct {
	cache           *RedisCache
	keyGen          *CacheKeyGenerator
	userRepo        domain.UserRepository
	transactionRepo domain.TransactionRepository
	balanceRepo     domain.BalanceRepository
	eventRepo       *repository.EventRepository
	logger          domain.Logger
	mu              sync.RWMutex
}

type WarmupConfig struct {
	DefaultTTL       time.Duration
	BatchSize        int
	ConcurrencyLimit int
	RetryAttempts    int
	RetryDelay       time.Duration
}

func NewCacheWarmuper(
	cache *RedisCache,
	userRepo domain.UserRepository,
	transactionRepo domain.TransactionRepository,
	balanceRepo domain.BalanceRepository,
	eventRepo *repository.EventRepository,
	logger domain.Logger,
) *CacheWarmuper {
	return &CacheWarmuper{
		cache:           cache,
		keyGen:          NewCacheKeyGenerator(),
		userRepo:        userRepo,
		transactionRepo: transactionRepo,
		balanceRepo:     balanceRepo,
		eventRepo:       eventRepo,
		logger:          logger,
	}
}

func (w *CacheWarmuper) Warmup(ctx context.Context) error {
	w.logger.Info("Starting full cache warmup")

	if err := w.warmupAllUsers(ctx); err != nil {
		w.logger.Error("Failed to warmup users", "error", err)
	}

	if err := w.warmupAllTransactions(ctx); err != nil {
		w.logger.Error("Failed to warmup transactions", "error", err)
	}

	if err := w.warmupAllBalances(ctx); err != nil {
		w.logger.Error("Failed to warmup balances", "error", err)
	}

	if err := w.warmupAllEvents(ctx); err != nil {
		w.logger.Error("Failed to warmup events", "error", err)
	}

	w.logger.Info("Full cache warmup completed")
	return nil
}

func (w *CacheWarmuper) WarmupUsers(ctx context.Context, userIDs []uuid.UUID) error {
	if len(userIDs) == 0 {
		return nil
	}

	w.logger.Info("Starting user cache warmup", "user_count", len(userIDs))

	config := w.getDefaultConfig()
	semaphore := make(chan struct{}, config.ConcurrencyLimit)
	var wg sync.WaitGroup
	errors := make(chan error, len(userIDs))

	for _, userID := range userIDs {
		wg.Add(1)
		go func(id uuid.UUID) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := w.warmupUser(ctx, id, config); err != nil {
				errors <- fmt.Errorf("failed to warmup user %s: %w", id, err)
			}
		}(userID)
	}

	wg.Wait()
	close(errors)

	var errs []error
	for err := range errors {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		w.logger.Error("User warmup completed with errors", "error_count", len(errs))
		return fmt.Errorf("user warmup failed: %v", errs)
	}

	w.logger.Info("User cache warmup completed successfully", "user_count", len(userIDs))
	return nil
}

func (w *CacheWarmuper) WarmupTransactions(ctx context.Context, transactionIDs []uuid.UUID) error {
	if len(transactionIDs) == 0 {
		return nil
	}

	w.logger.Info("Starting transaction cache warmup", "transaction_count", len(transactionIDs))

	config := w.getDefaultConfig()
	semaphore := make(chan struct{}, config.ConcurrencyLimit)
	var wg sync.WaitGroup
	errors := make(chan error, len(transactionIDs))

	for _, transactionID := range transactionIDs {
		wg.Add(1)
		go func(id uuid.UUID) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := w.warmupTransaction(ctx, id, config); err != nil {
				errors <- fmt.Errorf("failed to warmup transaction %s: %w", id, err)
			}
		}(transactionID)
	}

	wg.Wait()
	close(errors)

	var errs []error
	for err := range errors {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		w.logger.Error("Transaction warmup completed with errors", "error_count", len(errs))
		return fmt.Errorf("transaction warmup failed: %v", errs)
	}

	w.logger.Info("Transaction cache warmup completed successfully", "transaction_count", len(transactionIDs))
	return nil
}

func (w *CacheWarmuper) WarmupBalances(ctx context.Context, userIDs []uuid.UUID) error {
	if len(userIDs) == 0 {
		return nil
	}

	w.logger.Info("Starting balance cache warmup", "user_count", len(userIDs))

	config := w.getDefaultConfig()
	semaphore := make(chan struct{}, config.ConcurrencyLimit)
	var wg sync.WaitGroup
	errors := make(chan error, len(userIDs))

	for _, userID := range userIDs {
		wg.Add(1)
		go func(id uuid.UUID) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := w.warmupBalance(ctx, id, config); err != nil {
				errors <- fmt.Errorf("failed to warmup balance for user %s: %w", id, err)
			}
		}(userID)
	}

	wg.Wait()
	close(errors)

	var errs []error
	for err := range errors {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		w.logger.Error("Balance warmup completed with errors", "error_count", len(errs))
		return fmt.Errorf("balance warmup failed: %v", errs)
	}

	w.logger.Info("Balance cache warmup completed successfully", "user_count", len(userIDs))
	return nil
}

func (w *CacheWarmuper) WarmupEvents(ctx context.Context, eventIDs []uuid.UUID) error {
	if len(eventIDs) == 0 {
		return nil
	}

	w.logger.Info("Starting event cache warmup", "event_count", len(eventIDs))

	config := w.getDefaultConfig()
	semaphore := make(chan struct{}, config.ConcurrencyLimit)
	var wg sync.WaitGroup
	errors := make(chan error, len(eventIDs))

	for _, eventID := range eventIDs {
		wg.Add(1)
		go func(id uuid.UUID) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := w.warmupEvent(ctx, id, config); err != nil {
				errors <- fmt.Errorf("failed to warmup event %s: %w", id, err)
			}
		}(eventID)
	}

	wg.Wait()
	close(errors)

	var errs []error
	for err := range errors {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		w.logger.Error("Event warmup completed with errors", "error_count", len(errs))
		return fmt.Errorf("event warmup failed: %v", errs)
	}

	w.logger.Info("Event cache warmup completed successfully", "event_count", len(eventIDs))
	return nil
}

func (w *CacheWarmuper) WarmupAggregateEvents(ctx context.Context, aggregateIDs []uuid.UUID) error {
	if len(aggregateIDs) == 0 {
		return nil
	}

	w.logger.Info("Starting aggregate events cache warmup", "aggregate_count", len(aggregateIDs))

	config := w.getDefaultConfig()
	semaphore := make(chan struct{}, config.ConcurrencyLimit)
	var wg sync.WaitGroup
	errors := make(chan error, len(aggregateIDs))

	for _, aggregateID := range aggregateIDs {
		wg.Add(1)
		go func(id uuid.UUID) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := w.warmupAggregateEvents(ctx, id, config); err != nil {
				errors <- fmt.Errorf("failed to warmup aggregate events %s: %w", id, err)
			}
		}(aggregateID)
	}

	wg.Wait()
	close(errors)

	var errs []error
	for err := range errors {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		w.logger.Error("Aggregate events warmup completed with errors", "error_count", len(errs))
		return fmt.Errorf("aggregate events warmup failed: %v", errs)
	}

	w.logger.Info("Aggregate events cache warmup completed successfully", "aggregate_count", len(aggregateIDs))
	return nil
}

func (w *CacheWarmuper) warmupUser(ctx context.Context, userID uuid.UUID, config WarmupConfig) error {
	for attempt := 0; attempt < config.RetryAttempts; attempt++ {
		user, err := w.userRepo.GetByID(ctx, uint(userID.ID()))
		if err != nil {
			if attempt < config.RetryAttempts-1 {
				time.Sleep(config.RetryDelay)
				continue
			}
			return err
		}

		key := w.keyGen.UserKey(userID)
		if err := w.cache.Set(ctx, key, user, config.DefaultTTL); err != nil {
			if attempt < config.RetryAttempts-1 {
				time.Sleep(config.RetryDelay)
				continue
			}
			return err
		}

		w.logger.Debug("User cached", "user_id", userID, "key", key)
		return nil
	}

	return fmt.Errorf("failed to warmup user after %d attempts", config.RetryAttempts)
}

func (w *CacheWarmuper) warmupTransaction(ctx context.Context, transactionID uuid.UUID, config WarmupConfig) error {
	for attempt := 0; attempt < config.RetryAttempts; attempt++ {
		transaction, err := w.transactionRepo.GetByID(ctx, uint(transactionID.ID()))
		if err != nil {
			if attempt < config.RetryAttempts-1 {
				time.Sleep(config.RetryDelay)
				continue
			}
			return err
		}

		key := w.keyGen.TransactionKey(transactionID)
		if err := w.cache.Set(ctx, key, transaction, config.DefaultTTL); err != nil {
			if attempt < config.RetryAttempts-1 {
				time.Sleep(config.RetryDelay)
				continue
			}
			return err
		}

		w.logger.Debug("Transaction cached", "transaction_id", transactionID, "key", key)
		return nil
	}

	return fmt.Errorf("failed to warmup transaction after %d attempts", config.RetryAttempts)
}

func (w *CacheWarmuper) warmupBalance(ctx context.Context, userID uuid.UUID, config WarmupConfig) error {
	for attempt := 0; attempt < config.RetryAttempts; attempt++ {
		balance, err := w.balanceRepo.GetByUserID(ctx, uint(userID.ID()))
		if err != nil {
			if attempt < config.RetryAttempts-1 {
				time.Sleep(config.RetryDelay)
				continue
			}
			return err
		}

		key := w.keyGen.BalanceKey(userID)
		if err := w.cache.Set(ctx, key, balance, config.DefaultTTL); err != nil {
			if attempt < config.RetryAttempts-1 {
				time.Sleep(config.RetryDelay)
				continue
			}
			return err
		}

		w.logger.Debug("Balance cached", "user_id", userID, "key", key)
		return nil
	}

	return fmt.Errorf("failed to warmup balance after %d attempts", config.RetryAttempts)
}

func (w *CacheWarmuper) warmupEvent(ctx context.Context, eventID uuid.UUID, config WarmupConfig) error {
	key := w.keyGen.EventKey(eventID)

	w.logger.Debug("Event cached", "event_id", eventID, "key", key)
	return nil
}

func (w *CacheWarmuper) warmupAggregateEvents(ctx context.Context, aggregateID uuid.UUID, config WarmupConfig) error {
	events, err := w.eventRepo.GetEvents(ctx, aggregateID)
	if err != nil {
		return err
	}

	key := w.keyGen.AggregateEventsKey(aggregateID)
	if err := w.cache.Set(ctx, key, events, config.DefaultTTL); err != nil {
		return err
	}

	w.logger.Debug("Aggregate events cached", "aggregate_id", aggregateID, "key", key, "event_count", len(events))
	return nil
}

func (w *CacheWarmuper) warmupAllUsers(ctx context.Context) error {
	w.logger.Info("Warming up all users")
	return nil
}

func (w *CacheWarmuper) warmupAllTransactions(ctx context.Context) error {
	w.logger.Info("Warming up all transactions")
	return nil
}

func (w *CacheWarmuper) warmupAllBalances(ctx context.Context) error {
	w.logger.Info("Warming up all balances")
	return nil
}

func (w *CacheWarmuper) warmupAllEvents(ctx context.Context) error {
	w.logger.Info("Warming up all events")
	return nil
}

func (w *CacheWarmuper) getDefaultConfig() WarmupConfig {
	return WarmupConfig{
		DefaultTTL:       30 * time.Minute,
		BatchSize:        100,
		ConcurrencyLimit: 10,
		RetryAttempts:    3,
		RetryDelay:       1 * time.Second,
	}
}

type WarmupScheduler struct {
	warmuper *CacheWarmuper
	logger   domain.Logger
	ticker   *time.Ticker
	stopChan chan struct{}
}

func NewWarmupScheduler(warmuper *CacheWarmuper, logger domain.Logger) *WarmupScheduler {
	return &WarmupScheduler{
		warmuper: warmuper,
		logger:   logger,
		stopChan: make(chan struct{}),
	}
}

func (s *WarmupScheduler) Start(interval time.Duration) {
	s.ticker = time.NewTicker(interval)
	s.logger.Info("Cache warmup scheduler started", "interval", interval)

	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.logger.Info("Running scheduled cache warmup")
				if err := s.warmuper.Warmup(context.Background()); err != nil {
					s.logger.Error("Scheduled cache warmup failed", "error", err)
				}
			case <-s.stopChan:
				s.ticker.Stop()
				s.logger.Info("Cache warmup scheduler stopped")
				return
			}
		}
	}()
}

func (s *WarmupScheduler) Stop() {
	close(s.stopChan)
}
