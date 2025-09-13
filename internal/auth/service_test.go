package auth

import (
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"golang.org/x/crypto/bcrypt"
)

func TestAuth(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Auth Module Suite")
}

type mockUserRepository struct {
	users         map[string]string
	userIDs       map[string]string
	usersByID     map[int64]*User
	returnError   bool
	errorToReturn error
}

func newMockUserRepository() *mockUserRepository {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("correct_password"), bcrypt.DefaultCost)

	return &mockUserRepository{
		users: map[string]string{
			"user@example.com":    string(hashedPassword),
			"admin@example.com":   string(hashedPassword),
			"manager@example.com": string(hashedPassword),
		},
		userIDs: map[string]string{
			"user@example.com":    "1",
			"admin@example.com":   "2",
			"manager@example.com": "3",
		},
		usersByID: map[int64]*User{
			1: {ID: 1, Email: "user@example.com", Permissions: []string{"can_read_expense"}},
			2: {ID: 2, Email: "admin@example.com", Permissions: []string{"can_read_expense", "can_approve", "can_reject"}},
			3: {ID: 3, Email: "manager@example.com", Permissions: []string{"can_read_expense", "can_approve"}},
		},
	}
}

func (m *mockUserRepository) GetPasswordForUsername(username string) (string, string, error) {
	if m.returnError {
		return "", "", m.errorToReturn
	}

	if hash, exists := m.users[username]; exists {
		if userID, userExists := m.userIDs[username]; userExists {
			return hash, userID, nil
		}
	}
	return "", "", errors.New("user not found")
}

func (m *mockUserRepository) GetUserWithPermissions(userID int64) (*User, error) {
	if m.returnError {
		return nil, m.errorToReturn
	}

	if user, exists := m.usersByID[userID]; exists {
		return user, nil
	}
	return nil, errors.New("user not found")
}

func (m *mockUserRepository) setError(err error) {
	m.returnError = true
	m.errorToReturn = err
}

func (m *mockUserRepository) clearError() {
	m.returnError = false
	m.errorToReturn = nil
}

