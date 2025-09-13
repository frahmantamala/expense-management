package user

import (
	"errors"
	"time"

	userDatamodel "github.com/frahmantamala/expense-management/internal/core/datamodel/user"
)

type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	PasswordHash string    `json:"-"`
	Department   string    `json:"department"`
	IsActive     bool      `json:"is_active"`
	Permissions  []string  `json:"permissions,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (u *User) HasPermission(permission string) bool {
	for _, p := range u.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

func (u *User) HasAnyPermission(permissions []string) bool {
	for _, userPerm := range u.Permissions {
		for _, requiredPerm := range permissions {
			if userPerm == requiredPerm {
				return true
			}
		}
	}
	return false
}

func (u *User) IsManager() bool {
	managerPerms := []string{"approve_expenses", "reject_expenses", "admin"}
	return u.HasAnyPermission(managerPerms)
}

func (u *User) IsAdmin() bool {
	return u.HasPermission("admin")
}

func (u *User) IsActiveUser() bool {
	return u.IsActive
}

type Permission struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type UserPermission struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"user_id"`
	PermissionID int64     `json:"permission_id"`
	GrantedBy    *int64    `json:"granted_by,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

var ErrNotFound = errors.New("user not found")

func ToDataModel(u *User) *userDatamodel.User {
	return &userDatamodel.User{
		ID:           u.ID,
		Email:        u.Email,
		Name:         u.Name,
		PasswordHash: u.PasswordHash,
		Department:   u.Department,
		IsActive:     u.IsActive,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}

func FromDataModel(u *userDatamodel.User) *User {
	return &User{
		ID:           u.ID,
		Email:        u.Email,
		Name:         u.Name,
		PasswordHash: u.PasswordHash,
		Department:   u.Department,
		IsActive:     u.IsActive,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
		Permissions:  []string{},
	}
}

func FromDataModelWithPermissions(u *userDatamodel.User, permissions []string) *User {
	domainUser := FromDataModel(u)
	domainUser.Permissions = permissions
	return domainUser
}
