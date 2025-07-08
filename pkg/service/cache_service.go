package service

import (
	"context"
	"time"

	"transaction-api-w-go/pkg/cache"
	"transaction-api-w-go/pkg/domain"
	"transaction-api-w-go/pkg/repository"

	"github.com/google/uuid"
)

// CacheService cache işlemlerini yöneten service
type CacheService struct {
	cache           *cache.RedisCache
	invalidator     *cache.CacheInvalidator
	warmuper        *cache.CacheWarmuper
	keyGen          *cache.CacheKeyGenerator
	userRepo        domain.UserRepository
	transactionRepo domain.TransactionRepository
	balanceRepo     domain.BalanceRepository
	logger          domain.Logger
}

func NewCacheService(
	redisCache *cache.RedisCache,
	userRepo domain.UserRepository,
	transactionRepo domain.TransactionRepository,
	balanceRepo domain.BalanceRepository,
	eventRepo *repository.EventRepository,
	logger domain.Logger,
) *CacheService {
	invalidator := cache.NewCacheInvalidator(redisCache, logger)
	warmuper := cache.NewCacheWarmuper(redisCache, userRepo, transactionRepo, balanceRepo, eventRepo, logger)

	return &CacheService{
		cache:           redisCache,
		invalidator:     invalidator,
		warmuper:        warmuper,
		keyGen:          cache.NewCacheKeyGenerator(),
		userRepo:        userRepo,
		transactionRepo: transactionRepo,
		balanceRepo:     balanceRepo,
		logger:          logger,
	}
}

func (s *CacheService) GetUser(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	key := s.keyGen.UserKey(userID)
	var user domain.User

	err := s.cache.Get(ctx, key, &user)
	if err == nil {
		s.logger.Debug("User found in cache", "user_id", userID)
		return &user, nil
	}

	if err != domain.ErrCacheMiss {
		s.logger.Error("Cache error", "error", err)
	}

	userFromDB, err := s.userRepo.GetByID(ctx, uint(userID.ID()))
	if err != nil {
		return nil, err
	}

	if err := s.cache.Set(ctx, key, userFromDB, 30*time.Minute); err != nil {
		s.logger.Error("Failed to cache user", "error", err)
	}

	return userFromDB, nil
}

func (s *CacheService) GetTransaction(ctx context.Context, transactionID uuid.UUID) (*domain.Transaction, error) {
	key := s.keyGen.TransactionKey(transactionID)
	var transaction domain.Transaction

	err := s.cache.Get(ctx, key, &transaction)
	if err == nil {
		s.logger.Debug("Transaction found in cache", "transaction_id", transactionID)
		return &transaction, nil
	}

	if err != domain.ErrCacheMiss {
		s.logger.Error("Cache error", "error", err)
	}

	transactionFromDB, err := s.transactionRepo.GetByID(ctx, uint(transactionID.ID()))
	if err != nil {
		return nil, err
	}

	if err := s.cache.Set(ctx, key, transactionFromDB, 30*time.Minute); err != nil {
		s.logger.Error("Failed to cache transaction", "error", err)
	}

	return transactionFromDB, nil
}

func (s *CacheService) GetBalance(ctx context.Context, userID uuid.UUID) (*domain.Balance, error) {
	key := s.keyGen.BalanceKey(userID)
	var balance domain.Balance

	err := s.cache.Get(ctx, key, &balance)
	if err == nil {
		s.logger.Debug("Balance found in cache", "user_id", userID)
		return &balance, nil
	}

	if err != domain.ErrCacheMiss {
		s.logger.Error("Cache error", "error", err)
	}

	balanceFromDB, err := s.balanceRepo.GetByUserID(ctx, uint(userID.ID()))
	if err != nil {
		return nil, err
	}

	if err := s.cache.Set(ctx, key, balanceFromDB, 15*time.Minute); err != nil {
		s.logger.Error("Failed to cache balance", "error", err)
	}

	return balanceFromDB, nil
}

func (s *CacheService) GetUserTransactions(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Transaction, error) {
	key := s.keyGen.UserTransactionsKey(userID, limit, offset)
	var transactions []*domain.Transaction

	err := s.cache.Get(ctx, key, &transactions)
	if err == nil {
		s.logger.Debug("User transactions found in cache", "user_id", userID)
		return transactions, nil
	}

	if err != domain.ErrCacheMiss {
		s.logger.Error("Cache error", "error", err)
	}

	transactionsFromDB, err := s.transactionRepo.GetByUserID(ctx, uint(userID.ID()))
	if err != nil {
		return nil, err
	}

	start := offset
	end := start + limit
	if end > len(transactionsFromDB) {
		end = len(transactionsFromDB)
	}
	if start > len(transactionsFromDB) {
		start = len(transactionsFromDB)
	}

	paginatedTransactions := transactionsFromDB[start:end]

	if err := s.cache.Set(ctx, key, paginatedTransactions, 10*time.Minute); err != nil {
		s.logger.Error("Failed to cache user transactions", "error", err)
	}

	return paginatedTransactions, nil
}

