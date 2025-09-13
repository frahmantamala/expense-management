package expense

import (
	"fmt"
	"log/slog"
	"time"

	expenseDatamodel "github.com/frahmantamala/expense-management/internal/core/datamodel/expense"
)

type RepositoryAPI interface {
	Create(expense *expenseDatamodel.Expense) error
	GetByID(id int64) (*expenseDatamodel.Expense, error)
	GetByUserID(userID int64, params *ExpenseQueryParams) ([]*expenseDatamodel.Expense, error)
	GetAllExpenses(params *ExpenseQueryParams) ([]*expenseDatamodel.Expense, error)
	Update(expense *expenseDatamodel.Expense) error
	UpdateStatus(id int64, status string, processedAt time.Time) error
}

type PaymentProcessorAPI interface {
	ProcessPayment(expenseID int64, amount int64) (externalID string, err error)
	RetryPayment(expenseID int64, externalID string) error
	GetPaymentStatus(expenseID int64) (interface{}, error)
}

type Service struct {
	repo             RepositoryAPI
	paymentProcessor PaymentProcessorAPI
	logger           *slog.Logger
}

func NewService(repo RepositoryAPI, paymentProcessor PaymentProcessorAPI, logger *slog.Logger) *Service {
	return &Service{
		repo:             repo,
		paymentProcessor: paymentProcessor,
		logger:           logger,
	}
}

func (s *Service) CreateExpense(req *CreateExpenseDTO, userID int64) (*Expense, error) {
	if err := req.Validate(); err != nil {
		s.logger.Error("expense validation failed", "error", err, "user_id", userID)
		return nil, err
	}

	expense := NewExpense(userID, *req)

	// Convert to datamodel for repository
	expenseData := ToDataModel(expense)
	if err := s.repo.Create(expenseData); err != nil {
		s.logger.Error("failed to create expense", "error", err, "user_id", userID)
		return nil, fmt.Errorf("failed to create expense: %w", err)
	}

	// Update domain entity with generated ID
	expense.ID = expenseData.ID

	if expense.NeedsPaymentProcessing() {
		s.logger.Info("expense auto-approved, triggering payment",
			"expense_id", expense.ID,
			"amount", expense.AmountIDR)

		go s.processPaymentAsync(expense.ID, expense.AmountIDR)
	}

	s.logger.Info("expense created successfully",
		"expense_id", expense.ID,
		"user_id", userID,
		"amount", req.AmountIDR,
		"status", expense.ExpenseStatus)

	return expense, nil
}

func (s *Service) GetExpenseByID(id, userID int64, userPermissions []string) (*Expense, error) {
	expenseData, err := s.repo.GetByID(id)
	if err != nil {
		s.logger.Error("failed to get expense", "error", err, "expense_id", id)
		return nil, ErrExpenseNotFound
	}

	// Convert to domain entity
	expense := FromDataModel(expenseData)

	if !expense.CanBeAccessedBy(userID, userPermissions) {
		s.logger.Warn("unauthorized access to expense", "expense_id", id, "user_id", userID, "expense_user_id", expense.UserID)
		return nil, ErrUnauthorizedAccess
	}

	return expense, nil
}

func (s *Service) GetExpensesByUserID(userID int64, userPermissions []string) ([]*Expense, error) {
	// This is a wrapper around GetUserExpenses with no pagination
	return s.GetUserExpenses(userID, 100, 0) // Default limit
}

func (s *Service) UpdateExpenseStatus(expenseID int64, status string, userID int64, userPermissions []string) (*Expense, error) {
	// Update status via repository
	if err := s.repo.UpdateStatus(expenseID, status, time.Now()); err != nil {
		s.logger.Error("failed to update expense status", "error", err, "expense_id", expenseID, "status", status)
		return nil, err
	}

	// Return updated expense
	return s.GetExpenseByID(expenseID, userID, userPermissions)
}

func (s *Service) SubmitExpenseForApproval(expenseID int64, userID int64, userPermissions []string) (*Expense, error) {
	return s.UpdateExpenseStatus(expenseID, "submitted", userID, userPermissions)
}

