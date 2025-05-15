package service

import (
	"transaction-api-w-go/pkg/domain"
	"transaction-api-w-go/pkg/repository"
)

type UserService struct {
	userRepo *repository.UserRepository
}

func NewUserService(userRepo *repository.UserRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

func (s *UserService) List() ([]domain.User, error) {
	return s.userRepo.List()
}

func (s *UserService) GetByID(id string) (*domain.User, error) {
	return s.userRepo.GetByID(id)
}

func (s *UserService) Update(user *domain.User) error {
	return s.userRepo.Update(user)
}

func (s *UserService) Delete(id string) error {
	return s.userRepo.Delete(id)
}