var _ = ginkgo.Describe("AuthService", func() {
	var (
		service       *Service
		mockRepo      *mockUserRepository
		tokenGen      *JWTTokenGenerator
		logger        *slog.Logger
		accessSecret  string        = "test-access-secret"
		refreshSecret string        = "test-refresh-secret"
		accessTTL     time.Duration = 15 * time.Minute
		refreshTTL    time.Duration = 24 * time.Hour
	)

	ginkgo.BeforeEach(func() {
		mockRepo = newMockUserRepository()
		tokenGen = NewJWTTokenGenerator(accessSecret, refreshSecret, accessTTL, refreshTTL)
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
		service = NewService(mockRepo, tokenGen, bcrypt.DefaultCost, logger)
	})

	ginkgo.Describe("Authenticate", func() {
		ginkgo.Context("when credentials are valid", func() {
			ginkgo.It("should return access and refresh tokens", func() {

				dto := LoginDTO{
					Email:    "user@example.com",
					Password: "correct_password",
				}

				tokens, err := service.Authenticate(dto)

				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(tokens.AccessToken).ToNot(gomega.BeEmpty())
				gomega.Expect(tokens.RefreshToken).ToNot(gomega.BeEmpty())
				gomega.Expect(tokens.AccessToken).ToNot(gomega.Equal(tokens.RefreshToken))
			})

			ginkgo.It("should generate valid JWT tokens", func() {

				dto := LoginDTO{
					Email:    "admin@example.com",
					Password: "correct_password",
				}

				tokens, err := service.Authenticate(dto)

				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				claims, err := service.ValidateAccessToken(tokens.AccessToken)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(claims.UserID).To(gomega.Equal("2"))
				gomega.Expect(claims.Email).To(gomega.Equal("admin@example.com"))
			})
		})

		ginkgo.Context("when credentials are invalid", func() {
			ginkgo.It("should return error for invalid email", func() {

				dto := LoginDTO{
					Email:    "nonexistent@example.com",
					Password: "any_password",
				}

				tokens, err := service.Authenticate(dto)

				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err).To(gomega.Equal(ErrInvalidCredentials))
				gomega.Expect(tokens.AccessToken).To(gomega.BeEmpty())
				gomega.Expect(tokens.RefreshToken).To(gomega.BeEmpty())
			})

			ginkgo.It("should return error for invalid password", func() {

				dto := LoginDTO{
					Email:    "user@example.com",
					Password: "wrong_password",
				}

				tokens, err := service.Authenticate(dto)

				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err).To(gomega.Equal(ErrInvalidCredentials))
				gomega.Expect(tokens.AccessToken).To(gomega.BeEmpty())
				gomega.Expect(tokens.RefreshToken).To(gomega.BeEmpty())
			})
		})

		ginkgo.Context("when input validation fails", func() {
			ginkgo.It("should return validation error for empty email", func() {

				dto := LoginDTO{
					Email:    "",
					Password: "password",
				}

				tokens, err := service.Authenticate(dto)

				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err.Error()).To(gomega.ContainSubstring("email is required"))
				gomega.Expect(tokens.AccessToken).To(gomega.BeEmpty())
			})

			ginkgo.It("should return validation error for empty password", func() {

				dto := LoginDTO{
					Email:    "user@example.com",
					Password: "",
				}

				tokens, err := service.Authenticate(dto)

				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err.Error()).To(gomega.ContainSubstring("password is required"))
				gomega.Expect(tokens.AccessToken).To(gomega.BeEmpty())
			})
		})

		ginkgo.Context("when repository returns error", func() {
			ginkgo.It("should return invalid credentials error", func() {

				mockRepo.setError(errors.New("database error"))
				dto := LoginDTO{
					Email:    "user@example.com",
					Password: "correct_password",
				}

				tokens, err := service.Authenticate(dto)

				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err).To(gomega.Equal(ErrInvalidCredentials))
				gomega.Expect(tokens.AccessToken).To(gomega.BeEmpty())
			})
		})
	})

	ginkgo.Describe("RefreshTokens", func() {
		var validRefreshToken string

		ginkgo.BeforeEach(func() {

			dto := LoginDTO{
				Email:    "user@example.com",
				Password: "correct_password",
			}
			tokens, err := service.Authenticate(dto)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			validRefreshToken = tokens.RefreshToken
		})

		ginkgo.Context("when refresh token is valid", func() {
			ginkgo.It("should return new access and refresh tokens", func() {

				time.Sleep(time.Millisecond)

				newTokens, err := service.RefreshTokens(validRefreshToken)

				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(newTokens.AccessToken).ToNot(gomega.BeEmpty())
				gomega.Expect(newTokens.RefreshToken).ToNot(gomega.BeEmpty())

			})

			ginkgo.It("should preserve user information in new tokens", func() {

				newTokens, err := service.RefreshTokens(validRefreshToken)

				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				claims, err := service.ValidateAccessToken(newTokens.AccessToken)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(claims.UserID).To(gomega.Equal("1"))
				gomega.Expect(claims.Email).To(gomega.Equal("user@example.com"))
			})
		})

		ginkgo.Context("when refresh token is invalid", func() {
			ginkgo.It("should return error for malformed token", func() {

				tokens, err := service.RefreshTokens("invalid.token.format")

				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(tokens.AccessToken).To(gomega.BeEmpty())
				gomega.Expect(tokens.RefreshToken).To(gomega.BeEmpty())
			})

			ginkgo.It("should return error for expired token", func() {

				expiredTokenGen := NewJWTTokenGenerator(accessSecret, refreshSecret, -1*time.Hour, -1*time.Hour)
				expiredToken, err := expiredTokenGen.GenerateRefreshToken("1", "user@example.com")
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				tokens, err := service.RefreshTokens(expiredToken)

				gomega.Expect(err).To(gomega.HaveOccurred())

				gomega.Expect(err).To(gomega.Or(gomega.Equal(ErrTokenExpired), gomega.Equal(ErrInvalidToken)))
				gomega.Expect(tokens.AccessToken).To(gomega.BeEmpty())
			})
		})
	})

	ginkgo.Describe("ValidateAccessToken", func() {
		var validAccessToken string

		ginkgo.BeforeEach(func() {

			dto := LoginDTO{
				Email:    "manager@example.com",
				Password: "correct_password",
			}
			tokens, err := service.Authenticate(dto)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			validAccessToken = tokens.AccessToken
		})

		ginkgo.Context("when access token is valid", func() {
			ginkgo.It("should return claims with user information", func() {

				claims, err := service.ValidateAccessToken(validAccessToken)

				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(claims).ToNot(gomega.BeNil())
				gomega.Expect(claims.UserID).To(gomega.Equal("3"))
				gomega.Expect(claims.Email).To(gomega.Equal("manager@example.com"))
				gomega.Expect(claims.ExpiresAt).ToNot(gomega.BeNil())
			})
		})

		ginkgo.Context("when access token is invalid", func() {
			ginkgo.It("should return error for malformed token", func() {

				claims, err := service.ValidateAccessToken("invalid.token")

				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(claims).To(gomega.BeNil())
			})

			ginkgo.It("should return error for empty token", func() {

				claims, err := service.ValidateAccessToken("")

				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(claims).To(gomega.BeNil())
			})

			ginkgo.It("should return error for expired token", func() {

				expiredTokenGen := NewJWTTokenGenerator(accessSecret, refreshSecret, -1*time.Hour, refreshTTL)
				expiredToken, err := expiredTokenGen.GenerateAccessToken("1", "user@example.com")
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				claims, err := service.ValidateAccessToken(expiredToken)

				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err).To(gomega.Equal(ErrTokenExpired))
				gomega.Expect(claims).To(gomega.BeNil())
			})
		})
	})

	ginkgo.Describe("GetUserWithPermissions", func() {
		ginkgo.Context("when user exists", func() {
			ginkgo.It("should return user with permissions", func() {

				user, err := service.GetUserWithPermissions(2)

				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(user).ToNot(gomega.BeNil())
				gomega.Expect(user.ID).To(gomega.Equal(int64(2)))
				gomega.Expect(user.Email).To(gomega.Equal("admin@example.com"))
				gomega.Expect(user.Permissions).To(gomega.ContainElements("can_read_expense", "can_approve", "can_reject"))
			})
		})

		ginkgo.Context("when user does not exist", func() {
			ginkgo.It("should return error", func() {

				user, err := service.GetUserWithPermissions(999)

				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(user).To(gomega.BeNil())
			})
		})

		ginkgo.Context("when repository returns error", func() {
			ginkgo.It("should return repository error", func() {

				mockRepo.setError(errors.New("database error"))

				user, err := service.GetUserWithPermissions(1)

				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err.Error()).To(gomega.Equal("database error"))
				gomega.Expect(user).To(gomega.BeNil())
			})
		})
	})

	ginkgo.Describe("HashPassword", func() {
		ginkgo.Context("when password is valid", func() {
			ginkgo.It("should return hashed password", func() {

				password := "test_password_123"

				hash, err := service.HashPassword(password)

				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(hash).ToNot(gomega.BeEmpty())
				gomega.Expect(hash).ToNot(gomega.Equal(password))

				err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
			})

			ginkgo.It("should generate different hashes for same password", func() {

				password := "same_password"

				hash1, err1 := service.HashPassword(password)
				hash2, err2 := service.HashPassword(password)

				gomega.Expect(err1).ToNot(gomega.HaveOccurred())
				gomega.Expect(err2).ToNot(gomega.HaveOccurred())
				gomega.Expect(hash1).ToNot(gomega.Equal(hash2))
			})
		})
	})
})

