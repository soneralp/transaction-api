package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
	"transaction-api-w-go/pkg/domain"
	"transaction-api-w-go/pkg/worker"
)

type BasicBalanceService struct {
	balances map[uint]float64
	mu       sync.Mutex
}

func NewBasicBalanceService() *BasicBalanceService {
	return &BasicBalanceService{
		balances: make(map[uint]float64),
	}
}

func (s *BasicBalanceService) AddFunds(ctx context.Context, userID uint, amount float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.balances[userID] += amount
	log.Printf("Kullanıcı %d'ye %.2f TL eklendi. Yeni bakiye: %.2f TL", userID, amount, s.balances[userID])
	return nil
}

func (s *BasicBalanceService) WithdrawFunds(ctx context.Context, userID uint, amount float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.balances[userID] < amount {
		return domain.ErrInsufficientFunds
	}

	s.balances[userID] -= amount
	log.Printf("Kullanıcı %d'den %.2f TL çekildi. Yeni bakiye: %.2f TL", userID, amount, s.balances[userID])
	return nil
}

func (s *BasicBalanceService) GetBalance(ctx context.Context, userID uint) (*domain.Balance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	amount, exists := s.balances[userID]
	if !exists {
		amount = 0
	}

	return &domain.Balance{
		UserID:    userID,
		Amount:    amount,
		UpdatedAt: time.Now(),
	}, nil
}

func (s *BasicBalanceService) TransferFunds(ctx context.Context, fromUserID uint, toUserID uint, amount float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.balances[fromUserID] < amount {
		return domain.ErrInsufficientFunds
	}

	s.balances[fromUserID] -= amount
	s.balances[toUserID] += amount

	log.Printf("Kullanıcı %d'den kullanıcı %d'ye %.2f TL transfer edildi", fromUserID, toUserID, amount)
	log.Printf("Kullanıcı %d'nin yeni bakiyesi: %.2f TL", fromUserID, s.balances[fromUserID])
	log.Printf("Kullanıcı %d'nin yeni bakiyesi: %.2f TL", toUserID, s.balances[toUserID])

	return nil
}

func main() {
	balanceService := NewBasicBalanceService()

	processor := worker.NewBatchProcessor(balanceService)
	processor.Start()

	processor.SubmitJob(worker.BatchJob{
		UserIDs:     []uint{1, 2, 3},
		Amount:      100.0,
		Operation:   "add",
		Description: "Toplu para ekleme",
	})

	processor.SubmitJob(worker.BatchJob{
		UserIDs:     []uint{1, 2},
		Amount:      50.0,
		Operation:   "withdraw",
		Description: "Toplu para çekme",
	})

	processor.SubmitJob(worker.BatchJob{
		UserIDs:     []uint{1, 2},
		Amount:      25.0,
		Operation:   "transfer",
		Description: "Transfer işlemi",
	})

	time.Sleep(1 * time.Second)

	stats := processor.GetStats()
	fmt.Println("\nİşlem İstatistikleri:")
	fmt.Printf("Toplam İşlenen: %d\n", stats.TotalProcessed)
	fmt.Printf("Toplam Başarısız: %d\n", stats.TotalFailed)
	fmt.Printf("Toplam Miktar: %.2f TL\n", stats.TotalAmount)
	fmt.Printf("Ortalama İşlem Süresi: %.2f saniye\n", stats.AverageProcessTime)

	fmt.Println("\nGüncel Bakiyeler:")
	for userID, amount := range balanceService.balances {
		fmt.Printf("Kullanıcı %d: %.2f TL\n", userID, amount)
	}

	processor.Stop()
}
