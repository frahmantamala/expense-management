package postgres_test

import (
	"testing"
	"time"

	"github.com/frahmantamala/expense-management/internal/category"
	categoryPostgres "github.com/frahmantamala/expense-management/internal/category/postgres"
	categoryDatamodel "github.com/frahmantamala/expense-management/internal/core/datamodel/category"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestCategoryPostgres(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Category Postgres Suite")
}

// SQLiteCategory is a SQLite-compatible model for testing
type SQLiteCategory struct {
	ID          int64     `gorm:"primaryKey"`
	Name        string    `gorm:"column:name;uniqueIndex;not null"`
	Description string    `gorm:"column:description"`
	IsActive    bool      `gorm:"column:is_active;default:true"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (SQLiteCategory) TableName() string {
	return "categories"
}

var _ = Describe("Category PostgreSQL Repository", func() {
	var (
		db   *gorm.DB
		repo category.RepositoryAPI
	)

	BeforeEach(func() {
		var err error
		// Use SQLite in-memory database for testing
		db, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		Expect(err).NotTo(HaveOccurred())

		// Create the table using SQLite-compatible model
		err = db.AutoMigrate(&SQLiteCategory{})
		Expect(err).NotTo(HaveOccurred())

		repo = categoryPostgres.NewCategoryRepository(db)
	})

	Describe("Create", func() {
		It("should create a new category successfully", func() {
			cat := &categoryDatamodel.ExpenseCategory{
				Name:        "makan",
				Description: "Meals and entertainment",
				IsActive:    true,
			}

			err := repo.Create(cat)
			Expect(err).NotTo(HaveOccurred())
			Expect(cat.ID).To(BeNumerically(">", 0))
			Expect(cat.CreatedAt).NotTo(BeZero())
		})

		It("should fail to create duplicate category", func() {
			cat1 := &categoryDatamodel.ExpenseCategory{
				Name:        "makan",
				Description: "Meals and entertainment",
				IsActive:    true,
			}

			err := repo.Create(cat1)
			Expect(err).NotTo(HaveOccurred())

			cat2 := &categoryDatamodel.ExpenseCategory{
				Name:        "makan",
				Description: "Duplicate category",
				IsActive:    true,
			}

			err = repo.Create(cat2)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("GetAll", func() {
		BeforeEach(func() {
			activeCategories := []*categoryDatamodel.ExpenseCategory{
				{
					Name:        "makan",
					Description: "Meals and entertainment",
					IsActive:    true,
				},
				{
					Name:        "perjalanan",
					Description: "Business travel",
					IsActive:    true,
				},
			}

			for _, cat := range activeCategories {
				err := repo.Create(cat)
				Expect(err).NotTo(HaveOccurred())
			}

			// Create inactive category separately and update it
			inactiveCategory := &categoryDatamodel.ExpenseCategory{
				Name:        "kantor",
				Description: "Office supplies",
				IsActive:    true, // Create as active first
			}
			err := repo.Create(inactiveCategory)
			Expect(err).NotTo(HaveOccurred())

			// Then update to inactive
			inactiveCategory.IsActive = false
			err = repo.Update(inactiveCategory)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should retrieve all categories ordered by name", func() {
			categories, err := repo.GetAll()
			Expect(err).NotTo(HaveOccurred())
			Expect(categories).To(HaveLen(3))

			// Should be ordered by name ASC
			Expect(categories[0].Name).To(Equal("kantor"))
			Expect(categories[1].Name).To(Equal("makan"))
			Expect(categories[2].Name).To(Equal("perjalanan"))
		})

		It("should include both active and inactive categories", func() {
			categories, err := repo.GetAll()
			Expect(err).NotTo(HaveOccurred())

			activeCount := 0
			inactiveCount := 0
			for _, cat := range categories {
				if cat.IsActive {
					activeCount++
				} else {
					inactiveCount++
				}
			}

			Expect(activeCount).To(Equal(2))
			Expect(inactiveCount).To(Equal(1))
		})
	})

	Describe("GetByName", func() {
		var testCategory *categoryDatamodel.ExpenseCategory

		BeforeEach(func() {
			testCategory = &categoryDatamodel.ExpenseCategory{
				Name:        "makan",
				Description: "Meals and entertainment",
				IsActive:    true,
			}
			err := repo.Create(testCategory)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should retrieve category by name successfully", func() {
			result, err := repo.GetByName("makan")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.Name).To(Equal("makan"))
			Expect(result.Description).To(Equal("Meals and entertainment"))
			Expect(result.IsActive).To(BeTrue())
		})

		It("should return nil for non-existent category", func() {
			result, err := repo.GetByName("nonexistent")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeNil())
		})

		It("should be case sensitive", func() {
			result, err := repo.GetByName("MAKAN")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})

	Describe("Update", func() {
		var testCategory *categoryDatamodel.ExpenseCategory

		BeforeEach(func() {
			testCategory = &categoryDatamodel.ExpenseCategory{
				Name:        "makan",
				Description: "Meals and entertainment",
				IsActive:    true,
			}
			err := repo.Create(testCategory)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should update category successfully", func() {
			testCategory.Description = "Updated description"
			testCategory.IsActive = false

			err := repo.Update(testCategory)
			Expect(err).NotTo(HaveOccurred())

			// Verify the update
			result, err := repo.GetByName("makan")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Description).To(Equal("Updated description"))
			Expect(result.IsActive).To(BeFalse())
		})

		It("should update the updated_at timestamp", func() {
			originalUpdatedAt := testCategory.UpdatedAt
			time.Sleep(10 * time.Millisecond) // Ensure timestamp difference

			testCategory.Description = "New description"
			err := repo.Update(testCategory)
			Expect(err).NotTo(HaveOccurred())

			result, err := repo.GetByName("makan")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.UpdatedAt).To(BeTemporally(">", originalUpdatedAt))
		})
	})

	Describe("Delete", func() {
		var testCategory *categoryDatamodel.ExpenseCategory

		BeforeEach(func() {
			testCategory = &categoryDatamodel.ExpenseCategory{
				Name:        "makan",
				Description: "Meals and entertainment",
				IsActive:    true,
			}
			err := repo.Create(testCategory)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should soft delete category by setting is_active to false", func() {
			err := repo.Delete(testCategory.ID)
			Expect(err).NotTo(HaveOccurred())

			// Category should still exist but be inactive
			result, err := repo.GetByName("makan")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.IsActive).To(BeFalse())
		})

		It("should handle non-existent ID gracefully", func() {
			err := repo.Delete(999)
			Expect(err).NotTo(HaveOccurred())

			// Original category should remain unchanged
			result, err := repo.GetByName("makan")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsActive).To(BeTrue())
		})
	})

	Describe("Database constraints", func() {
		It("should enforce unique constraint on name", func() {
			cat1 := &categoryDatamodel.ExpenseCategory{
				Name:        "duplicate",
				Description: "First category",
				IsActive:    true,
			}
			err := repo.Create(cat1)
			Expect(err).NotTo(HaveOccurred())

			cat2 := &categoryDatamodel.ExpenseCategory{
				Name:        "duplicate",
				Description: "Second category",
				IsActive:    true,
			}
			err = repo.Create(cat2)
			Expect(err).To(HaveOccurred())
		})

		It("should set default values correctly", func() {
			cat := &categoryDatamodel.ExpenseCategory{
				Name:        "test",
				Description: "Test category",
				// IsActive not set, should default to true
			}
			err := repo.Create(cat)
			Expect(err).NotTo(HaveOccurred())

			result, err := repo.GetByName("test")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsActive).To(BeTrue())
			Expect(result.CreatedAt).NotTo(BeZero())
			Expect(result.UpdatedAt).NotTo(BeZero())
		})
	})
})
