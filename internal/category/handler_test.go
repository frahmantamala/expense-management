package category_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/frahmantamala/expense-management/internal/category"
	categoryPostgres "github.com/frahmantamala/expense-management/internal/category/postgres"
	categoryDatamodel "github.com/frahmantamala/expense-management/internal/core/datamodel/category"
	"github.com/frahmantamala/expense-management/internal/transport"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var _ = Describe("Category Handler Integration", func() {
	var (
		db      *gorm.DB
		repo    category.RepositoryAPI
		service *category.Service
		handler *category.Handler
		slogger *slog.Logger
	)

	BeforeEach(func() {
		var err error
		slogger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

		db, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		Expect(err).NotTo(HaveOccurred())

		err = db.AutoMigrate(&category.Category{})
		Expect(err).NotTo(HaveOccurred())

		repo = categoryPostgres.NewCategoryRepository(db)
		service = category.NewService(repo, slogger)
		baseHandler := &transport.BaseHandler{Logger: slogger}
		handler = category.NewHandler(baseHandler, service)

		testCategories := []*category.Category{
			{Name: "makan", Description: "Meals and entertainment", IsActive: true},
			{Name: "perjalanan", Description: "Business travel", IsActive: true},
		}

		for _, cat := range testCategories {
			err := repo.Create(category.ToDataModel(cat))
			Expect(err).NotTo(HaveOccurred())
		}

		inactiveCategory := &categoryDatamodel.ExpenseCategory{
			Name:        "inactive",
			Description: "Inactive category",
			IsActive:    true,
		}
		err = repo.Create(inactiveCategory)
		Expect(err).NotTo(HaveOccurred())

		inactiveCategory.IsActive = false
		err = repo.Update(inactiveCategory)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should handle GET /categories request successfully", func() {
		req := httptest.NewRequest(http.MethodGet, "/categories", nil)
		w := httptest.NewRecorder()

		handler.GetCategories(w, req)

		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Header().Get("Content-Type")).To(ContainSubstring("application/json"))

		var response category.CategoriesResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		Expect(err).NotTo(HaveOccurred())

		Expect(response.Categories).To(HaveLen(2))

		names := make([]string, len(response.Categories))
		for i, cat := range response.Categories {
			names[i] = cat.Name
		}
		Expect(names).To(ConsistOf("makan", "perjalanan"))
	})

	It("should return proper JSON structure", func() {
		req := httptest.NewRequest(http.MethodGet, "/categories", nil)
		w := httptest.NewRecorder()

		handler.GetCategories(w, req)

		var response category.CategoriesResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		Expect(err).NotTo(HaveOccurred())

		for _, cat := range response.Categories {
			Expect(cat.Name).NotTo(BeEmpty())
			Expect(cat.Description).NotTo(BeEmpty())
		}
	})
})
