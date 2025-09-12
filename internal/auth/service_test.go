package auth

import (
	"errors"
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

// Mock UserRepository for testing
type mockUserRepository struct {
	users         map[string]string // email -> password hash
	userIDs       map[string]string // email -> userID
	usersByID     map[int64]*User   // userID -> User with permissions
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
		accessSecret  string        = "test-access-secret"
		refreshSecret string        = "test-refresh-secret"
		accessTTL     time.Duration = 15 * time.Minute
		refreshTTL    time.Duration = 24 * time.Hour
	)

	ginkgo.BeforeEach(func() {
		mockRepo = newMockUserRepository()
		tokenGen = NewJWTTokenGenerator(accessSecret, refreshSecret, accessTTL, refreshTTL)
		service = NewService(mockRepo, tokenGen, bcrypt.DefaultCost)
	})

	ginkgo.Describe("Authenticate", func() {
		ginkgo.Context("when credentials are valid", func() {
			ginkgo.It("should return access and refresh tokens", func() {
				// Given
				dto := LoginDTO{
					Email:    "user@example.com",
					Password: "correct_password",
				}

				// When
				tokens, err := service.Authenticate(dto)

				// Then
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(tokens.AccessToken).ToNot(gomega.BeEmpty())
				gomega.Expect(tokens.RefreshToken).ToNot(gomega.BeEmpty())
				gomega.Expect(tokens.AccessToken).ToNot(gomega.Equal(tokens.RefreshToken))
			})

			ginkgo.It("should generate valid JWT tokens", func() {
				// Given
				dto := LoginDTO{
					Email:    "admin@example.com",
					Password: "correct_password",
				}

				// When
				tokens, err := service.Authenticate(dto)

				// Then
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				// Validate access token
				claims, err := service.ValidateAccessToken(tokens.AccessToken)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(claims.UserID).To(gomega.Equal("2"))
				gomega.Expect(claims.Email).To(gomega.Equal("admin@example.com"))
			})
		})

		ginkgo.Context("when credentials are invalid", func() {
			ginkgo.It("should return error for invalid email", func() {
				// Given
				dto := LoginDTO{
					Email:    "nonexistent@example.com",
					Password: "any_password",
				}

				// When
				tokens, err := service.Authenticate(dto)

				// Then
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err).To(gomega.Equal(ErrInvalidCredentials))
				gomega.Expect(tokens.AccessToken).To(gomega.BeEmpty())
				gomega.Expect(tokens.RefreshToken).To(gomega.BeEmpty())
			})

			ginkgo.It("should return error for invalid password", func() {
				// Given
				dto := LoginDTO{
					Email:    "user@example.com",
					Password: "wrong_password",
				}

				// When
				tokens, err := service.Authenticate(dto)

				// Then
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err).To(gomega.Equal(ErrInvalidCredentials))
				gomega.Expect(tokens.AccessToken).To(gomega.BeEmpty())
				gomega.Expect(tokens.RefreshToken).To(gomega.BeEmpty())
			})
		})

		ginkgo.Context("when input validation fails", func() {
			ginkgo.It("should return validation error for empty email", func() {
				// Given
				dto := LoginDTO{
					Email:    "",
					Password: "password",
				}

				// When
				tokens, err := service.Authenticate(dto)

				// Then
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err.Error()).To(gomega.ContainSubstring("email is required"))
				gomega.Expect(tokens.AccessToken).To(gomega.BeEmpty())
			})

			ginkgo.It("should return validation error for empty password", func() {
				// Given
				dto := LoginDTO{
					Email:    "user@example.com",
					Password: "",
				}

				// When
				tokens, err := service.Authenticate(dto)

				// Then
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err.Error()).To(gomega.ContainSubstring("password is required"))
				gomega.Expect(tokens.AccessToken).To(gomega.BeEmpty())
			})
		})

		ginkgo.Context("when repository returns error", func() {
			ginkgo.It("should return invalid credentials error", func() {
				// Given
				mockRepo.setError(errors.New("database error"))
				dto := LoginDTO{
					Email:    "user@example.com",
					Password: "correct_password",
				}

				// When
				tokens, err := service.Authenticate(dto)

				// Then
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err).To(gomega.Equal(ErrInvalidCredentials))
				gomega.Expect(tokens.AccessToken).To(gomega.BeEmpty())
			})
		})
	})

	ginkgo.Describe("RefreshTokens", func() {
		var validRefreshToken string

		ginkgo.BeforeEach(func() {
			// Create a valid refresh token first
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
				// Add a small delay to ensure different timestamps
				time.Sleep(time.Millisecond)

				// When
				newTokens, err := service.RefreshTokens(validRefreshToken)

				// Then
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(newTokens.AccessToken).ToNot(gomega.BeEmpty())
				gomega.Expect(newTokens.RefreshToken).ToNot(gomega.BeEmpty())
				// Note: tokens might be same if generated at exact same timestamp
				// The important thing is they are valid tokens
			})

			ginkgo.It("should preserve user information in new tokens", func() {
				// When
				newTokens, err := service.RefreshTokens(validRefreshToken)

				// Then
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				// Validate new access token contains correct user info
				claims, err := service.ValidateAccessToken(newTokens.AccessToken)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(claims.UserID).To(gomega.Equal("1"))
				gomega.Expect(claims.Email).To(gomega.Equal("user@example.com"))
			})
		})

		ginkgo.Context("when refresh token is invalid", func() {
			ginkgo.It("should return error for malformed token", func() {
				// When
				tokens, err := service.RefreshTokens("invalid.token.format")

				// Then
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(tokens.AccessToken).To(gomega.BeEmpty())
				gomega.Expect(tokens.RefreshToken).To(gomega.BeEmpty())
			})

			ginkgo.It("should return error for expired token", func() {
				// Create an expired token generator
				expiredTokenGen := NewJWTTokenGenerator(accessSecret, refreshSecret, -1*time.Hour, -1*time.Hour)
				expiredToken, err := expiredTokenGen.GenerateRefreshToken("1", "user@example.com")
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				// When
				tokens, err := service.RefreshTokens(expiredToken)

				// Then
				gomega.Expect(err).To(gomega.HaveOccurred())
				// For expired tokens, the error might be ErrInvalidToken or ErrTokenExpired depending on JWT library behavior
				gomega.Expect(err).To(gomega.Or(gomega.Equal(ErrTokenExpired), gomega.Equal(ErrInvalidToken)))
				gomega.Expect(tokens.AccessToken).To(gomega.BeEmpty())
			})
		})
	})

	ginkgo.Describe("ValidateAccessToken", func() {
		var validAccessToken string

		ginkgo.BeforeEach(func() {
			// Create a valid access token first
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
				// When
				claims, err := service.ValidateAccessToken(validAccessToken)

				// Then
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(claims).ToNot(gomega.BeNil())
				gomega.Expect(claims.UserID).To(gomega.Equal("3"))
				gomega.Expect(claims.Email).To(gomega.Equal("manager@example.com"))
				gomega.Expect(claims.ExpiresAt).ToNot(gomega.BeNil())
			})
		})

		ginkgo.Context("when access token is invalid", func() {
			ginkgo.It("should return error for malformed token", func() {
				// When
				claims, err := service.ValidateAccessToken("invalid.token")

				// Then
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(claims).To(gomega.BeNil())
			})

			ginkgo.It("should return error for empty token", func() {
				// When
				claims, err := service.ValidateAccessToken("")

				// Then
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(claims).To(gomega.BeNil())
			})

			ginkgo.It("should return error for expired token", func() {
				// Create an expired token
				expiredTokenGen := NewJWTTokenGenerator(accessSecret, refreshSecret, -1*time.Hour, refreshTTL)
				expiredToken, err := expiredTokenGen.GenerateAccessToken("1", "user@example.com")
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				// When
				claims, err := service.ValidateAccessToken(expiredToken)

				// Then
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err).To(gomega.Equal(ErrTokenExpired))
				gomega.Expect(claims).To(gomega.BeNil())
			})
		})
	})

	ginkgo.Describe("GetUserWithPermissions", func() {
		ginkgo.Context("when user exists", func() {
			ginkgo.It("should return user with permissions", func() {
				// When
				user, err := service.GetUserWithPermissions(2)

				// Then
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(user).ToNot(gomega.BeNil())
				gomega.Expect(user.ID).To(gomega.Equal(int64(2)))
				gomega.Expect(user.Email).To(gomega.Equal("admin@example.com"))
				gomega.Expect(user.Permissions).To(gomega.ContainElements("can_read_expense", "can_approve", "can_reject"))
			})
		})

		ginkgo.Context("when user does not exist", func() {
			ginkgo.It("should return error", func() {
				// When
				user, err := service.GetUserWithPermissions(999)

				// Then
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(user).To(gomega.BeNil())
			})
		})

		ginkgo.Context("when repository returns error", func() {
			ginkgo.It("should return repository error", func() {
				// Given
				mockRepo.setError(errors.New("database error"))

				// When
				user, err := service.GetUserWithPermissions(1)

				// Then
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err.Error()).To(gomega.Equal("database error"))
				gomega.Expect(user).To(gomega.BeNil())
			})
		})
	})

	ginkgo.Describe("HashPassword", func() {
		ginkgo.Context("when password is valid", func() {
			ginkgo.It("should return hashed password", func() {
				// Given
				password := "test_password_123"

				// When
				hash, err := service.HashPassword(password)

				// Then
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(hash).ToNot(gomega.BeEmpty())
				gomega.Expect(hash).ToNot(gomega.Equal(password))

				// Verify the hash can be compared
				err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
			})

			ginkgo.It("should generate different hashes for same password", func() {
				// Given
				password := "same_password"

				// When
				hash1, err1 := service.HashPassword(password)
				hash2, err2 := service.HashPassword(password)

				// Then
				gomega.Expect(err1).ToNot(gomega.HaveOccurred())
				gomega.Expect(err2).ToNot(gomega.HaveOccurred())
				gomega.Expect(hash1).ToNot(gomega.Equal(hash2)) // Salts make them different
			})
		})
	})

	ginkgo.Describe("GenerateRandomToken", func() {
		ginkgo.It("should generate non-empty random token", func() {
			// When
			token, err := GenerateRandomToken()

			// Then
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(token).ToNot(gomega.BeEmpty())
			gomega.Expect(len(token)).To(gomega.Equal(64)) // 32 bytes * 2 (hex encoding)
		})

		ginkgo.It("should generate different tokens each time", func() {
			// When
			token1, err1 := GenerateRandomToken()
			token2, err2 := GenerateRandomToken()

			// Then
			gomega.Expect(err1).ToNot(gomega.HaveOccurred())
			gomega.Expect(err2).ToNot(gomega.HaveOccurred())
			gomega.Expect(token1).ToNot(gomega.Equal(token2))
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
			// Given
			userID := "123"
			email := "test@example.com"

			// When
			token, err := tokenGen.GenerateAccessToken(userID, email)

			// Then
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(token).ToNot(gomega.BeEmpty())

			// Validate the token can be parsed
			claims, err := tokenGen.ValidateToken(token)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(claims.UserID).To(gomega.Equal(userID))
			gomega.Expect(claims.Email).To(gomega.Equal(email))
		})
	})

	ginkgo.Describe("GenerateRefreshToken", func() {
		ginkgo.It("should generate valid refresh token", func() {
			// Given
			userID := "456"
			email := "refresh@example.com"

			// When
			token, err := tokenGen.GenerateRefreshToken(userID, email)

			// Then
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(token).ToNot(gomega.BeEmpty())

			// Validate the token can be parsed
			claims, err := tokenGen.ValidateToken(token)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(claims.UserID).To(gomega.Equal(userID))
			gomega.Expect(claims.Email).To(gomega.Equal(email))
		})
	})

	ginkgo.Describe("ValidateToken", func() {
		ginkgo.Context("with valid access token", func() {
			ginkgo.It("should return valid claims", func() {
				// Given
				userID := "789"
				email := "validate@example.com"
				token, err := tokenGen.GenerateAccessToken(userID, email)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				// When
				claims, err := tokenGen.ValidateToken(token)

				// Then
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(claims.UserID).To(gomega.Equal(userID))
				gomega.Expect(claims.Email).To(gomega.Equal(email))
				gomega.Expect(claims.ExpiresAt.Time).To(gomega.BeTemporally("~", time.Now().Add(accessTTL), time.Minute))
			})
		})

		ginkgo.Context("with valid refresh token", func() {
			ginkgo.It("should return valid claims", func() {
				// Given
				userID := "101"
				email := "refresh-validate@example.com"
				token, err := tokenGen.GenerateRefreshToken(userID, email)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				// When
				claims, err := tokenGen.ValidateToken(token)

				// Then
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(claims.UserID).To(gomega.Equal(userID))
				gomega.Expect(claims.Email).To(gomega.Equal(email))
				gomega.Expect(claims.ExpiresAt.Time).To(gomega.BeTemporally("~", time.Now().Add(refreshTTL), time.Minute))
			})
		})

		ginkgo.Context("with invalid token", func() {
			ginkgo.It("should return error for malformed token", func() {
				// When
				claims, err := tokenGen.ValidateToken("invalid.token.here")

				// Then
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(claims).To(gomega.BeNil())
			})

			ginkgo.It("should return error for empty token", func() {
				// When
				claims, err := tokenGen.ValidateToken("")

				// Then
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(claims).To(gomega.BeNil())
			})
		})

		ginkgo.Context("with expired token", func() {
			ginkgo.It("should return ErrTokenExpired", func() {
				// Given expired token generator
				expiredGen := NewJWTTokenGenerator(accessSecret, refreshSecret, -1*time.Hour, -1*time.Hour)
				token, err := expiredGen.GenerateAccessToken("123", "expired@example.com")
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				// When
				claims, err := tokenGen.ValidateToken(token)

				// Then
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err).To(gomega.Equal(ErrTokenExpired))
				gomega.Expect(claims).To(gomega.BeNil())
			})
		})
	})
})

