package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/frahmantamala/expense-management/internal/auth"
)

// RequirePermissions creates a middleware that checks if user has required permissions
func RequirePermissions(permissions ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := auth.UserFromContext(r.Context())
			if !ok || user == nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Check if user has any of the required permissions
			hasPermission := false
			for _, requiredPerm := range permissions {
				for _, userPerm := range user.Permissions {
					if userPerm == requiredPerm {
						hasPermission = true
						break
					}
				}
				if hasPermission {
					break
				}
			}

			if !hasPermission {
				slog.Warn("Access denied: user lacks required permissions",
					"user_id", user.ID,
					"required_permissions", permissions,
					"user_permissions", user.Permissions)
				http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
				return
			}

			// Add permissions to context for service layer use
			ctx := context.WithValue(r.Context(), "user_permissions", user.Permissions)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// HasManagerPermissions checks if user has manager-level permissions
func HasManagerPermissions(user *auth.User) bool {
	managerPerms := []string{"approve_expenses", "reject_expenses", "admin", "manager"}
	for _, requiredPerm := range managerPerms {
		for _, userPerm := range user.Permissions {
			if userPerm == requiredPerm {
				return true
			}
		}
	}
	return false
}
