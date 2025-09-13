package category_test

import (
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/frahmantamala/expense-management/internal/category"
	categoryDatamodel "github.com/frahmantamala/expense-management/internal/core/datamodel/category"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCategoryService(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Category Service Suite")
}

// MockRepository implements category.Repository for testing
type MockRepository struct {
	categories map[string]*categoryDatamodel.ExpenseCategory
	shouldFail bool
	failError  error
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		categories: make(map[string]*categoryDatamodel.ExpenseCategory),
		shouldFail: false,
	}
}

func (m *MockRepository) GetAll() ([]*categoryDatamodel.ExpenseCategory, error) {
	if m.shouldFail {
		return nil, m.failError
	}

	var result []*categoryDatamodel.ExpenseCategory
	for _, cat := range m.categories {
		result = append(result, cat)
	}
	return result, nil
}

func (m *MockRepository) GetByName(name string) (*categoryDatamodel.ExpenseCategory, error) {
	if m.shouldFail {
		return nil, m.failError
	}

	cat, exists := m.categories[name]
	if !exists {
		return nil, nil
	}
	return cat, nil
}

func (m *MockRepository) Create(cat *categoryDatamodel.ExpenseCategory) error {
	if m.shouldFail {
		return m.failError
	}
	m.categories[cat.Name] = cat
	return nil
}

func (m *MockRepository) Update(cat *categoryDatamodel.ExpenseCategory) error {
	if m.shouldFail {
		return m.failError
	}
	m.categories[cat.Name] = cat
	return nil
}

func (m *MockRepository) GetByID(id int64) (*categoryDatamodel.ExpenseCategory, error) {
	if m.shouldFail {
		return nil, m.failError
	}
	for _, cat := range m.categories {
		if cat.ID == id {
			return cat, nil
		}
	}
	return nil, nil
}

func (m *MockRepository) Delete(id int64) error {
	if m.shouldFail {
		return m.failError
	}
	for name, cat := range m.categories {
		if cat.ID == id {
			cat.IsActive = false
			m.categories[name] = cat
			break
		}
	}
	return nil
}

// Helper methods for testing
func (m *MockRepository) SetShouldFail(shouldFail bool, err error) {
	m.shouldFail = shouldFail
	m.failError = err
}

func (m *MockRepository) AddCategory(cat *category.Category) {
	dataCategory := category.ToDataModel(cat)
	m.categories[dataCategory.Name] = dataCategory
}

var _ = Describe("Category Service", func() {
	var (
		mockRepo *MockRepository
		service  *category.Service
		logger   *slog.Logger
	)

	BeforeEach(func() {
		mockRepo = NewMockRepository()
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
		service = category.NewService(mockRepo, logger)
	})

	Describe("GetAllCategories", func() {
		Context("when repository has categories", func() {
			BeforeEach(func() {
				mockRepo.AddCategory(&category.Category{
					ID:          1,
					Name:        "makan",
					Description: "Meals and entertainment",
					IsActive:    true,
				})
				mockRepo.AddCategory(&category.Category{
					ID:          2,
					Name:        "perjalanan",
					Description: "Business travel",
					IsActive:    true,
				})
				mockRepo.AddCategory(&category.Category{
					ID:          3,
					Name:        "inactive",
					Description: "Inactive category",
					IsActive:    false,
				})
			})

			It("should return only active categories", func() {
				categories, err := service.GetAllCategories()
				Expect(err).NotTo(HaveOccurred())
				Expect(categories).To(HaveLen(2))

				names := make([]string, len(categories))
				for i, cat := range categories {
					names[i] = cat.Name
				}
				Expect(names).To(ConsistOf("makan", "perjalanan"))
			})

			It("should return category responses with correct structure", func() {
				categories, err := service.GetAllCategories()
				Expect(err).NotTo(HaveOccurred())
				Expect(categories[0].Name).NotTo(BeEmpty())
				Expect(categories[0].Description).NotTo(BeEmpty())
			})
		})

		Context("when repository returns error", func() {
			BeforeEach(func() {
				mockRepo.SetShouldFail(true, errors.New("database error"))
			})

			It("should return error", func() {
				categories, err := service.GetAllCategories()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("database error"))
				Expect(categories).To(BeNil())
			})
		})

		Context("when repository is empty", func() {
			It("should return empty slice", func() {
				categories, err := service.GetAllCategories()
				Expect(err).NotTo(HaveOccurred())
				Expect(categories).To(HaveLen(0))
			})
		})
	})

	Describe("GetCategoryByName", func() {
		Context("when category exists and is active", func() {
			BeforeEach(func() {
				mockRepo.AddCategory(&category.Category{
					ID:          1,
					Name:        "makan",
					Description: "Meals and entertainment",
					IsActive:    true,
				})
			})

			It("should return the category", func() {
				result, err := service.GetCategoryByName("makan")
				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Name).To(Equal("makan"))
				Expect(result.Description).To(Equal("Meals and entertainment"))
			})
		})

		Context("when category exists but is inactive", func() {
			BeforeEach(func() {
				mockRepo.AddCategory(&category.Category{
					ID:          1,
					Name:        "inactive",
					Description: "Inactive category",
					IsActive:    false,
				})
			})

			It("should return nil", func() {
				result, err := service.GetCategoryByName("inactive")
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(BeNil())
			})
		})

		Context("when category does not exist", func() {
			It("should return nil", func() {
				result, err := service.GetCategoryByName("nonexistent")
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(BeNil())
			})
		})

		Context("when repository returns error", func() {
			BeforeEach(func() {
				mockRepo.SetShouldFail(true, errors.New("connection error"))
			})

			It("should return error", func() {
				result, err := service.GetCategoryByName("makan")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("connection error"))
				Expect(result).To(BeNil())
			})
		})
	})

	Describe("IsValidCategory", func() {
		Context("when category exists and is active", func() {
			BeforeEach(func() {
				mockRepo.AddCategory(&category.Category{
					ID:          1,
					Name:        "makan",
					Description: "Meals and entertainment",
					IsActive:    true,
				})
			})

			It("should return true", func() {
				result := service.IsValidCategory("makan")
				Expect(result).To(BeTrue())
			})
		})

		Context("when category does not exist", func() {
			It("should return false", func() {
				result := service.IsValidCategory("nonexistent")
				Expect(result).To(BeFalse())
			})
		})

		Context("when category exists but is inactive", func() {
			BeforeEach(func() {
				mockRepo.AddCategory(&category.Category{
					ID:          1,
					Name:        "inactive",
					Description: "Inactive category",
					IsActive:    false,
				})
			})

			It("should return false", func() {
				result := service.IsValidCategory("inactive")
				Expect(result).To(BeFalse())
			})
		})

		Context("when repository returns error", func() {
			BeforeEach(func() {
				mockRepo.SetShouldFail(true, errors.New("database error"))
			})

			It("should return false and log warning", func() {
				result := service.IsValidCategory("makan")
				Expect(result).To(BeFalse())
			})
		})
	})
})
