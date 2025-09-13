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
	// First get all categories and find by name (since we don't have GetByName in RepositoryAPI)
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

// New methods matching the ServiceAPI interface
func (s *Service) GetAll() ([]*Category, error) {
	return nil, nil // Implementation needed
}

func (s *Service) GetByID(id int64) (*Category, error) {
	return nil, nil // Implementation needed
}

func (s *Service) Create(name, description string) (*Category, error) {
	newCategory := NewCategory(name, description)

	// Convert to datamodel for repository
	dataCategory := ToDataModel(newCategory)
	if err := s.repo.Create(dataCategory); err != nil {
		s.logger.Error("failed to create category", "error", err)
		return nil, err
	}

	// Update domain entity with generated ID
	newCategory.ID = dataCategory.ID
	newCategory.CreatedAt = dataCategory.CreatedAt
	newCategory.UpdatedAt = dataCategory.UpdatedAt

	return newCategory, nil
}

func (s *Service) Update(id int64, name, description string) (*Category, error) {
	// Get existing category
	dataCategory, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Convert to domain entity for business logic
	category := FromDataModel(dataCategory)
	category.Name = name
	category.Description = description

	// Convert back to datamodel for repository update
	updatedDataCategory := ToDataModel(category)
	if err := s.repo.Update(updatedDataCategory); err != nil {
		s.logger.Error("failed to update category", "error", err)
		return nil, err
	}

	return category, nil
}

func (s *Service) Delete(id int64) error {
	return s.repo.Delete(id)
}

func (s *Service) Activate(id int64) (*Category, error) {
	// Get existing category
	dataCategory, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Convert to domain entity for business logic
	category := FromDataModel(dataCategory)
	category.IsActive = true

	// Convert back to datamodel for repository update
	updatedDataCategory := ToDataModel(category)
	if err := s.repo.Update(updatedDataCategory); err != nil {
		s.logger.Error("failed to activate category", "error", err)
		return nil, err
	}

	return category, nil
}

func (s *Service) Deactivate(id int64) (*Category, error) {
	// Get existing category
	dataCategory, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Convert to domain entity for business logic
	category := FromDataModel(dataCategory)
	category.IsActive = false

	// Convert back to datamodel for repository update
	updatedDataCategory := ToDataModel(category)
	if err := s.repo.Update(updatedDataCategory); err != nil {
		s.logger.Error("failed to deactivate category", "error", err)
		return nil, err
	}

	return category, nil
}
