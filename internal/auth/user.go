package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type ServiceAPI interface {
	Authenticate(dto LoginDTO) (AuthTokens, error)
	RefreshTokens(refreshToken string) (AuthTokens, error)
	ValidateAccessToken(tokenString string) (*Claims, error)
	GetUserWithPermissions(userID int64) (*User, error)
	HashPassword(password string) (string, error)
}

type RepositoryAPI interface {
	GetPasswordForUsername(username string) (passwordHash string, userID string, err error)
	GetUserWithPermissions(userID int64) (*User, error)
}

type TokenGeneratorAPI interface {
	GenerateAccessToken(userID string, email string) (token string, err error)
	GenerateRefreshToken(userID string, email string) (token string, err error)
	ValidateToken(tokenString string) (*Claims, error)
}

type User struct {
	ID          int64    `json:"id"`
	Email       string   `json:"email"`
	Permissions []string `json:"permissions,omitempty"`
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

type AuthInfo struct {
	UserID    string
	Token     string
	ExpiresAt time.Time
}

type AuthResponseV1 struct {
	ID    string `json:"id"`
	Token string `json:"token"`
}

type UserInfo struct {
	ID          int64     `db:"id"`
	Email       string    `db:"email"`
	Name        string    `db:"name"`
	Department  string    `db:"department"`
	IsActive    bool      `db:"is_active"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
	Permissions []string  `db:"-"`
}

type AuthTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

type JWTTokenGenerator struct {
	AccessTokenSecret  []byte
	RefreshTokenSecret []byte
	AccessTokenTTL     time.Duration
	RefreshTokenTTL    time.Duration
}

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
	ErrUserInactive       = errors.New("user is inactive")
)

func (a AuthInfo) ToV1() AuthResponseV1 {
	return AuthResponseV1{
		ID:    a.UserID,
		Token: a.Token,
	}
}

func VerifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

func HashPassword(password string, cost int) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
