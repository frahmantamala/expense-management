package expense

import (
	"fmt"
	"log/slog"
	"time"
)

// PaymentProcessor interface defines payment processing methods
type PaymentProcessor interface {
	ProcessPayment(expenseID int64, amount int64) (externalID string, err error)
	RetryPayment(expenseID int64, externalID string) error
	GetPaymentStatus(expenseID int64) (interface{}, error) // Returns payment view
}

// Repository interface defines the data access methods for expenses
type Repository interface {
	Create(expense *Expense) error
	GetByID(id int64) (*Expense, error)
	GetByUserID(userID int64, limit, offset int) ([]*Expense, error)
	GetAllExpenses(limit, offset int) ([]*Expense, error)
	Update(expense *Expense) error
	UpdateStatus(id int64, status string, processedAt time.Time) error
}

// Service handles expense business logic
type Service struct {
	repo             Repository
	paymentProcessor PaymentProcessor
	logger           *slog.Logger
}

// NewService creates a new expense service
func NewService(repo Repository, paymentProcessor PaymentProcessor, logger *slog.Logger) *Service {
	return &Service{
		repo:             repo,
		paymentProcessor: paymentProcessor,
		logger:           logger,
	}
}

// CreateExpense creates a new expense with automatic approval logic
func (s *Service) CreateExpense(userID int64, dto CreateExpenseDTO) (*Expense, error) {
	if err := dto.Validate(); err != nil {
		s.logger.Error("expense validation failed", "error", err, "user_id", userID)
		return nil, err
	}

	// Determine initial status based on amount
	status := ExpenseStatusPendingApproval
	var processedAt *time.Time
	if dto.AmountIDR < AutoApprovalThreshold {
		status = ExpenseStatusApproved
		now := time.Now()
		processedAt = &now
	}

	expense := &Expense{
		UserID:          userID,
		AmountIDR:       dto.AmountIDR,
		Description:     dto.Description,
		Category:        dto.Category,
		ReceiptURL:      dto.ReceiptURL,
		ReceiptFileName: dto.ReceiptFileName,
		ExpenseStatus:   status,
		ExpenseDate:     dto.ExpenseDate,
		SubmittedAt:     time.Now(),
		ProcessedAt:     processedAt,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.repo.Create(expense); err != nil {
		s.logger.Error("failed to create expense", "error", err, "user_id", userID)
		return nil, err
	}

	// If expense is approved (auto-approved), trigger payment processing
	if status == ExpenseStatusApproved {
		s.logger.Info("expense auto-approved, triggering payment",
			"expense_id", expense.ID,
			"amount", expense.AmountIDR)

		go s.processPaymentAsync(expense.ID, expense.AmountIDR)
	}

	s.logger.Info("expense created successfully",
		"expense_id", expense.ID,
		"user_id", userID,
		"amount", dto.AmountIDR,
		"status", status)

	return expense, nil
}

// GetExpenseByID retrieves an expense by ID with access control
func (s *Service) GetExpenseByID(id, userID int64, isManager bool) (*Expense, error) {
	expense, err := s.repo.GetByID(id)
	if err != nil {
		s.logger.Error("failed to get expense", "error", err, "expense_id", id)
		return nil, ErrExpenseNotFound
	}

	// Check access permissions
	if !isManager && expense.UserID != userID {
		s.logger.Warn("unauthorized access to expense", "expense_id", id, "user_id", userID, "expense_user_id", expense.UserID)
		return nil, ErrUnauthorizedAccess
	}

	return expense, nil
}

// GetUserExpenses retrieves expenses for a specific user
func (s *Service) GetUserExpenses(userID int64, limit, offset int) ([]*Expense, error) {
	expenses, err := s.repo.GetByUserID(userID, limit, offset)
	if err != nil {
		s.logger.Error("failed to get user expenses", "error", err, "user_id", userID)
		return nil, err
	}

	return expenses, nil
}

func (s *Service) GetAllExpenses(limit, offset int, userPermissions []string) ([]*Expense, error) {
	if !s.hasManagerPermissions(userPermissions) {
		s.logger.Warn("get all expenses denied: insufficient permissions", "permissions", userPermissions)
		return nil, ErrUnauthorizedAccess
	}

	expenses, err := s.repo.GetAllExpenses(limit, offset)
	if err != nil {
		s.logger.Error("failed to get all expenses", "error", err)
		return nil, err
	}

	return expenses, nil
}

func (s *Service) ApproveExpense(expenseID, managerID int64, userPermissions []string) error {
	if !s.hasManagerPermissions(userPermissions) {
		s.logger.Warn("approve expense denied: insufficient permissions",
			"expense_id", expenseID,
			"manager_id", managerID,
			"permissions", userPermissions)
		return ErrUnauthorizedAccess
	}

	expense, err := s.repo.GetByID(expenseID)
	if err != nil {
		s.logger.Error("expense not found for approval", "error", err, "expense_id", expenseID)
		return ErrExpenseNotFound
	}

	if expense.ExpenseStatus != ExpenseStatusPendingApproval {
		s.logger.Warn("cannot approve expense in current status",
			"expense_id", expenseID,
			"current_status", expense.ExpenseStatus)
		return ErrInvalidExpenseStatus
	}

	processedAt := time.Now()
	if err := s.repo.UpdateStatus(expenseID, ExpenseStatusApproved, processedAt); err != nil {
		s.logger.Error("failed to update expense status to approved", "error", err, "expense_id", expenseID)
		return err
	}

	s.logger.Info("expense approved successfully",
		"expense_id", expenseID,
		"manager_id", managerID,
		"amount", expense.AmountIDR)

	s.logger.Info("expense manually approved, triggering payment",
		"expense_id", expenseID,
		"amount", expense.AmountIDR)

	go s.processPaymentAsync(expenseID, expense.AmountIDR)

	return nil
}

func (s *Service) RejectExpense(expenseID, managerID int64, reason string, userPermissions []string) error {
	if !s.hasManagerPermissions(userPermissions) {
		s.logger.Warn("reject expense denied: insufficient permissions",
			"expense_id", expenseID,
			"manager_id", managerID,
			"permissions", userPermissions)
		return ErrUnauthorizedAccess
	}

	expense, err := s.repo.GetByID(expenseID)
	if err != nil {
		s.logger.Error("expense not found for rejection", "error", err, "expense_id", expenseID)
		return ErrExpenseNotFound
	}

	if expense.ExpenseStatus != ExpenseStatusPendingApproval {
		s.logger.Warn("cannot reject expense in current status",
			"expense_id", expenseID,
			"current_status", expense.ExpenseStatus)
		return ErrInvalidExpenseStatus
	}

	processedAt := time.Now()
	if err := s.repo.UpdateStatus(expenseID, ExpenseStatusRejected, processedAt); err != nil {
		s.logger.Error("failed to update expense status to rejected", "error", err, "expense_id", expenseID)
		return err
	}

	s.logger.Info("expense rejected successfully",
		"expense_id", expenseID,
		"manager_id", managerID,
		"reason", reason,
		"amount", expense.AmountIDR)

	return nil
}

// hasManagerPermissions checks if user has manager-level permissions
func (s *Service) hasManagerPermissions(userPermissions []string) bool {
	managerPerms := []string{"approve_expenses", "reject_expenses", "admin", "manager"}
	for _, requiredPerm := range managerPerms {
		for _, userPerm := range userPermissions {
			if userPerm == requiredPerm {
				return true
			}
		}
	}
	return false
}

func (s *Service) processPaymentAsync(expenseID int64, amount int64) {
	s.logger.Info("starting payment processing", "expense_id", expenseID, "amount", amount)

	externalID, err := s.paymentProcessor.ProcessPayment(expenseID, amount)
	if err != nil {
		s.logger.Error("payment processing failed", "error", err, "expense_id", expenseID, "external_id", externalID)
		return
	}

	s.logger.Info("payment processing initiated successfully",
		"expense_id", expenseID,
		"external_id", externalID)
}

func (s *Service) RetryPayment(expenseID int64, userPermissions []string) error {
	// Check permissions (managers can retry payments)
	if !s.hasManagerPermissions(userPermissions) {
		s.logger.Warn("user lacks manager permissions for payment retry", "expense_id", expenseID)
		return ErrUnauthorizedAccess
	}

	expense, err := s.repo.GetByID(expenseID)
	if err != nil {
		s.logger.Error("failed to get expense for payment retry", "error", err, "expense_id", expenseID)
		return ErrExpenseNotFound
	}

	if expense.ExpenseStatus != ExpenseStatusApproved {
		s.logger.Error("expense not approved for payment retry", "expense_id", expenseID, "status", expense.ExpenseStatus)
		return ErrInvalidExpenseStatus
	}

	// Get payment status from payment service to validate it exists
	_, err = s.paymentProcessor.GetPaymentStatus(expenseID)
	if err != nil {
		s.logger.Error("failed to get payment status", "error", err, "expense_id", expenseID)
		return ErrInvalidExpenseStatus
	}

	s.logger.Info("retrying payment", "expense_id", expenseID, "amount", expense.AmountIDR)

	externalID := fmt.Sprintf("expense-%d-%d", expenseID, expense.AmountIDR)
	err = s.paymentProcessor.RetryPayment(expenseID, externalID)
	if err != nil {
		s.logger.Error("payment retry failed", "error", err, "expense_id", expenseID)
		return fmt.Errorf("payment retry failed: %w", err)
	}

	return nil
}
