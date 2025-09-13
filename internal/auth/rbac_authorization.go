package auth

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/frahmantamala/expense-management/internal"
)

type PermissionAuthorizer interface {
	HasPermission(ctx context.Context, userPermissions []string, permission string) (bool, error)
	CanApproveExpensesCtx(ctx context.Context, userPermissions []string) (bool, error)
	CanRejectExpensesCtx(ctx context.Context, userPermissions []string) (bool, error)
	CanRetryPaymentsCtx(ctx context.Context, userPermissions []string) (bool, error)
	IsManagerCtx(ctx context.Context, userPermissions []string) (bool, error)
	IsAdminCtx(ctx context.Context, userPermissions []string) (bool, error)
}

type RBACAuthorization struct {
	authorizer PermissionAuthorizer
	logger     *slog.Logger
}

func NewRBACAuthorization(authorizer PermissionAuthorizer, logger *slog.Logger) *RBACAuthorization {
	return &RBACAuthorization{
		authorizer: authorizer,
		logger:     logger,
	}
}

func (ra *RBACAuthorization) Check(next http.HandlerFunc, permission string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := internal.UserFromContext(r.Context())
		if !ok || user == nil {
			ra.logger.Warn("authorization check failed: user not found in context")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		hasAccess, err := ra.authorizer.HasPermission(r.Context(), user.Permissions, permission)
		if err != nil {
			ra.logger.ErrorContext(r.Context(), "authorization check failed", "error", err, "user_id", user.ID, "permission", permission)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if !hasAccess {
			ra.logger.WarnContext(r.Context(), "access denied: insufficient permissions",
				"user_id", user.ID,
				"required_permission", permission,
				"user_permissions", user.Permissions)
			http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	}
}

func (ra *RBACAuthorization) Middleware(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return ra.Check(next.ServeHTTP, permission)
	}
}

func (ra *RBACAuthorization) RequireApproveExpense() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := internal.UserFromContext(r.Context())
			if !ok || user == nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			canApprove, err := ra.authorizer.CanApproveExpensesCtx(r.Context(), user.Permissions)
			if err != nil {
				ra.logger.ErrorContext(r.Context(), "approval check failed", "error", err, "user_id", user.ID)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			if !canApprove {
				ra.logger.WarnContext(r.Context(), "access denied: cannot approve expenses", "user_id", user.ID)
				http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (ra *RBACAuthorization) RequireRejectExpense() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := internal.UserFromContext(r.Context())
			if !ok || user == nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			canReject, err := ra.authorizer.CanRejectExpensesCtx(r.Context(), user.Permissions)
			if err != nil {
				ra.logger.ErrorContext(r.Context(), "rejection check failed", "error", err, "user_id", user.ID)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			if !canReject {
				ra.logger.WarnContext(r.Context(), "access denied: cannot reject expenses", "user_id", user.ID)
				http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (ra *RBACAuthorization) RequireRetryPayment() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := internal.UserFromContext(r.Context())
			if !ok || user == nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			canRetry, err := ra.authorizer.CanRetryPaymentsCtx(r.Context(), user.Permissions)
			if err != nil {
				ra.logger.ErrorContext(r.Context(), "retry payment check failed", "error", err, "user_id", user.ID)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			if !canRetry {
				ra.logger.WarnContext(r.Context(), "access denied: cannot retry payments", "user_id", user.ID)
				http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (ra *RBACAuthorization) RequireManager() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := internal.UserFromContext(r.Context())
			if !ok || user == nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			isManager, err := ra.authorizer.IsManagerCtx(r.Context(), user.Permissions)
			if err != nil {
				ra.logger.ErrorContext(r.Context(), "manager check failed", "error", err, "user_id", user.ID)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			if !isManager {
				ra.logger.WarnContext(r.Context(), "access denied: manager permissions required", "user_id", user.ID)
				http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (ra *RBACAuthorization) RequireAdmin() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := internal.UserFromContext(r.Context())
			if !ok || user == nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			isAdmin, err := ra.authorizer.IsAdminCtx(r.Context(), user.Permissions)
			if err != nil {
				ra.logger.ErrorContext(r.Context(), "admin check failed", "error", err, "user_id", user.ID)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			if !isAdmin {
				ra.logger.WarnContext(r.Context(), "access denied: admin permissions required", "user_id", user.ID)
				http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