var _ = ginkgo.Describe("JWTTokenGenerator", func() {
	var (
		tokenGen      *JWTTokenGenerator
		accessSecret  string        = "test-access-secret-key"
		refreshSecret string        = "test-refresh-secret-key"
		accessTTL     time.Duration = 15 * time.Minute
		refreshTTL    time.Duration = 24 * time.Hour
	)

	ginkgo.BeforeEach(func() {
		tokenGen = NewJWTTokenGenerator(accessSecret, refreshSecret, accessTTL, refreshTTL)
	})

	ginkgo.Describe("GenerateAccessToken", func() {
		ginkgo.It("should generate valid access token", func() {

			userID := "123"
			email := "test@example.com"

			token, err := tokenGen.GenerateAccessToken(userID, email)

			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(token).ToNot(gomega.BeEmpty())

			claims, err := tokenGen.ValidateToken(token)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(claims.UserID).To(gomega.Equal(userID))
			gomega.Expect(claims.Email).To(gomega.Equal(email))
		})
	})

	ginkgo.Describe("GenerateRefreshToken", func() {
		ginkgo.It("should generate valid refresh token", func() {

			userID := "456"
			email := "refresh@example.com"

			token, err := tokenGen.GenerateRefreshToken(userID, email)

			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(token).ToNot(gomega.BeEmpty())

			claims, err := tokenGen.ValidateToken(token)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(claims.UserID).To(gomega.Equal(userID))
			gomega.Expect(claims.Email).To(gomega.Equal(email))
		})
	})

	ginkgo.Describe("ValidateToken", func() {
		ginkgo.Context("with valid access token", func() {
			ginkgo.It("should return valid claims", func() {

				userID := "789"
				email := "validate@example.com"
				token, err := tokenGen.GenerateAccessToken(userID, email)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				claims, err := tokenGen.ValidateToken(token)

				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(claims.UserID).To(gomega.Equal(userID))
				gomega.Expect(claims.Email).To(gomega.Equal(email))
				gomega.Expect(claims.ExpiresAt.Time).To(gomega.BeTemporally("~", time.Now().Add(accessTTL), time.Minute))
			})
		})

		ginkgo.Context("with valid refresh token", func() {
			ginkgo.It("should return valid claims", func() {

				userID := "101"
				email := "refresh-validate@example.com"
				token, err := tokenGen.GenerateRefreshToken(userID, email)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				claims, err := tokenGen.ValidateToken(token)

				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(claims.UserID).To(gomega.Equal(userID))
				gomega.Expect(claims.Email).To(gomega.Equal(email))
				gomega.Expect(claims.ExpiresAt.Time).To(gomega.BeTemporally("~", time.Now().Add(refreshTTL), time.Minute))
			})
		})

		ginkgo.Context("with invalid token", func() {
			ginkgo.It("should return error for malformed token", func() {

				claims, err := tokenGen.ValidateToken("invalid.token.here")

				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(claims).To(gomega.BeNil())
			})

			ginkgo.It("should return error for empty token", func() {

				claims, err := tokenGen.ValidateToken("")

				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(claims).To(gomega.BeNil())
			})
		})

		ginkgo.Context("with expired token", func() {
			ginkgo.It("should return ErrTokenExpired", func() {

				expiredGen := NewJWTTokenGenerator(accessSecret, refreshSecret, -1*time.Hour, -1*time.Hour)
				token, err := expiredGen.GenerateAccessToken("123", "expired@example.com")
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				claims, err := tokenGen.ValidateToken(token)

				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err).To(gomega.Equal(ErrTokenExpired))
				gomega.Expect(claims).To(gomega.BeNil())
			})
		})
	})
})