func (s *CacheService) GetAggregateEvents(ctx context.Context, aggregateID uuid.UUID) ([]domain.Event, error) {
	key := s.keyGen.AggregateEventsKey(aggregateID)
	var events []domain.Event

	err := s.cache.Get(ctx, key, &events)
	if err == nil {
		s.logger.Debug("Aggregate events found in cache", "aggregate_id", aggregateID)
		return events, nil
	}

	if err != domain.ErrCacheMiss {
		s.logger.Error("Cache error", "error", err)
	}

	events = []domain.Event{}

	if err := s.cache.Set(ctx, key, events, 5*time.Minute); err != nil {
		s.logger.Error("Failed to cache aggregate events", "error", err)
	}

	return events, nil
}

func (s *CacheService) SetUser(ctx context.Context, user *domain.User) error {
	key := s.keyGen.UserKey(user.ID)
	return s.cache.Set(ctx, key, user, 30*time.Minute)
}

func (s *CacheService) SetTransaction(ctx context.Context, transaction *domain.Transaction) error {
	key := s.keyGen.TransactionKey(transaction.ID)
	return s.cache.Set(ctx, key, transaction, 30*time.Minute)
}

func (s *CacheService) SetBalance(ctx context.Context, balance *domain.Balance) error {
	key := s.keyGen.BalanceKey(balance.UserID)
	return s.cache.Set(ctx, key, balance, 15*time.Minute)
}

func (s *CacheService) SetUserTransactions(ctx context.Context, userID uuid.UUID, transactions []*domain.Transaction, limit, offset int) error {
	key := s.keyGen.UserTransactionsKey(userID, limit, offset)
	return s.cache.Set(ctx, key, transactions, 10*time.Minute)
}

func (s *CacheService) SetAggregateEvents(ctx context.Context, aggregateID uuid.UUID, events []domain.Event) error {
	key := s.keyGen.AggregateEventsKey(aggregateID)
	return s.cache.Set(ctx, key, events, 5*time.Minute)
}

func (s *CacheService) InvalidateUser(ctx context.Context, userID uuid.UUID) error {
	return s.invalidator.InvalidateUser(ctx, userID)
}

func (s *CacheService) InvalidateTransaction(ctx context.Context, transactionID uuid.UUID) error {
	return s.invalidator.InvalidateTransaction(ctx, transactionID)
}

func (s *CacheService) InvalidateBalance(ctx context.Context, userID uuid.UUID) error {
	return s.invalidator.InvalidateBalance(ctx, userID)
}

func (s *CacheService) InvalidateAggregateEvents(ctx context.Context, aggregateID uuid.UUID) error {
	return s.invalidator.InvalidateAggregateEvents(ctx, aggregateID)
}

func (s *CacheService) WarmupUsers(ctx context.Context, userIDs []uuid.UUID) error {
	return s.warmuper.WarmupUsers(ctx, userIDs)
}

func (s *CacheService) WarmupTransactions(ctx context.Context, transactionIDs []uuid.UUID) error {
	return s.warmuper.WarmupTransactions(ctx, transactionIDs)
}

func (s *CacheService) WarmupBalances(ctx context.Context, userIDs []uuid.UUID) error {
	return s.warmuper.WarmupBalances(ctx, userIDs)
}

func (s *CacheService) WarmupAggregateEvents(ctx context.Context, aggregateIDs []uuid.UUID) error {
	return s.warmuper.WarmupAggregateEvents(ctx, aggregateIDs)
}

func (s *CacheService) GetCacheStats(ctx context.Context) (*cache.CacheStats, error) {
	return s.cache.GetStats(ctx)
}

func (s *CacheService) FlushAll(ctx context.Context) error {
	return s.cache.FlushAll(ctx)
}

func (s *CacheService) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	return s.cache.GetTTL(ctx, key)
}

func (s *CacheService) Exists(ctx context.Context, key string) (bool, error) {
	return s.cache.Exists(ctx, key)
}

func (s *CacheService) Increment(ctx context.Context, key string, value int64) (int64, error) {
	return s.cache.Increment(ctx, key, value)
}

func (s *CacheService) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	return s.cache.SetNX(ctx, key, value, expiration)
}

func (s *CacheService) Close() error {
	return s.cache.Close()
}
