package user

import "time"

type User struct {
	ID           int64
	Email        string
	Name         string
	PasswordHash string
	Department   string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Permission struct {
	ID          int64
	Name        string
	Description string
	CreatedAt   time.Time
}

type UserPermission struct {
	ID           int64
	UserID       int64
	PermissionID int64
	GrantedBy    *int64
	CreatedAt    time.Time
}
