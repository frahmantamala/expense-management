package expense

import (
	"log/slog"
	"time"
)

// PaymentProcessor interface defines payment processing methods
type PaymentProcessor interface {
	ProcessPayment(expenseID int64, amount int64) (paymentID, externalID string, err error)
	RetryPayment(externalID string, amount int64) (paymentID string, err error)
}

// Repository interface defines the data access methods for expenses
type Repository interface {
	Create(expense *Expense) error
	GetByID(id int64) (*Expense, error)
	GetByUserID(userID int64, limit, offset int) ([]*Expense, error)
	GetPendingApprovals(limit, offset int) ([]*Expense, error)
	Update(expense *Expense) error
	UpdateStatus(id int64, status string, processedAt time.Time) error
	UpdatePaymentInfo(id int64, paymentStatus, paymentID, paymentExternalID string, paidAt *time.Time) error
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

// GetPendingApprovals retrieves expenses pending approval with permission check
func (s *Service) GetPendingApprovals(limit, offset int, userPermissions []string) ([]*Expense, error) {
	// Check permissions at service level
	if !s.hasManagerPermissions(userPermissions) {
		s.logger.Warn("get pending approvals denied: insufficient permissions", "permissions", userPermissions)
		return nil, ErrUnauthorizedAccess
	}

	expenses, err := s.repo.GetPendingApprovals(limit, offset)
	if err != nil {
		s.logger.Error("failed to get pending approvals", "error", err)
		return nil, err
	}

	return expenses, nil
}

// ApproveExpense approves an expense (manager only)
func (s *Service) ApproveExpense(expenseID, managerID int64, userPermissions []string) error {
	// Check permissions at service level
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

	// Check if expense can be approved
	if expense.ExpenseStatus != ExpenseStatusPendingApproval {
		s.logger.Warn("cannot approve expense in current status",
			"expense_id", expenseID,
			"current_status", expense.ExpenseStatus)
		return ErrInvalidExpenseStatus
	}

	// Update status to approved
	processedAt := time.Now()
	if err := s.repo.UpdateStatus(expenseID, ExpenseStatusApproved, processedAt); err != nil {
		s.logger.Error("failed to update expense status to approved", "error", err, "expense_id", expenseID)
		return err
	}

	s.logger.Info("expense approved successfully",
		"expense_id", expenseID,
		"manager_id", managerID,
		"amount", expense.AmountIDR)

	// Trigger payment processing for approved expense
	s.logger.Info("expense manually approved, triggering payment",
		"expense_id", expenseID,
		"amount", expense.AmountIDR)

	go s.processPaymentAsync(expenseID, expense.AmountIDR)

	return nil
} // RejectExpense rejects an expense (manager only)
func (s *Service) RejectExpense(expenseID, managerID int64, reason string, userPermissions []string) error {
	// Check permissions at service level
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

	// Check if expense can be rejected
	if expense.ExpenseStatus != ExpenseStatusPendingApproval {
		s.logger.Warn("cannot reject expense in current status",
			"expense_id", expenseID,
			"current_status", expense.ExpenseStatus)
		return ErrInvalidExpenseStatus
	}

	// Update status to rejected
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

// processPaymentAsync processes payment for an approved expense asynchronously
func (s *Service) processPaymentAsync(expenseID int64, amount int64) {
	s.logger.Info("starting payment processing", "expense_id", expenseID, "amount", amount)

	// Set initial payment status to pending
	if err := s.repo.UpdatePaymentInfo(expenseID, PaymentStatusPending, "", "", nil); err != nil {
		s.logger.Error("failed to set payment status to pending", "error", err, "expense_id", expenseID)
		return
	}

	// Process payment through payment service
	paymentID, externalID, err := s.paymentProcessor.ProcessPayment(expenseID, amount)
	if err != nil {
		s.logger.Error("payment processing failed", "error", err, "expense_id", expenseID)

		// Update payment status to failed
		if updateErr := s.repo.UpdatePaymentInfo(expenseID, PaymentStatusFailed, "", externalID, nil); updateErr != nil {
			s.logger.Error("failed to update payment status to failed", "error", updateErr, "expense_id", expenseID)
		}
		return
	}

	// Payment succeeded, update status
	paidAt := time.Now()
	if err := s.repo.UpdatePaymentInfo(expenseID, PaymentStatusSuccess, paymentID, externalID, &paidAt); err != nil {
		s.logger.Error("failed to update payment status to success", "error", err, "expense_id", expenseID)
		return
	}

	s.logger.Info("payment processed successfully",
		"expense_id", expenseID,
		"payment_id", paymentID,
		"external_id", externalID)
}

// RetryPayment retries payment for a failed expense payment
func (s *Service) RetryPayment(expenseID int64, userPermissions []string) error {
	// Check permissions (managers can retry payments)
	if !s.hasManagerPermissions(userPermissions) {
		s.logger.Warn("user lacks manager permissions for payment retry", "expense_id", expenseID)
		return ErrUnauthorizedAccess
	}

	// Get expense
	expense, err := s.repo.GetByID(expenseID)
	if err != nil {
		s.logger.Error("failed to get expense for payment retry", "error", err, "expense_id", expenseID)
		return ErrExpenseNotFound
	}

	// Check if expense is approved and payment failed
	if expense.ExpenseStatus != ExpenseStatusApproved {
		s.logger.Error("expense not approved for payment retry", "expense_id", expenseID, "status", expense.ExpenseStatus)
		return ErrInvalidExpenseStatus
	}

	if expense.PaymentStatus == nil || *expense.PaymentStatus != PaymentStatusFailed {
		s.logger.Error("payment retry not allowed for current payment status",
			"expense_id", expenseID,
			"payment_status", expense.PaymentStatus)
		return ErrInvalidExpenseStatus
	}

	s.logger.Info("retrying payment", "expense_id", expenseID, "amount", expense.AmountIDR)

	// Trigger payment processing asynchronously
	go s.processPaymentAsync(expenseID, expense.AmountIDR)

	return nil
}
