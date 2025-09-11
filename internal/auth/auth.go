package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// AuthInfo is the internal domain model used by services and converters.
type AuthInfo struct {
	UserID    string
	Token     string
	ExpiresAt time.Time
}

type User struct {
	ID          int64    `json:"id"`
	Email       string   `json:"email"`
	Permissions []string `json:"permissions,omitempty"`
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

// TokenGenerator creates tokens and expiration times.
type TokenGenerator interface {
	GenerateAccessToken(userID string, email string) (token string, err error)
	GenerateRefreshToken(userID string, email string) (token string, err error)
	ValidateToken(tokenString string) (*Claims, error)
}

// AuthService performs authentication-related business logic.
type AuthService interface {
	Authenticate(dto LoginDTO) (AuthTokens, error)
	RefreshTokens(refreshToken string) (AuthTokens, error)
	ValidateAccessToken(tokenString string) (*Claims, error)
}

type AuthTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Claims represents JWT token claims
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

// ToV1 converts internal AuthInfo domain model to API-ready view model.
func (a AuthInfo) ToV1() AuthResponseV1 {
	return AuthResponseV1{
		ID:    a.UserID,
		Token: a.Token,
	}
}
