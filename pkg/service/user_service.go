package service

import (
	"context"
	"transaction-api-w-go/pkg/domain"
)

type userService struct {
	userRepo domain.UserRepository
}

func NewUserService(userRepo domain.UserRepository) domain.UserService {
	return &userService{
		userRepo: userRepo,
	}
}

func (s *userService) Register(ctx context.Context, username, email, password string) (*domain.User, error) {
	existingUser, err := s.userRepo.GetByEmail(ctx, email)
	if err == nil && existingUser != nil {
		return nil, domain.ErrUserAlreadyExists
	}

	user, err := domain.NewUser(username, email, password)
	if err != nil {
		return nil, err
	}

	err = s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *userService) GetByID(ctx context.Context, id uint) (*domain.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

func (s *userService) Update(ctx context.Context, user *domain.User) error {
	existingUser, err := s.userRepo.GetByID(ctx, user.ID)
	if err != nil {
		return err
	}

	if existingUser.Email != user.Email {
		emailUser, err := s.userRepo.GetByEmail(ctx, user.Email)
		if err == nil && emailUser != nil && emailUser.ID != user.ID {
			return domain.ErrUserAlreadyExists
		}
	}

	return s.userRepo.Update(ctx, user)
}

func (s *userService) Delete(ctx context.Context, id uint) error {
	return s.userRepo.Delete(ctx, id)
}
