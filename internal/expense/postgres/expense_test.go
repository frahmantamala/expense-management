package postgres

import (
	"testing"
	"time"

	expenseDatamodel "github.com/frahmantamala/expense-management/internal/core/datamodel/expense"
	"github.com/frahmantamala/expense-management/internal/expense"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestExpenseRepository(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ExpenseRepository Suite")
}

type SQLiteExpense struct {
	ID              int64      `gorm:"primaryKey"`
	UserID          int64      `gorm:"column:user_id;not null"`
	AmountIDR       int64      `gorm:"column:amount_idr;not null"`
	Description     string     `gorm:"not null"`
	Category        string     `gorm:"column:category"`
	ReceiptURL      *string    `gorm:"column:receipt_url"`
	ReceiptFileName *string    `gorm:"column:receipt_filename"`
	ExpenseStatus   string     `gorm:"column:expense_status;default:'pending_approval'"`
	ExpenseDate     time.Time  `gorm:"column:expense_date"`
	SubmittedAt     time.Time  `gorm:"column:submitted_at"`
	ProcessedAt     *time.Time `gorm:"column:processed_at"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at"`
}

func (SQLiteExpense) TableName() string {
	return "expenses"
}

var _ = Describe("ExpenseRepository", func() {
	var (
		db   *gorm.DB
		repo expense.RepositoryAPI
	)

	BeforeEach(func() {
		var err error

		db, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		Expect(err).NotTo(HaveOccurred())

		err = db.AutoMigrate(&SQLiteExpense{})
		Expect(err).NotTo(HaveOccurred())

		repo = NewExpenseRepository(db)
	})

	AfterEach(func() {

		sqlDB, err := db.DB()
		Expect(err).NotTo(HaveOccurred())
		err = sqlDB.Close()
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("Create", func() {
		It("should create an expense successfully", func() {
			expense := &expenseDatamodel.Expense{
				UserID:        1,
				AmountIDR:     100000,
				Description:   "Test expense",
				Category:      "makan",
				ExpenseStatus: "pending_approval",
				ExpenseDate:   time.Now().AddDate(0, 0, -1),
				SubmittedAt:   time.Now(),
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}

			err := repo.Create(expense)
			Expect(err).NotTo(HaveOccurred())
			Expect(expense.ID).To(BeNumerically(">", 0))
		})
	})

	Describe("GetByID", func() {
		var createdExpense *expenseDatamodel.Expense

		BeforeEach(func() {
			createdExpense = &expenseDatamodel.Expense{
				UserID:        1,
				AmountIDR:     100000,
				Description:   "Test expense",
				Category:      "Travel",
				ExpenseStatus: "pending_approval",
				ExpenseDate:   time.Now(),
				SubmittedAt:   time.Now(),
			}
			err := repo.Create(createdExpense)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should retrieve expense by ID successfully", func() {
			retrieved, err := repo.GetByID(createdExpense.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved).NotTo(BeNil())
			Expect(retrieved.ID).To(Equal(createdExpense.ID))
			Expect(retrieved.UserID).To(Equal(createdExpense.UserID))
			Expect(retrieved.AmountIDR).To(Equal(createdExpense.AmountIDR))
			Expect(retrieved.Description).To(Equal(createdExpense.Description))
			Expect(retrieved.Category).To(Equal(createdExpense.Category))
			Expect(retrieved.ExpenseStatus).To(Equal(createdExpense.ExpenseStatus))
		})

		It("should return ErrExpenseNotFound for non-existent ID", func() {
			retrieved, err := repo.GetByID(99999)
			Expect(err).To(Equal(expense.ErrExpenseNotFound))
			Expect(retrieved).To(BeNil())
		})
	})

	Describe("Update", func() {
		var createdExpense *expenseDatamodel.Expense

		BeforeEach(func() {
			createdExpense = &expenseDatamodel.Expense{
				UserID:        1,
				AmountIDR:     100000,
				Description:   "Original description",
				Category:      "Travel",
				ExpenseStatus: "pending_approval",
				ExpenseDate:   time.Now(),
				SubmittedAt:   time.Now(),
			}
			err := repo.Create(createdExpense)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should update expense successfully", func() {

			createdExpense.Description = "Updated description"
			createdExpense.AmountIDR = 200000
			createdExpense.Category = "Food"

			err := repo.Update(createdExpense)
			Expect(err).NotTo(HaveOccurred())

			retrieved, err := repo.GetByID(createdExpense.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved.Description).To(Equal("Updated description"))
			Expect(retrieved.AmountIDR).To(Equal(int64(200000)))
			Expect(retrieved.Category).To(Equal("Food"))
		})
	})

	Describe("UpdateStatus", func() {
		var createdExpense *expenseDatamodel.Expense

		BeforeEach(func() {
			createdExpense = &expenseDatamodel.Expense{
				UserID:        1,
				AmountIDR:     100000,
				Description:   "Test expense",
				Category:      "Travel",
				ExpenseStatus: "pending_approval",
				ExpenseDate:   time.Now(),
				SubmittedAt:   time.Now(),
			}
			err := repo.Create(createdExpense)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should update status and processed_at successfully", func() {
			processedAt := time.Now()

			err := repo.UpdateStatus(createdExpense.ID, "approved", processedAt)
			Expect(err).NotTo(HaveOccurred())

			retrieved, err := repo.GetByID(createdExpense.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved.ExpenseStatus).To(Equal("approved"))
			Expect(retrieved.ProcessedAt).NotTo(BeNil())
			Expect(retrieved.ProcessedAt.Unix()).To(Equal(processedAt.Unix()))
		})
	})
})
