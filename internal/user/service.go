package user

import (
	"fmt"

	userDatamodel "github.com/frahmantamala/expense-management/internal/core/datamodel/user"
)

type RepositoryAPI interface {
	GetByID(userID int64) (*userDatamodel.User, error)
	GetPermissions(userID int64) ([]string, error)
}

type Service struct {
	repo RepositoryAPI
}

func NewService(repo RepositoryAPI) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) GetByID(userID int64) (*User, error) {
	dataUser, err := s.repo.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	// Get permissions separately
	permissions, err := s.repo.GetPermissions(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}

	// Convert datamodel to domain entity with permissions
	return FromDataModelWithPermissions(dataUser, permissions), nil
}

func (s *Service) GetPermissions(userID int64) ([]string, error) {
	return s.repo.GetPermissions(userID)
}
