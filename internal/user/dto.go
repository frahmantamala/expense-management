package user

import "time"

// User represents the internal user model
type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	PasswordHash string    `json:"-"` // Never expose password hash
	Department   string    `json:"department"`
	IsActive     bool      `json:"is_active"`
	Permissions  []string  `json:"permissions,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Permission represents a system permission
type Permission struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// UserPermission represents the relationship between users and permissions
type UserPermission struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"user_id"`
	PermissionID int64     `json:"permission_id"`
	GrantedBy    *int64    `json:"granted_by,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}