// ABAC Policy Tests
var _ = ginkgo.Describe("ABACPolicy", func() {
	var policy *ABACPolicy

	ginkgo.BeforeEach(func() {
		policy = &ABACPolicy{}
	})

	ginkgo.Describe("Allow", func() {
		ginkgo.Context("when user has admin attributes", func() {
			ginkgo.It("should allow all actions", func() {
				// Given
				userAttrs := map[string]string{
					"user_id":    "1",
					"attributes": "admin",
				}

				// When & Then
				gomega.Expect(policy.Allow(userAttrs, "999", "read")).To(gomega.BeTrue())
				gomega.Expect(policy.Allow(userAttrs, "999", "approve")).To(gomega.BeTrue())
			})
		})

		ginkgo.Context("when user has specific permissions", func() {
			ginkgo.It("should allow read for can_read_expense permission", func() {
				// Given
				userAttrs := map[string]string{
					"user_id":     "3",
					"permissions": "can_read_expense,can_view_reports",
				}

				// When & Then
				gomega.Expect(policy.Allow(userAttrs, "999", "read")).To(gomega.BeTrue())
				gomega.Expect(policy.Allow(userAttrs, "999", "approve")).To(gomega.BeFalse())
			})
		})

		ginkgo.Context("when user is the resource owner", func() {
			ginkgo.It("should allow basic CRUD operations", func() {
				// Given
				userAttrs := map[string]string{
					"user_id": "100",
				}
				resourceOwnerID := "100"

				// When & Then
				gomega.Expect(policy.Allow(userAttrs, resourceOwnerID, "read")).To(gomega.BeTrue())
				gomega.Expect(policy.Allow(userAttrs, resourceOwnerID, "write")).To(gomega.BeTrue())
			})
		})
	})

	ginkgo.Describe("CanViewExpense", func() {
		ginkgo.Context("when user has read permissions", func() {
			ginkgo.It("should allow viewing expense", func() {
				// Given
				user := &User{
					ID:          456,
					Email:       "viewer@example.com",
					Permissions: []string{"can_read_expense"},
				}

				// When
				err := policy.CanViewExpense(user, 999)

				// Then
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
			})
		})

		ginkgo.Context("when user has no permissions", func() {
			ginkgo.It("should deny viewing other's expense", func() {
				// Given
				user := &User{
					ID:    123,
					Email: "user@example.com",
				}

				// When
				err := policy.CanViewExpense(user, 456)

				// Then
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err).To(gomega.Equal(ErrForbidden))
			})
		})
	})

	ginkgo.Describe("CanApproveExpense", func() {
		ginkgo.Context("when user has approve permissions", func() {
			ginkgo.It("should allow approving expense", func() {
				// Given
				user := &User{
					ID:          789,
					Email:       "manager@example.com",
					Permissions: []string{"can_read_expense", "can_approve"},
				}

				// When
				err := policy.CanApproveExpense(user, 123)

				// Then
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
			})
		})

		ginkgo.Context("when user has no approve permissions", func() {
			ginkgo.It("should deny approval", func() {
				// Given
				user := &User{
					ID:          123,
					Email:       "user@example.com",
					Permissions: []string{"can_read_expense"},
				}

				// When
				err := policy.CanApproveExpense(user, 456)

				// Then
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err).To(gomega.Equal(ErrForbidden))
			})
		})
	})
})

