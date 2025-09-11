package auth

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-chi/chi"
	"github.com/jmoiron/sqlx"
)

type ctxKey string

const ContextUserKey ctxKey = "user"

var ErrForbidden = errors.New("forbidden")

func UserFromContext(ctx context.Context) (*User, bool) {
	u, ok := ctx.Value(ContextUserKey).(*User)
	return u, ok
}

// ABACPolicy is a small attribute-based access control helper.
type ABACPolicy struct{}

func (p *ABACPolicy) Allow(userAttrs map[string]string, resourceOwnerID string, action string) bool {
	if attr, ok := userAttrs["attributes"]; ok && attr == "admin" {
		return true
	}

	if role, ok := userAttrs["role"]; ok && role == "admin" {
		return true
	}

	// Permission-based access
	if permissions, ok := userAttrs["permissions"]; ok {
		switch action {
		case "read":
			if strings.Contains(permissions, "can_read_expense") {
				return true
			}
		case "approve":
			if strings.Contains(permissions, "can_approve") {
				return true
			}
		case "reject":
			if strings.Contains(permissions, "can_reject") {
				return true
			}
		}
	}

	// Owner access for basic operations
	if uid, ok := userAttrs["user_id"]; ok && uid == resourceOwnerID {
		return action == "read" || action == "write" || action == "update" || action == "delete"
	}

	return false
}

// CanViewExpense checks whether the user can view the expense owned by ownerID.
func (p *ABACPolicy) CanViewExpense(u *User, ownerID int64) error {
	attrs := extractUserAttributes(u)
	if attrs["user_id"] == "" {
		return ErrForbidden
	}

	allowed := p.Allow(attrs, strconv.FormatInt(ownerID, 10), "read")
	if allowed {
		return nil
	}
	return ErrForbidden
}

// CanApproveExpense checks whether the user can approve expenses
func (p *ABACPolicy) CanApproveExpense(u *User, expenseUserID int64) error {
	attrs := extractUserAttributes(u)
	if attrs["user_id"] == "" {
		return ErrForbidden
	}

	allowed := p.Allow(attrs, strconv.FormatInt(expenseUserID, 10), "approve")
	if allowed {
		return nil
	}
	return ErrForbidden
}

// Enhanced user attribute extraction
func extractUserAttributes(u *User) map[string]string {
	if u == nil {
		return map[string]string{}
	}

	attrs := map[string]string{
		"user_id": extractUserID(u),
	}

	// Use reflection to get additional attributes
	v := reflect.ValueOf(u)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return attrs
	}

	// Extract permissions if available
	if permsField := v.FieldByName("Permissions"); permsField.IsValid() {
		if permsField.Kind() == reflect.Slice {
			var permissions []string
			for i := 0; i < permsField.Len(); i++ {
				perm := permsField.Index(i)
				if perm.Kind() == reflect.String {
					permissions = append(permissions, perm.String())
				}
			}
			attrs["permissions"] = strings.Join(permissions, ",")
		}
	}

	// Extract other common fields
	fields := []string{"Role", "Department", "Attributes"}
	for _, field := range fields {
		if f := v.FieldByName(field); f.IsValid() && f.Kind() == reflect.String {
			attrs[strings.ToLower(field)] = f.String()
		}
	}

	return attrs
}

func extractUserID(u *User) string {
	if u == nil {
		return ""
	}
	v := reflect.ValueOf(u)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return ""
	}

	candidates := []string{"ID", "UserID", "Id", "UserId"}
	for _, name := range candidates {
		f := v.FieldByName(name)
		if f.IsValid() {
			switch f.Kind() {
			case reflect.String:
				return f.String()
			case reflect.Int, reflect.Int64:
				return strconv.FormatInt(f.Int(), 10)
			case reflect.Int32:
				return strconv.FormatInt(f.Int(), 10)
			}
		}
	}
	return ""
}

// RequireABAC is a generic middleware wrapper that runs an ABAC check function.
func RequireABAC(abac *ABACPolicy, check func(a *ABACPolicy, u *User, r *http.Request) error) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, ok := UserFromContext(r.Context())
			if !ok || u == nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if err := check(abac, u, r); err != nil {
				if errors.Is(err, ErrForbidden) {
					http.Error(w, "forbidden", http.StatusForbidden)
					return
				}
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireCanViewExpense builds a middleware that checks if the authenticated user can view the expense.
func RequireCanViewExpense(db *sqlx.DB, abac *ABACPolicy) func(next http.Handler) http.Handler {
	return RequireABAC(abac, func(a *ABACPolicy, u *User, r *http.Request) error {
		idStr := chi.URLParam(r, "id")
		if idStr == "" {
			return ErrForbidden
		}
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return err
		}

		var ownerID int64
		err = db.GetContext(r.Context(), &ownerID, "SELECT user_id FROM expenses WHERE id=$1", id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrForbidden
			}
			return err
		}
		return a.CanViewExpense(u, ownerID)
	})
}

// RequireCanApproveExpense builds a middleware that checks if the user can approve expenses.
func RequireCanApproveExpense(db *sqlx.DB, abac *ABACPolicy) func(next http.Handler) http.Handler {
	return RequireABAC(abac, func(a *ABACPolicy, u *User, r *http.Request) error {
		idStr := chi.URLParam(r, "id")
		if idStr == "" {
			return ErrForbidden
		}
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return err
		}

		var ownerID int64
		err = db.GetContext(r.Context(), &ownerID, "SELECT user_id FROM expenses WHERE id=$1", id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrForbidden
			}
			return err
		}

		return a.CanApproveExpense(u, ownerID)
	})
}
