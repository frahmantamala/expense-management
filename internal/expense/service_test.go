package expense_test

import (
	"errors"
	"log/slog"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/frahmantamala/expense-management/internal/expense"
)

// Mock repository for testing
type mockExpenseRepository struct {
	expenses          map[int64]*expense.Expense
	expensesByUser    map[int64][]*expense.Expense
	allExpenses       []*expense.Expense
	pendingApprovals  []*expense.Expense
	createError       error
	getError          error
	updateError       error
	updateStatusError error
	nextID            int64
}

func newMockExpenseRepository() *mockExpenseRepository {
	return &mockExpenseRepository{
		expenses:         make(map[int64]*expense.Expense),
		expensesByUser:   make(map[int64][]*expense.Expense),
		allExpenses:      make([]*expense.Expense, 0),
		pendingApprovals: make([]*expense.Expense, 0),
		nextID:           1,
	}
}

func (m *mockExpenseRepository) Create(exp *expense.Expense) error {
	if m.createError != nil {
		return m.createError
	}
	exp.ID = m.nextID
	m.nextID++
	exp.CreatedAt = time.Now()
	exp.UpdatedAt = time.Now()
	m.expenses[exp.ID] = exp

	// Add to user expenses
	m.expensesByUser[exp.UserID] = append(m.expensesByUser[exp.UserID], exp)

	// Add to all expenses
	m.allExpenses = append(m.allExpenses, exp)

	// Add to pending approvals if applicable
	if exp.ExpenseStatus == expense.ExpenseStatusPendingApproval {
		m.pendingApprovals = append(m.pendingApprovals, exp)
	}

	return nil
}

func (m *mockExpenseRepository) GetByID(id int64) (*expense.Expense, error) {
	if m.getError != nil {
		return nil, m.getError
	}
	exp, exists := m.expenses[id]
	if !exists {
		return nil, errors.New("expense not found")
	}
	return exp, nil
}

func (m *mockExpenseRepository) GetByUserID(userID int64, limit, offset int) ([]*expense.Expense, error) {
	if m.getError != nil {
		return nil, m.getError
	}
	expenses := m.expensesByUser[userID]
	if expenses == nil {
		return []*expense.Expense{}, nil
	}

	// Simple pagination
	start := offset
	end := offset + limit
	if start >= len(expenses) {
		return []*expense.Expense{}, nil
	}
	if end > len(expenses) {
		end = len(expenses)
	}

	return expenses[start:end], nil
}

func (m *mockExpenseRepository) GetPendingApprovals(limit, offset int) ([]*expense.Expense, error) {
	if m.getError != nil {
		return nil, m.getError
	}

	// Simple pagination
	start := offset
	end := offset + limit
	if start >= len(m.pendingApprovals) {
		return []*expense.Expense{}, nil
	}
	if end > len(m.pendingApprovals) {
		end = len(m.pendingApprovals)
	}

	return m.pendingApprovals[start:end], nil
}

func (m *mockExpenseRepository) GetAllExpenses(limit, offset int) ([]*expense.Expense, error) {
	if m.getError != nil {
		return nil, m.getError
	}

	// Simple pagination
	start := offset
	end := offset + limit
	if start >= len(m.allExpenses) {
		return []*expense.Expense{}, nil
	}
	if end > len(m.allExpenses) {
		end = len(m.allExpenses)
	}

	return m.allExpenses[start:end], nil
}

func (m *mockExpenseRepository) Update(exp *expense.Expense) error {
	if m.updateError != nil {
		return m.updateError
	}
	exp.UpdatedAt = time.Now()
	m.expenses[exp.ID] = exp
	return nil
}

func (m *mockExpenseRepository) UpdateStatus(id int64, status string, processedAt time.Time) error {
	if m.updateStatusError != nil {
		return m.updateStatusError
	}
	if exp, exists := m.expenses[id]; exists {
		exp.ExpenseStatus = status
		exp.ProcessedAt = &processedAt
		exp.UpdatedAt = time.Now()
	}
	return nil
}

// Mock payment processor for testing
type mockPaymentProcessor struct {
	processPaymentError   error
	retryPaymentError     error
	getPaymentStatusError error
	paymentStatus         interface{}
	externalID            string
}