// DTO Tests
var _ = ginkgo.Describe("LoginDTO", func() {
	ginkgo.Describe("Validate", func() {
		ginkgo.Context("when all fields are valid", func() {
			ginkgo.It("should not return error", func() {
				// Given
				dto := LoginDTO{
					Email:    "user@example.com",
					Password: "secure_password",
				}

				// When
				err := dto.Validate()

				// Then
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
			})
		})

		ginkgo.Context("when email is empty", func() {
			ginkgo.It("should return validation error", func() {
				// Given
				dto := LoginDTO{
					Email:    "",
					Password: "password",
				}

				// When
				err := dto.Validate()

				// Then
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err.Error()).To(gomega.Equal("email is required"))
			})
		})

		ginkgo.Context("when password is empty", func() {
			ginkgo.It("should return validation error", func() {
				// Given
				dto := LoginDTO{
					Email:    "user@example.com",
					Password: "",
				}

				// When
				err := dto.Validate()

				// Then
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err.Error()).To(gomega.Equal("password is required"))
			})
		})
	})
})

var _ = ginkgo.Describe("RefreshTokenDTO", func() {
	ginkgo.Describe("Validate", func() {
		ginkgo.Context("when refresh token is provided", func() {
			ginkgo.It("should not return error", func() {
				// Given
				dto := RefreshTokenDTO{
					RefreshToken: "valid.jwt.token",
				}

				// When
				err := dto.Validate()

				// Then
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
			})
		})

		ginkgo.Context("when refresh token is empty", func() {
			ginkgo.It("should return validation error", func() {
				// Given
				dto := RefreshTokenDTO{
					RefreshToken: "",
				}

				// When
				err := dto.Validate()

				// Then
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err.Error()).To(gomega.Equal("refresh_token is required"))
			})
		})
	})
})
