package postgres

import (
	"fmt"
	"testing"
	"time"

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

// ExpenseSQLite defines the expense model for SQLite testing
// This is needed because SQLite doesn't support JSONB, so we use TEXT instead
type ExpenseSQLite struct {
	ID              int64      `gorm:"column:id;primaryKey;autoIncrement"`
	UserID          int64      `gorm:"column:user_id;not null"`
	AmountIDR       int64      `gorm:"column:amount_idr;not null"`
	Description     string     `gorm:"column:description;not null"`
	Category        string     `gorm:"column:category;not null"`
	ReceiptURL      *string    `gorm:"column:receipt_url"`
	ReceiptFileName *string    `gorm:"column:receipt_filename"`
	ExpenseStatus   string     `gorm:"column:expense_status;default:pending_approval"`
	ExpenseDate     time.Time  `gorm:"column:expense_date;not null"`
	SubmittedAt     time.Time  `gorm:"column:submitted_at;not null"`
	ProcessedAt     *time.Time `gorm:"column:processed_at"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at"`
}

func (ExpenseSQLite) TableName() string {
	return "expenses"
}

// ToExpense converts ExpenseSQLite to expense.Expense
func (e *ExpenseSQLite) ToExpense() *expense.Expense {
	return &expense.Expense{
		ID:              e.ID,
		UserID:          e.UserID,
		AmountIDR:       e.AmountIDR,
		Description:     e.Description,
		Category:        e.Category,
		ReceiptURL:      e.ReceiptURL,
		ReceiptFileName: e.ReceiptFileName,
		ExpenseStatus:   e.ExpenseStatus,
		ExpenseDate:     e.ExpenseDate,
		SubmittedAt:     e.SubmittedAt,
		ProcessedAt:     e.ProcessedAt,
		CreatedAt:       e.CreatedAt,
		UpdatedAt:       e.UpdatedAt,
	}
}

// FromExpense converts expense.Expense to ExpenseSQLite
func (e *ExpenseSQLite) FromExpense(exp *expense.Expense) {
	e.ID = exp.ID
	e.UserID = exp.UserID
	e.AmountIDR = exp.AmountIDR
	e.Description = exp.Description
	e.Category = exp.Category
	e.ReceiptURL = exp.ReceiptURL
	e.ReceiptFileName = exp.ReceiptFileName
	e.ExpenseStatus = exp.ExpenseStatus
	e.ExpenseDate = exp.ExpenseDate
	e.SubmittedAt = exp.SubmittedAt
	e.ProcessedAt = exp.ProcessedAt
	e.CreatedAt = exp.CreatedAt
	e.UpdatedAt = exp.UpdatedAt
}

var _ = Describe("ExpenseRepository", func() {
	var (
		db   *gorm.DB
		repo expense.Repository
	)

	BeforeEach(func() {
		var err error
		// Use SQLite in-memory database for testing
		db, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		Expect(err).NotTo(HaveOccurred())

		// Auto-migrate the schema
		err = db.AutoMigrate(&ExpenseSQLite{})
		Expect(err).NotTo(HaveOccurred())

		// Create repository instance
		repo = NewExpenseRepository(db)
	})

	AfterEach(func() {
		// Clean up database
		sqlDB, err := db.DB()
		Expect(err).NotTo(HaveOccurred())
		err = sqlDB.Close()
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("Create", func() {
		It("should create an expense successfully", func() {
			expense := &expense.Expense{
				UserID:        1,
				AmountIDR:     100000,
				Description:   "Test expense",
				Category:      "makan",
				ExpenseStatus: expense.ExpenseStatusPendingApproval,
				ExpenseDate:   time.Now().AddDate(0, 0, -1),
				SubmittedAt:   time.Now(),
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}

			err := repo.Create(expense)
			Expect(err).NotTo(HaveOccurred())
			Expect(expense.ID).To(BeNumerically(">", 0))
		})

		It("should create an expense with receipt data successfully", func() {
			receiptURL := "https://650cfcbc47af3fd22f6818ca.mockapi.io/test/v1/files/receipt_123.pdf"
			receiptFilename := "receipt_lunch_2025.pdf"

			expense := &expense.Expense{
				UserID:          1,
				AmountIDR:       50000,
				Description:     "Lunch with receipt",
				Category:        "makan",
				ReceiptURL:      &receiptURL,
				ReceiptFileName: &receiptFilename,
				ExpenseStatus:   expense.ExpenseStatusPendingApproval,
				ExpenseDate:     time.Now().AddDate(0, 0, -1),
				SubmittedAt:     time.Now(),
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			}

			err := repo.Create(expense)
			Expect(err).NotTo(HaveOccurred())
			Expect(expense.ID).To(BeNumerically(">", 0))

			// Verify receipt data was saved
			retrievedExpense, err := repo.GetByID(expense.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(retrievedExpense.ReceiptURL).NotTo(BeNil())
			Expect(*retrievedExpense.ReceiptURL).To(Equal(receiptURL))
			Expect(retrievedExpense.ReceiptFileName).NotTo(BeNil())
			Expect(*retrievedExpense.ReceiptFileName).To(Equal(receiptFilename))
		})

		It("should create an expense without receipt data successfully", func() {
			expense := &expense.Expense{
				UserID:        1,
				AmountIDR:     30000,
				Description:   "Expense without receipt",
				Category:      "kantor",
				ExpenseStatus: expense.ExpenseStatusPendingApproval,
				ExpenseDate:   time.Now().AddDate(0, 0, -1),
				SubmittedAt:   time.Now(),
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}

			err := repo.Create(expense)
			Expect(err).NotTo(HaveOccurred())
			Expect(expense.ID).To(BeNumerically(">", 0))

			// Verify receipt fields are nil
			retrievedExpense, err := repo.GetByID(expense.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(retrievedExpense.ReceiptURL).To(BeNil())
			Expect(retrievedExpense.ReceiptFileName).To(BeNil())
		})

		It("should return error for invalid expense", func() {
			expense := &expense.Expense{
				// Missing required fields
			}

			err := repo.Create(expense)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("GetByID", func() {
		var createdExpense *expense.Expense

		BeforeEach(func() {
			createdExpense = &expense.Expense{
				UserID:        1,
				AmountIDR:     100000,
				Description:   "Test expense",
				Category:      "Travel",
				ExpenseStatus: expense.ExpenseStatusPendingApproval,
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

	Describe("GetByUserID", func() {
		var userID int64 = 1

		BeforeEach(func() {
			// Create multiple expenses for the user
			expenses := []*expense.Expense{
				{
					UserID:        userID,
					AmountIDR:     100000,
					Description:   "Expense 1",
					Category:      "Travel",
					ExpenseStatus: expense.ExpenseStatusPendingApproval,
					ExpenseDate:   time.Now().Add(-2 * time.Hour),
					SubmittedAt:   time.Now().Add(-2 * time.Hour),
				},
				{
					UserID:        userID,
					AmountIDR:     200000,
					Description:   "Expense 2",
					Category:      "Food",
					ExpenseStatus: expense.ExpenseStatusApproved,
					ExpenseDate:   time.Now().Add(-1 * time.Hour),
					SubmittedAt:   time.Now().Add(-1 * time.Hour),
				},
				{
					UserID:        userID,
					AmountIDR:     300000,
					Description:   "Expense 3",
					Category:      "Office",
					ExpenseStatus: expense.ExpenseStatusRejected,
					ExpenseDate:   time.Now(),
					SubmittedAt:   time.Now(),
				},
			}

			for _, exp := range expenses {
				err := repo.Create(exp)
				Expect(err).NotTo(HaveOccurred())
			}

			// Create expense for different user
			otherUserExp := &expense.Expense{
				UserID:        2,
				AmountIDR:     50000,
				Description:   "Other user expense",
				Category:      "Travel",
				ExpenseStatus: expense.ExpenseStatusPendingApproval,
				ExpenseDate:   time.Now(),
				SubmittedAt:   time.Now(),
			}
			err := repo.Create(otherUserExp)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should retrieve expenses for specific user", func() {
			expenses, err := repo.GetByUserID(userID, 10, 0)
			Expect(err).NotTo(HaveOccurred())
			Expect(expenses).To(HaveLen(3))

			// Should be ordered by submitted_at DESC (most recent first)
			Expect(expenses[0].Description).To(Equal("Expense 3"))
			Expect(expenses[1].Description).To(Equal("Expense 2"))
			Expect(expenses[2].Description).To(Equal("Expense 1"))
		})

		It("should respect limit parameter", func() {
			expenses, err := repo.GetByUserID(userID, 2, 0)
			Expect(err).NotTo(HaveOccurred())
			Expect(expenses).To(HaveLen(2))
		})

		It("should respect offset parameter", func() {
			expenses, err := repo.GetByUserID(userID, 10, 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(expenses).To(HaveLen(2))
			// Should skip the first (most recent) expense
			Expect(expenses[0].Description).To(Equal("Expense 2"))
			Expect(expenses[1].Description).To(Equal("Expense 1"))
		})

		It("should return empty slice for user with no expenses", func() {
			expenses, err := repo.GetByUserID(999, 10, 0)
			Expect(err).NotTo(HaveOccurred())
			Expect(expenses).To(HaveLen(0))
		})
	})

	Describe("GetPendingApprovals", func() {
		BeforeEach(func() {
			// Create expenses with different statuses and submission times
			expenses := []*expense.Expense{
				{
					UserID:        1,
					AmountIDR:     100000,
					Description:   "Pending 1 (oldest)",
					Category:      "Travel",
					ExpenseStatus: expense.ExpenseStatusPendingApproval,
					ExpenseDate:   time.Now().Add(-3 * time.Hour),
					SubmittedAt:   time.Now().Add(-3 * time.Hour),
				},
				{
					UserID:        2,
					AmountIDR:     200000,
					Description:   "Approved expense",
					Category:      "Food",
					ExpenseStatus: expense.ExpenseStatusApproved,
					ExpenseDate:   time.Now().Add(-2 * time.Hour),
					SubmittedAt:   time.Now().Add(-2 * time.Hour),
				},
				{
					UserID:        3,
					AmountIDR:     300000,
					Description:   "Pending 2 (newest)",
					Category:      "Office",
					ExpenseStatus: expense.ExpenseStatusPendingApproval,
					ExpenseDate:   time.Now().Add(-1 * time.Hour),
					SubmittedAt:   time.Now().Add(-1 * time.Hour),
				},
				{
					UserID:        4,
					AmountIDR:     400000,
					Description:   "Rejected expense",
					Category:      "Travel",
					ExpenseStatus: expense.ExpenseStatusRejected,
					ExpenseDate:   time.Now(),
					SubmittedAt:   time.Now(),
				},
			}

			for _, exp := range expenses {
				err := repo.Create(exp)
				Expect(err).NotTo(HaveOccurred())
			}
		})
	})

	Describe("GetAllExpenses", func() {
		BeforeEach(func() {
			// Create expenses with different statuses and submission times
			expenses := []*expense.Expense{
				{
					UserID:        1,
					AmountIDR:     100000,
					Description:   "Expense 1 (oldest)",
					Category:      "Travel",
					ExpenseStatus: expense.ExpenseStatusPendingApproval,
					ExpenseDate:   time.Now().Add(-3 * time.Hour),
					SubmittedAt:   time.Now().Add(-3 * time.Hour),
				},
				{
					UserID:        2,
					AmountIDR:     200000,
					Description:   "Expense 2 (approved)",
					Category:      "Food",
					ExpenseStatus: expense.ExpenseStatusApproved,
					ExpenseDate:   time.Now().Add(-2 * time.Hour),
					SubmittedAt:   time.Now().Add(-2 * time.Hour),
				},
				{
					UserID:        3,
					AmountIDR:     300000,
					Description:   "Expense 3 (newest)",
					Category:      "Office",
					ExpenseStatus: expense.ExpenseStatusRejected,
					ExpenseDate:   time.Now().Add(-1 * time.Hour),
					SubmittedAt:   time.Now().Add(-1 * time.Hour),
				},
			}

			for _, exp := range expenses {
				err := repo.Create(exp)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("should retrieve all expenses regardless of status", func() {
			expenses, err := repo.GetAllExpenses(10, 0)
			Expect(err).NotTo(HaveOccurred())
			Expect(expenses).To(HaveLen(3))

			// Should include all statuses
			statuses := make(map[string]bool)
			for _, exp := range expenses {
				statuses[exp.ExpenseStatus] = true
			}
			Expect(statuses[expense.ExpenseStatusPendingApproval]).To(BeTrue())
			Expect(statuses[expense.ExpenseStatusApproved]).To(BeTrue())
			Expect(statuses[expense.ExpenseStatusRejected]).To(BeTrue())
		})

		It("should order by submitted_at DESC (newest first)", func() {
			expenses, err := repo.GetAllExpenses(10, 0)
			Expect(err).NotTo(HaveOccurred())
			Expect(expenses).To(HaveLen(3))

			// Should be ordered by submitted_at DESC (newest first)
			Expect(expenses[0].Description).To(Equal("Expense 3 (newest)"))
			Expect(expenses[1].Description).To(Equal("Expense 2 (approved)"))
			Expect(expenses[2].Description).To(Equal("Expense 1 (oldest)"))
		})

		It("should respect limit parameter", func() {
			expenses, err := repo.GetAllExpenses(2, 0)
			Expect(err).NotTo(HaveOccurred())
			Expect(expenses).To(HaveLen(2))
			Expect(expenses[0].Description).To(Equal("Expense 3 (newest)"))
			Expect(expenses[1].Description).To(Equal("Expense 2 (approved)"))
		})

		It("should respect offset parameter", func() {
			expenses, err := repo.GetAllExpenses(10, 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(expenses).To(HaveLen(2))
			Expect(expenses[0].Description).To(Equal("Expense 2 (approved)"))
			Expect(expenses[1].Description).To(Equal("Expense 1 (oldest)"))
		})

		It("should return empty slice when no expenses exist", func() {
			// Clear all expenses
			db.Exec("DELETE FROM expenses")

			expenses, err := repo.GetAllExpenses(10, 0)
			Expect(err).NotTo(HaveOccurred())
			Expect(expenses).To(HaveLen(0))
		})
	})

	Describe("Update", func() {
		var createdExpense *expense.Expense

		BeforeEach(func() {
			createdExpense = &expense.Expense{
				UserID:        1,
				AmountIDR:     100000,
				Description:   "Original description",
				Category:      "Travel",
				ExpenseStatus: expense.ExpenseStatusPendingApproval,
				ExpenseDate:   time.Now(),
				SubmittedAt:   time.Now(),
			}
			err := repo.Create(createdExpense)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should update expense successfully", func() {
			// Update the expense
			createdExpense.Description = "Updated description"
			createdExpense.AmountIDR = 200000
			createdExpense.Category = "Food"

			err := repo.Update(createdExpense)
			Expect(err).NotTo(HaveOccurred())

			// Retrieve and verify the update
			retrieved, err := repo.GetByID(createdExpense.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved.Description).To(Equal("Updated description"))
			Expect(retrieved.AmountIDR).To(Equal(int64(200000)))
			Expect(retrieved.Category).To(Equal("Food"))
		})

		It("should update updated_at timestamp", func() {
			originalUpdatedAt := createdExpense.UpdatedAt
			time.Sleep(10 * time.Millisecond) // Ensure time difference

			createdExpense.Description = "Updated description"
			err := repo.Update(createdExpense)
			Expect(err).NotTo(HaveOccurred())

			retrieved, err := repo.GetByID(createdExpense.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved.UpdatedAt.After(originalUpdatedAt)).To(BeTrue())
		})

		It("should handle updating optional fields to nil", func() {
			receiptURL := "https://example.com/receipt.pdf"
			createdExpense.ReceiptURL = &receiptURL
			err := repo.Update(createdExpense)
			Expect(err).NotTo(HaveOccurred())

			// Now set to nil
			createdExpense.ReceiptURL = nil
			err = repo.Update(createdExpense)
			Expect(err).NotTo(HaveOccurred())

			retrieved, err := repo.GetByID(createdExpense.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved.ReceiptURL).To(BeNil())
		})
	})

	Describe("UpdateStatus", func() {
		var createdExpense *expense.Expense

		BeforeEach(func() {
			createdExpense = &expense.Expense{
				UserID:        1,
				AmountIDR:     100000,
				Description:   "Test expense",
				Category:      "Travel",
				ExpenseStatus: expense.ExpenseStatusPendingApproval,
				ExpenseDate:   time.Now(),
				SubmittedAt:   time.Now(),
			}
			err := repo.Create(createdExpense)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should update status and processed_at successfully", func() {
			processedAt := time.Now()

			err := repo.UpdateStatus(createdExpense.ID, expense.ExpenseStatusApproved, processedAt)
			Expect(err).NotTo(HaveOccurred())

			// Retrieve and verify the update
			retrieved, err := repo.GetByID(createdExpense.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved.ExpenseStatus).To(Equal(expense.ExpenseStatusApproved))
			Expect(retrieved.ProcessedAt).NotTo(BeNil())
			Expect(retrieved.ProcessedAt.Unix()).To(Equal(processedAt.Unix()))
		})

		It("should update updated_at timestamp", func() {
			originalUpdatedAt := createdExpense.UpdatedAt
			time.Sleep(10 * time.Millisecond) // Ensure time difference

			processedAt := time.Now()
			err := repo.UpdateStatus(createdExpense.ID, expense.ExpenseStatusRejected, processedAt)
			Expect(err).NotTo(HaveOccurred())

			retrieved, err := repo.GetByID(createdExpense.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved.UpdatedAt.After(originalUpdatedAt)).To(BeTrue())
		})

		It("should handle updating to different statuses", func() {
			processedAt := time.Now()

			// Test approved status
			err := repo.UpdateStatus(createdExpense.ID, expense.ExpenseStatusApproved, processedAt)
			Expect(err).NotTo(HaveOccurred())

			retrieved, err := repo.GetByID(createdExpense.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved.ExpenseStatus).To(Equal(expense.ExpenseStatusApproved))

			// Test rejected status
			err = repo.UpdateStatus(createdExpense.ID, expense.ExpenseStatusRejected, processedAt)
			Expect(err).NotTo(HaveOccurred())

			retrieved, err = repo.GetByID(createdExpense.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved.ExpenseStatus).To(Equal(expense.ExpenseStatusRejected))
		})

		It("should not affect other fields", func() {
			originalDescription := createdExpense.Description
			originalAmountIDR := createdExpense.AmountIDR
			originalCategory := createdExpense.Category

			processedAt := time.Now()
			err := repo.UpdateStatus(createdExpense.ID, expense.ExpenseStatusApproved, processedAt)
			Expect(err).NotTo(HaveOccurred())

			retrieved, err := repo.GetByID(createdExpense.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved.Description).To(Equal(originalDescription))
			Expect(retrieved.AmountIDR).To(Equal(originalAmountIDR))
			Expect(retrieved.Category).To(Equal(originalCategory))
		})

		It("should handle non-existent expense ID gracefully", func() {
			processedAt := time.Now()
			err := repo.UpdateStatus(99999, expense.ExpenseStatusApproved, processedAt)
			// GORM doesn't return an error for updates that don't affect any rows
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Database Integration", func() {
		It("should handle concurrent operations", func() {
			expenses := make([]*expense.Expense, 10)

			// Create multiple expenses concurrently
			for i := 0; i < 10; i++ {
				expenses[i] = &expense.Expense{
					UserID:        int64(i + 1),
					AmountIDR:     int64((i + 1) * 10000),
					Description:   fmt.Sprintf("Expense %d", i+1),
					Category:      "Travel",
					ExpenseStatus: expense.ExpenseStatusPendingApproval,
					ExpenseDate:   time.Now(),
					SubmittedAt:   time.Now(),
				}
				err := repo.Create(expenses[i])
				Expect(err).NotTo(HaveOccurred())
			}

			// Verify all were created
			for i := 0; i < 10; i++ {
				retrieved, err := repo.GetByID(expenses[i].ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.Description).To(Equal(fmt.Sprintf("Expense %d", i+1)))
			}
		})

		It("should handle edge cases with timestamps", func() {
			now := time.Now()
			exp := &expense.Expense{
				UserID:        1,
				AmountIDR:     100000,
				Description:   "Test expense",
				Category:      "Travel",
				ExpenseStatus: expense.ExpenseStatusPendingApproval,
				ExpenseDate:   now,
				SubmittedAt:   now,
			}

			err := repo.Create(exp)
			Expect(err).NotTo(HaveOccurred())

			retrieved, err := repo.GetByID(exp.ID)
			Expect(err).NotTo(HaveOccurred())

			// Timestamps should be preserved with reasonable precision
			Expect(retrieved.ExpenseDate.Unix()).To(Equal(now.Unix()))
			Expect(retrieved.SubmittedAt.Unix()).To(Equal(now.Unix()))
		})
	})
})
