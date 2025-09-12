package category

import (
	"log/slog"
)

type Repository interface {
	GetAll() ([]*Category, error)
	GetByName(name string) (*Category, error)
	Create(category *Category) error
	Update(category *Category) error
	Delete(id int64) error
}

type Service struct {
	repo   Repository
	logger *slog.Logger
}

func NewService(repo Repository, logger *slog.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

func (s *Service) GetAllCategories() ([]CategoryResponse, error) {
	categories, err := s.repo.GetAll()
	if err != nil {
		s.logger.Error("failed to get categories from repository", "error", err)
		return nil, err
	}

	var responses []CategoryResponse
	for _, category := range categories {
		if category.IsActive {
			responses = append(responses, category.ToResponse())
		}
	}

	s.logger.Info("retrieved categories", "count", len(responses))
	return responses, nil
}

func (s *Service) GetCategoryByName(name string) (*CategoryResponse, error) {
	category, err := s.repo.GetByName(name)
	if err != nil {
		s.logger.Error("failed to get category by name", "name", name, "error", err)
		return nil, err
	}

	if category == nil || !category.IsActive {
		return nil, nil
	}

	response := category.ToResponse()
	return &response, nil
}

func (s *Service) IsValidCategory(name string) bool {
	category, err := s.GetCategoryByName(name)
	if err != nil {
		s.logger.Warn("error checking category validity", "name", name, "error", err)
		return false
	}
	return category != nil
}
