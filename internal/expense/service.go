package expense

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/frahmantamala/expense-management/internal/auth"
	expenseDatamodel "github.com/frahmantamala/expense-management/internal/core/datamodel/expense"
	"github.com/frahmantamala/expense-management/internal/core/events"
)

type RepositoryAPI interface {
	Create(expense *expenseDatamodel.Expense) error
	GetByID(id int64) (*expenseDatamodel.Expense, error)
	GetByUserID(userID int64, params *ExpenseQueryParams) ([]*expenseDatamodel.Expense, error)
	GetAllExpenses(params *ExpenseQueryParams) ([]*expenseDatamodel.Expense, error)
	CountByUserID(userID int64, params *ExpenseQueryParams) (int64, error)
	CountAllExpenses(params *ExpenseQueryParams) (int64, error)
	Update(expense *expenseDatamodel.Expense) error
	UpdateStatus(id int64, status string, processedAt time.Time) error
}

type PaymentProcessorAPI interface {
	ProcessPayment(expenseID int64, amount int64) (externalID string, err error)
	RetryPayment(expenseID int64, externalID string) error
	GetPaymentStatus(expenseID int64) (interface{}, error)
}

type Service struct {
	repo              RepositoryAPI
	paymentProcessor  PaymentProcessorAPI
	permissionChecker auth.PermissionChecker
	eventBus          *events.EventBus
	logger            *slog.Logger
}

func NewService(repo RepositoryAPI, paymentProcessor PaymentProcessorAPI, permissionChecker auth.PermissionChecker, eventBus *events.EventBus, logger *slog.Logger) *Service {
	service := &Service{
		repo:              repo,
		paymentProcessor:  paymentProcessor,
		permissionChecker: permissionChecker,
		eventBus:          eventBus,
		logger:            logger,
	}

	service.RegisterEventHandlers()

	return service
}

