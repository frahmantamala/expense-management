package auth

// LoginDTO is the transport shape used by the HTTP handler to accept login requests.
type LoginDTO struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RefreshTokenDTO for refresh token requests
type RefreshTokenDTO struct {
	RefreshToken string `json:"refresh_token"`
}

// ValidationError represents a simple validation error from DTO validation.
type ValidationError struct {
	Msg string
}

func (v ValidationError) Error() string { return v.Msg }

// Validate checks required fields and returns a ValidationError on failure.
func (d LoginDTO) Validate() error {
	if d.Email == "" {
		return ValidationError{Msg: "email is required"}
	}
	if d.Password == "" {
		return ValidationError{Msg: "password is required"}
	}
	return nil
}

// Validate for refresh token DTO
func (d RefreshTokenDTO) Validate() error {
	if d.RefreshToken == "" {
		return ValidationError{Msg: "refresh_token is required"}
	}
	return nil
}
