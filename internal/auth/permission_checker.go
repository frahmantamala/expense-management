package auth

import "context"

type PermissionChecker interface {
	CanApproveExpenses(userPermissions []string) bool
	CanRejectExpenses(userPermissions []string) bool
	CanRetryPayments(userPermissions []string) bool
	CanViewAllExpenses(userPermissions []string) bool
	HasAnyPermission(userPermissions []string, requiredPermissions []string) bool
	IsManager(userPermissions []string) bool
	IsAdmin(userPermissions []string) bool
}

type DefaultPermissionChecker struct{}

func NewPermissionChecker() PermissionChecker {
	return &DefaultPermissionChecker{}
}

func (c *DefaultPermissionChecker) HasPermission(ctx context.Context, userPermissions []string, permission string) (bool, error) {
	return c.HasAnyPermission(userPermissions, []string{permission}), nil
}

func (c *DefaultPermissionChecker) CanApproveExpensesCtx(ctx context.Context, userPermissions []string) (bool, error) {
	return c.CanApproveExpenses(userPermissions), nil
}

func (c *DefaultPermissionChecker) CanRejectExpensesCtx(ctx context.Context, userPermissions []string) (bool, error) {
	return c.CanRejectExpenses(userPermissions), nil
}

func (c *DefaultPermissionChecker) CanRetryPaymentsCtx(ctx context.Context, userPermissions []string) (bool, error) {
	return c.CanRetryPayments(userPermissions), nil
}

func (c *DefaultPermissionChecker) IsManagerCtx(ctx context.Context, userPermissions []string) (bool, error) {
	return c.IsManager(userPermissions), nil
}

func (c *DefaultPermissionChecker) IsAdminCtx(ctx context.Context, userPermissions []string) (bool, error) {
	return c.IsAdmin(userPermissions), nil
}

func (c *DefaultPermissionChecker) CanApproveExpenses(userPermissions []string) bool {
	return c.HasAnyPermission(userPermissions, []string{"approve_expenses", "admin"})
}

func (c *DefaultPermissionChecker) CanRejectExpenses(userPermissions []string) bool {
	return c.HasAnyPermission(userPermissions, []string{"reject_expenses", "admin"})
}

func (c *DefaultPermissionChecker) CanRetryPayments(userPermissions []string) bool {
	return c.HasAnyPermission(userPermissions, []string{"retry_payments", "admin"})
}

func (c *DefaultPermissionChecker) CanViewAllExpenses(userPermissions []string) bool {
	managerPerms := []string{"admin", "approve_expenses", "reject_expenses", "manager"}
	return c.HasAnyPermission(userPermissions, managerPerms)
}

func (c *DefaultPermissionChecker) HasAnyPermission(userPermissions []string, requiredPermissions []string) bool {
	for _, userPerm := range userPermissions {
		for _, requiredPerm := range requiredPermissions {
			if userPerm == requiredPerm {
				return true
			}
		}
	}
	return false
}

func (c *DefaultPermissionChecker) IsManager(userPermissions []string) bool {
	managerPerms := []string{"manager", "admin", "approve_expenses", "reject_expenses"}
	return c.HasAnyPermission(userPermissions, managerPerms)
}

func (c *DefaultPermissionChecker) IsAdmin(userPermissions []string) bool {
	return c.HasAnyPermission(userPermissions, []string{"admin"})
}