func (s *Service) GetUserExpenses(userID int64, limit, offset int) ([]*Expense, error) {
	// Convert limit/offset to ExpenseQueryParams for backward compatibility
	params := &ExpenseQueryParams{
		PerPage: limit,
		Page:    (offset / limit) + 1,
	}
	params.SetDefaults()

	expensesData, err := s.repo.GetByUserID(userID, params)
	if err != nil {
		s.logger.Error("failed to get user expenses", "error", err, "user_id", userID)
		return nil, err
	}

	return FromDataModelSlice(expensesData), nil
}

func (s *Service) GetAllExpenses(params *ExpenseQueryParams) ([]*Expense, error) {
	params.SetDefaults()

	expensesData, err := s.repo.GetAllExpenses(params)
	if err != nil {
		s.logger.Error("failed to get all expenses", "error", err)
		return nil, err
	}

	return FromDataModelSlice(expensesData), nil
}

func (s *Service) GetExpensesForUser(userID int64, userPermissions []string, params *ExpenseQueryParams) ([]*Expense, error) {
	params.SetDefaults()

	if CanViewAllExpenses(userPermissions) {
		s.logger.Info("GetExpensesForUser: user has management permissions, returning all expenses",
			"user_id", userID, "permissions", userPermissions)
		return s.GetAllExpenses(params)
	} else {
		s.logger.Info("GetExpensesForUser: regular user, returning only user's expenses",
			"user_id", userID, "permissions", userPermissions)

		expensesData, err := s.repo.GetByUserID(userID, params)
		if err != nil {
			s.logger.Error("failed to get user expenses with query", "error", err, "user_id", userID)
			return nil, err
		}
		return FromDataModelSlice(expensesData), nil
	}
}

func (s *Service) ApproveExpense(expenseID, managerID int64, userPermissions []string) error {
	if !HasManagerPermissions(userPermissions) {
		s.logger.Warn("approve expense denied: insufficient permissions",
			"expense_id", expenseID,
			"manager_id", managerID,
			"permissions", userPermissions)
		return ErrUnauthorizedAccess
	}

	expenseData, err := s.repo.GetByID(expenseID)
	if err != nil {
		s.logger.Error("expense not found for approval", "error", err, "expense_id", expenseID)
		return ErrExpenseNotFound
	}

	// Convert to domain entity for business logic
	expense := FromDataModel(expenseData)

	if !expense.CanBeApproved() {
		s.logger.Warn("cannot approve expense in current status",
			"expense_id", expenseID,
			"current_status", expense.ExpenseStatus)
		return ErrInvalidExpenseStatus
	}

	expense.Approve()

	// Convert back to datamodel for repository update
	updatedExpenseData := ToDataModel(expense)
	if err := s.repo.Update(updatedExpenseData); err != nil {
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
	if !HasManagerPermissions(userPermissions) {
		s.logger.Warn("reject expense denied: insufficient permissions",
			"expense_id", expenseID,
			"manager_id", managerID,
			"permissions", userPermissions)
		return ErrUnauthorizedAccess
	}

	expenseData, err := s.repo.GetByID(expenseID)
	if err != nil {
		s.logger.Error("expense not found for rejection", "error", err, "expense_id", expenseID)
		return ErrExpenseNotFound
	}

	// Convert to domain entity for business logic
	expense := FromDataModel(expenseData)

	if !expense.CanBeRejected() {
		s.logger.Warn("cannot reject expense in current status",
			"expense_id", expenseID,
			"current_status", expense.ExpenseStatus)
		return ErrInvalidExpenseStatus
	}

	expense.Reject()

	// Convert back to datamodel for repository update
	updatedExpenseData := ToDataModel(expense)
	if err := s.repo.Update(updatedExpenseData); err != nil {
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

func (s *Service) RetryPayment(expenseID int64, userPermissions []string) error {
	if !HasManagerPermissions(userPermissions) {
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
