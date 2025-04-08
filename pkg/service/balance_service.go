package service

import (
	"context"
	"transaction-api-w-go/pkg/domain"
)

type balanceService struct {
	balanceRepo domain.BalanceRepository
}

func NewBalanceService(balanceRepo domain.BalanceRepository) domain.BalanceService {
	return &balanceService{
		balanceRepo: balanceRepo,
	}
}

func (s *balanceService) GetBalance(ctx context.Context, userID uint) (*domain.Balance, error) {
	return s.balanceRepo.GetByUserID(ctx, userID)
}

func (s *balanceService) AddFunds(ctx context.Context, userID uint, amount float64) error {
	balance, err := s.balanceRepo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}

	err = balance.Add(amount)
	if err != nil {
		return err
	}

	return s.balanceRepo.Update(ctx, balance)
}

func (s *balanceService) WithdrawFunds(ctx context.Context, userID uint, amount float64) error {
	balance, err := s.balanceRepo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}

	err = balance.Subtract(amount)
	if err != nil {
		return err
	}

	return s.balanceRepo.Update(ctx, balance)
}

func (s *balanceService) TransferFunds(ctx context.Context, fromUserID, toUserID uint, amount float64) error {
	fromBalance, err := s.balanceRepo.GetByUserID(ctx, fromUserID)
	if err != nil {
		return err
	}

	toBalance, err := s.balanceRepo.GetByUserID(ctx, toUserID)
	if err != nil {
		return err
	}

	err = fromBalance.Subtract(amount)
	if err != nil {
		return err
	}

	err = toBalance.Add(amount)
	if err != nil {
		return err
	}

	err = s.balanceRepo.Update(ctx, fromBalance)
	if err != nil {
		return err
	}

	return s.balanceRepo.Update(ctx, toBalance)
}
