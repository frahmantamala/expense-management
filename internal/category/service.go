package category

import (
	"log/slog"

	categoryDatamodel "github.com/frahmantamala/expense-management/internal/core/datamodel/category"
)

type RepositoryAPI interface {
	GetAll() ([]*categoryDatamodel.ExpenseCategory, error)
	GetByID(id int64) (*categoryDatamodel.ExpenseCategory, error)
	GetByName(name string) (*categoryDatamodel.ExpenseCategory, error)
	Create(category *categoryDatamodel.ExpenseCategory) error
	Update(category *categoryDatamodel.ExpenseCategory) error
	Delete(id int64) error
}

type Service struct {
	repo   RepositoryAPI
	logger *slog.Logger
}

func NewService(repo RepositoryAPI, logger *slog.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

func (s *Service) GetAllCategories() ([]CategoryResponse, error) {
	dataCategories, err := s.repo.GetAll()
	if err != nil {
		s.logger.Error("failed to get categories from repository", "error", err)
		return nil, err
	}

	var responses []CategoryResponse
	for _, dataCategory := range dataCategories {
		domainCategory := FromDataModel(dataCategory)
		if domainCategory.IsActiveCategory() {
			responses = append(responses, domainCategory.ToResponse())
		}
	}

	s.logger.Info("retrieved categories", "count", len(responses))
	return responses, nil
}

func (s *Service) GetCategoryByName(name string) (*CategoryResponse, error) {
	dataCategories, err := s.repo.GetAll()
	if err != nil {
		s.logger.Error("failed to get categories from repository", "error", err)
		return nil, err
	}

	for _, dataCategory := range dataCategories {
		if dataCategory.Name == name {
			domainCategory := FromDataModel(dataCategory)
			if domainCategory.IsActiveCategory() {
				response := domainCategory.ToResponse()
				return &response, nil
			}
		}
	}

	return nil, nil
}

func (s *Service) IsValidCategory(name string) bool {
	category, err := s.GetCategoryByName(name)
	if err != nil {
		s.logger.Warn("error checking category validity", "name", name, "error", err)
		return false
	}
	return category != nil
}