func (s *Service) CreateExpense(req *CreateExpenseDTO, userID int64) (*Expense, error) {
	if err := req.Validate(); err != nil {
		s.logger.Error("expense validation failed", "error", err, "user_id", userID)
		return nil, err
	}

	expense := NewExpense(userID, *req)

	expenseData := ToDataModel(expense)
	if err := s.repo.Create(expenseData); err != nil {
		s.logger.Error("failed to create expense", "error", err, "user_id", userID)
		return nil, fmt.Errorf("failed to create expense: %w", err)
	}

	expense.ID = expenseData.ID

	if expense.NeedsPaymentProcessing() {
		s.logger.Info("expense auto-approved, triggering payment via event",
			"expense_id", expense.ID,
			"amount", expense.AmountIDR)

		event := events.NewExpenseApprovedEvent(expense.ID, expense.AmountIDR, expense.UserID, "IDR")
		if err := s.eventBus.Publish(context.Background(), event); err != nil {
			s.logger.Error("failed to publish auto-approval event",
				"error", err,
				"expense_id", expense.ID)

		} else {
			s.logger.Info("auto-approval event published for async payment processing",
				"expense_id", expense.ID,
				"event_id", event.EventID())
		}
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

	expense := FromDataModel(expenseData)

	canAccess := expense.UserID == userID || s.permissionChecker.CanViewAllExpenses(userPermissions)
	if !canAccess {
		s.logger.Warn("unauthorized access to expense", "expense_id", id, "user_id", userID, "expense_user_id", expense.UserID)
		return nil, ErrUnauthorizedAccess
	}

	return expense, nil
}

func (s *Service) UpdateExpenseStatus(expenseID int64, status string, userID int64, userPermissions []string) (*Expense, error) {

	if err := s.repo.UpdateStatus(expenseID, status, time.Now()); err != nil {
		s.logger.Error("failed to update expense status", "error", err, "expense_id", expenseID, "status", status)
		return nil, err
	}

	return s.GetExpenseByID(expenseID, userID, userPermissions)
}

func (s *Service) SubmitExpenseForApproval(expenseID int64, userID int64, userPermissions []string) (*Expense, error) {
	return s.UpdateExpenseStatus(expenseID, "submitted", userID, userPermissions)
}

func (s *Service) GetAllExpenses(params *ExpenseQueryParams) ([]*Expense, error) {
	params.SetDefaults()

	s.logger.Info("GetAllExpenses: Starting with params",
		"page", params.Page,
		"per_page", params.PerPage,
		"offset_calculated", params.GetOffset(),
		"search", params.Search,
		"category", params.CategoryID,
		"status", params.Status)

	expensesData, err := s.repo.GetAllExpenses(params)
	if err != nil {
		s.logger.Error("failed to get all expenses", "error", err)
		return nil, err
	}

	return FromDataModelSlice(expensesData), nil
}

func (s *Service) GetExpensesForUser(userID int64, userPermissions []string, params *ExpenseQueryParams) ([]*Expense, error) {
	params.SetDefaults()

	if s.permissionChecker.CanViewAllExpenses(userPermissions) {
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

func (s *Service) GetExpensesCountForUser(userID int64, userPermissions []string, params *ExpenseQueryParams) (int64, error) {
	if s.permissionChecker.CanViewAllExpenses(userPermissions) {
		return s.repo.CountAllExpenses(params)
	} else {
		return s.repo.CountByUserID(userID, params)
	}
}

func (s *Service) ApproveExpense(expenseID, managerID int64, userPermissions []string) error {
	if !s.permissionChecker.CanApproveExpenses(userPermissions) {
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

	expense := FromDataModel(expenseData)

	if !expense.CanBeApproved() {
		s.logger.Warn("cannot approve expense in current status",
			"expense_id", expenseID,
			"current_status", expense.ExpenseStatus)
		return ErrInvalidExpenseStatus
	}

	expense.Approve()

	updatedExpenseData := ToDataModel(expense)
	if err := s.repo.Update(updatedExpenseData); err != nil {
		s.logger.Error("failed to update expense status to approved", "error", err, "expense_id", expenseID)
		return err
	}

	s.logger.Info("expense approved successfully",
		"expense_id", expenseID,
		"manager_id", managerID,
		"amount", expense.AmountIDR)

	event := events.NewExpenseApprovedEvent(expenseID, expense.AmountIDR, expense.UserID, "IDR")
	if err := s.eventBus.Publish(context.Background(), event); err != nil {
		s.logger.Error("failed to publish expense approved event",
			"error", err,
			"expense_id", expenseID)

	} else {
		s.logger.Info("expense approved event published for async payment processing",
			"expense_id", expenseID,
			"event_id", event.EventID())
	}

	return nil
}

func (s *Service) RejectExpense(expenseID, managerID int64, reason string, userPermissions []string) error {
	if !s.permissionChecker.CanRejectExpenses(userPermissions) {
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

	expense := FromDataModel(expenseData)

	if !expense.CanBeRejected() {
		s.logger.Warn("cannot reject expense in current status",
			"expense_id", expenseID,
			"current_status", expense.ExpenseStatus)
		return ErrInvalidExpenseStatus
	}

	expense.Reject()

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
	if !s.permissionChecker.CanRetryPayments(userPermissions) {
		s.logger.Warn("user lacks permissions for payment retry", "expense_id", expenseID)
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

	externalID := fmt.Sprintf("exp-%d-%d", expenseID, expense.AmountIDR)
	err = s.paymentProcessor.RetryPayment(expenseID, externalID)
	if err != nil {
		s.logger.Error("payment retry failed", "error", err, "expense_id", expenseID)
		return fmt.Errorf("payment retry failed: %w", err)
	}

	return nil
}

func (s *Service) RegisterEventHandlers() {
	s.eventBus.Subscribe(events.EventTypePaymentCompleted, s.handlePaymentCompleted)
	s.logger.Info("expense event handlers registered", "handlers", []string{events.EventTypePaymentCompleted})
}

func (s *Service) handlePaymentCompleted(ctx context.Context, event events.Event) error {
	paymentEvent, ok := event.(*events.PaymentCompletedEvent)
	if !ok {
		s.logger.Error("invalid event type for payment completed handler", "event_type", event.EventType())
		return fmt.Errorf("expected PaymentCompletedEvent, got %T", event)
	}

	s.logger.Info("handling payment completed event to update expense status",
		"expense_id", paymentEvent.ExpenseID,
		"payment_id", paymentEvent.PaymentID,
		"external_id", paymentEvent.ExternalID,
		"event_id", paymentEvent.EventID())

	err := s.repo.UpdateStatus(paymentEvent.ExpenseID, ExpenseStatusCompleted, time.Now())
	if err != nil {
		s.logger.Error("failed to update expense status after payment completion",
			"error", err,
			"expense_id", paymentEvent.ExpenseID,
			"payment_id", paymentEvent.PaymentID,
			"event_id", paymentEvent.EventID())
		return fmt.Errorf("expense status update failed for expense %d: %w", paymentEvent.ExpenseID, err)
	}

	s.logger.Info("expense status updated to completed successfully",
		"expense_id", paymentEvent.ExpenseID,
		"payment_id", paymentEvent.PaymentID,
		"external_id", paymentEvent.ExternalID,
		"event_id", paymentEvent.EventID())

	return nil
}
