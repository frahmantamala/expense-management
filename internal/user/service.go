package user

import (
	"fmt"
)

type Service struct {
	repo Repository
}

type Repository interface {
	GetByID(userID int64) (*User, error)
	GetPermissions(userID int64) ([]string, error)
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) GetByID(userID int64) (*User, error) {
	u, err := s.repo.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	perms, err := s.repo.GetPermissions(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}
	u.Permissions = perms

	return u, nil
}