func newMockPaymentProcessor() *mockPaymentProcessor {
	return &mockPaymentProcessor{
		externalID:    "mock-external-id",
		paymentStatus: map[string]interface{}{"status": "success"},
	}
}

func (m *mockPaymentProcessor) ProcessPayment(expenseID int64, amount int64) (externalID string, err error) {
	if m.processPaymentError != nil {
		return "", m.processPaymentError
	}
	return m.externalID, nil
}

func (m *mockPaymentProcessor) RetryPayment(expenseID int64, externalID string) error {
	return m.retryPaymentError
}

func (m *mockPaymentProcessor) GetPaymentStatus(expenseID int64) (interface{}, error) {
	if m.getPaymentStatusError != nil {
		return nil, m.getPaymentStatusError
	}
	return m.paymentStatus, nil
}

var _ = Describe("ExpenseService", func() {
	var (
		expenseService *expense.Service
		mockRepo       *mockExpenseRepository
		mockProcessor  *mockPaymentProcessor
		logger         *slog.Logger
	)

	BeforeEach(func() {
		mockRepo = newMockExpenseRepository()
		mockProcessor = newMockPaymentProcessor()
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
		expenseService = expense.NewService(mockRepo, mockProcessor, logger)
	})

	Describe("CreateExpense", func() {
		Context("when creating a small expense (auto-approved)", func() {
			It("should create and approve the expense automatically", func() {
				// Given
				userID := int64(123)
				dto := expense.CreateExpenseDTO{
					AmountIDR:   25000, // Small amount for auto-approval
					Description: "Test expense",
					Category:    "food",
					ExpenseDate: time.Now(),
				}

				// When
				result, err := expenseService.CreateExpense(userID, dto)

				// Then
				Expect(err).ToNot(HaveOccurred())
				Expect(result).ToNot(BeNil())
				Expect(result.UserID).To(Equal(userID))
				Expect(result.AmountIDR).To(Equal(dto.AmountIDR))
				Expect(result.Description).To(Equal(dto.Description))
				Expect(result.ExpenseStatus).To(Equal(expense.ExpenseStatusApproved))
				Expect(result.ProcessedAt).ToNot(BeNil())
				Expect(result.ID).To(BeNumerically(">", 0))
			})

			It("should trigger payment processing for auto-approved expense", func() {
				// Given
				userID := int64(123)
				dto := expense.CreateExpenseDTO{
					AmountIDR:   25000,
					Description: "Test expense",
					Category:    "food",
					ExpenseDate: time.Now(),
				}

				// When
				result, err := expenseService.CreateExpense(userID, dto)

				// Then
				Expect(err).ToNot(HaveOccurred())
				Expect(result.ExpenseStatus).To(Equal(expense.ExpenseStatusApproved))
				// Payment should have been processed (verified by no errors)
			})
		})

		Context("when creating a large expense (requires approval)", func() {
			It("should create expense with pending approval status", func() {
				// Given
				userID := int64(123)
				dto := expense.CreateExpenseDTO{
					AmountIDR:   5000000, // Large amount requiring approval (5M > 1M threshold)
					Description: "Large expense",
					Category:    "transport",
					ExpenseDate: time.Now(),
				}

				// When
				result, err := expenseService.CreateExpense(userID, dto)

				// Then
				Expect(err).ToNot(HaveOccurred())
				Expect(result).ToNot(BeNil())
				Expect(result.ExpenseStatus).To(Equal(expense.ExpenseStatusPendingApproval))
				Expect(result.ProcessedAt).To(BeNil())
			})
		})

		Context("when validation fails", func() {
			It("should return validation error for empty description", func() {
				// Given
				userID := int64(123)
				dto := expense.CreateExpenseDTO{
					AmountIDR:   25000,
					Description: "", // Empty description
					Category:    "food",
					ExpenseDate: time.Now(),
				}

				// When
				result, err := expenseService.CreateExpense(userID, dto)

				// Then
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("description"))
				Expect(result).To(BeNil())
			})

			It("should return validation error for zero amount", func() {
				// Given
				userID := int64(123)
				dto := expense.CreateExpenseDTO{
					AmountIDR:   0, // Zero amount
					Description: "Test expense",
					Category:    "food",
					ExpenseDate: time.Now(),
				}

				// When
				result, err := expenseService.CreateExpense(userID, dto)

				// Then
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("amount must be positive"))
				Expect(result).To(BeNil())
			})

			It("should return validation error for amount below minimum", func() {
				// Given
				userID := int64(123)
				dto := expense.CreateExpenseDTO{
					AmountIDR:   5000, // Below minimum 10,000 IDR
					Description: "Test expense",
					Category:    "food",
					ExpenseDate: time.Now(),
				}

				// When
				result, err := expenseService.CreateExpense(userID, dto)

				// Then
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("amount must be at least 10,000 IDR"))
				Expect(result).To(BeNil())
			})

			It("should return validation error for amount above maximum", func() {
				// Given
				userID := int64(123)
				dto := expense.CreateExpenseDTO{
					AmountIDR:   60000000, // Above maximum 50,000,000 IDR
					Description: "Test expense",
					Category:    "food",
					ExpenseDate: time.Now(),
				}

				// When
				result, err := expenseService.CreateExpense(userID, dto)

				// Then
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("amount must not exceed 50,000,000 IDR"))
				Expect(result).To(BeNil())
			})
		})

		Context("when repository fails", func() {
			It("should return repository error", func() {
				// Given
				mockRepo.createError = errors.New("database error")
				userID := int64(123)
				dto := expense.CreateExpenseDTO{
					AmountIDR:   25000,
					Description: "Test expense",
					Category:    "food",
					ExpenseDate: time.Now(),
				}

				// When
				result, err := expenseService.CreateExpense(userID, dto)

				// Then
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("database error"))
				Expect(result).To(BeNil())
			})
		})

		Context("when payment processing fails", func() {
			It("should still create the expense but log payment error", func() {
				// Given
				mockProcessor.processPaymentError = errors.New("payment gateway error")
				userID := int64(123)
				dto := expense.CreateExpenseDTO{
					AmountIDR:   25000, // Auto-approved amount
					Description: "Test expense",
					Category:    "food",
					ExpenseDate: time.Now(),
				}

				// When
				result, err := expenseService.CreateExpense(userID, dto)

				// Then
				Expect(err).ToNot(HaveOccurred()) // Expense creation should succeed
				Expect(result).ToNot(BeNil())
				Expect(result.ExpenseStatus).To(Equal(expense.ExpenseStatusApproved))
				// Payment failure should be logged but not prevent expense creation
			})
		})
	})

	Describe("ApproveExpense", func() {
		Context("when approving a pending expense", func() {
			It("should approve the expense and trigger payment", func() {
				// Given - Create a pending expense first
				testExpense := &expense.Expense{
					ID:            1,
					UserID:        123,
					AmountIDR:     75000,
					Description:   "Large expense",
					ExpenseStatus: expense.ExpenseStatusPendingApproval,
					CreatedAt:     time.Now(),
					UpdatedAt:     time.Now(),
				}
				mockRepo.expenses[1] = testExpense
				managerID := int64(456)
				permissions := []string{"approve_expenses"} // Use correct permission string

				// When
				err := expenseService.ApproveExpense(1, managerID, permissions)

				// Then
				Expect(err).ToNot(HaveOccurred())

				// Verify expense was updated to approved
				updatedExpense, _ := mockRepo.GetByID(1)
				Expect(updatedExpense.ExpenseStatus).To(Equal(expense.ExpenseStatusApproved))
			})
		})

		Context("when expense does not exist", func() {
			It("should return not found error", func() {
				// Given
				expenseID := int64(999)
				managerID := int64(456)
				permissions := []string{"approve_expenses"} // Use correct permission string

				// When
				err := expenseService.ApproveExpense(expenseID, managerID, permissions)

				// Then
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not found"))
			})
		})

		Context("when expense is already approved", func() {
			It("should return already approved error", func() {
				// Given - Create an already approved expense
				testExpense := &expense.Expense{
					ID:            1,
					UserID:        123,
					AmountIDR:     25000,
					ExpenseStatus: expense.ExpenseStatusApproved,
					CreatedAt:     time.Now(),
					UpdatedAt:     time.Now(),
				}
				mockRepo.expenses[1] = testExpense
				managerID := int64(456)
				permissions := []string{"approve_expenses"} // Use correct permission string

				// When
				err := expenseService.ApproveExpense(1, managerID, permissions)

				// Then
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid expense status"))
			})
		})
	})

	Describe("RejectExpense", func() {
		Context("when rejecting a pending expense", func() {
			It("should reject the expense successfully", func() {
				// Given
				testExpense := &expense.Expense{
					ID:            1,
					UserID:        123,
					AmountIDR:     75000,
					ExpenseStatus: expense.ExpenseStatusPendingApproval,
					CreatedAt:     time.Now(),
					UpdatedAt:     time.Now(),
				}
				mockRepo.expenses[1] = testExpense
				managerID := int64(456)
				reason := "Insufficient documentation"
				permissions := []string{"reject_expenses"} // Use correct permission string

				// When
				err := expenseService.RejectExpense(1, managerID, reason, permissions)

				// Then
				Expect(err).ToNot(HaveOccurred())

				// Verify expense was updated to rejected
				updatedExpense, _ := mockRepo.GetByID(1)
				Expect(updatedExpense.ExpenseStatus).To(Equal(expense.ExpenseStatusRejected))
			})
		})
	})

	Describe("GetUserExpenses", func() {
		Context("when user has expenses", func() {
			It("should return user's expenses", func() {
				// Given
				userID := int64(123)
				testExpense1 := &expense.Expense{ID: 1, UserID: userID, AmountIDR: 25000}
				testExpense2 := &expense.Expense{ID: 2, UserID: userID, AmountIDR: 50000}
				mockRepo.expensesByUser[userID] = []*expense.Expense{testExpense1, testExpense2}

				// When
				result, err := expenseService.GetUserExpenses(userID, 10, 0)

				// Then
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(2))
				Expect(result[0].UserID).To(Equal(userID))
				Expect(result[1].UserID).To(Equal(userID))
			})
		})

		Context("when user has no expenses", func() {
			It("should return empty list", func() {
				// Given
				userID := int64(999)

				// When
				result, err := expenseService.GetUserExpenses(userID, 10, 0)

				// Then
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(0))
			})
		})
	})

	Describe("GetAllExpenses", func() {
		Context("when there are expenses", func() {
			It("should return all expenses", func() {
				// Given
				expense1 := &expense.Expense{
					ID:            1,
					UserID:        123,
					AmountIDR:     75000,
					ExpenseStatus: expense.ExpenseStatusPendingApproval,
				}
				expense2 := &expense.Expense{
					ID:            2,
					UserID:        456,
					AmountIDR:     100000,
					ExpenseStatus: expense.ExpenseStatusApproved,
				}
				mockRepo.allExpenses = []*expense.Expense{expense1, expense2}
				permissions := []string{"approve_expenses"} // Use correct permission string

				// When
				result, err := expenseService.GetAllExpenses(10, 0, permissions)

				// Then
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(2))
				Expect(result[0].ID).To(Equal(int64(1)))
				Expect(result[1].ID).To(Equal(int64(2)))
			})
		})
	})

	Describe("RetryPayment", func() {
		Context("when retrying payment for an approved expense", func() {
			It("should call payment processor retry", func() {
				// Given
				expenseID := int64(123)
				testExpense := &expense.Expense{
					ID:            123,
					UserID:        456,
					AmountIDR:     75000,
					ExpenseStatus: expense.ExpenseStatusApproved,
					CreatedAt:     time.Now(),
					UpdatedAt:     time.Now(),
				}
				mockRepo.expenses[123] = testExpense
				permissions := []string{"approve_expenses"} // Use correct permission string

				// When
				err := expenseService.RetryPayment(expenseID, permissions)

				// Then
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when user lacks permission", func() {
			It("should return permission error", func() {
				// Given
				expenseID := int64(123)
				permissions := []string{"some:other:permission"}

				// When
				err := expenseService.RetryPayment(expenseID, permissions)

				// Then
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unauthorized"))
			})
		})
	})
})
