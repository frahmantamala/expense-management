package auth

type LoginDTO struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshTokenDTO struct {
	RefreshToken string `json:"refresh_token"`
}

type ValidationError struct {
	Msg string
}

func (v ValidationError) Error() string { return v.Msg }

func (d LoginDTO) Validate() error {
	if d.Email == "" {
		return ValidationError{Msg: "email is required"}
	}
	if d.Password == "" {
		return ValidationError{Msg: "password is required"}
	}
	return nil
}

func (d RefreshTokenDTO) Validate() error {
	if d.RefreshToken == "" {
		return ValidationError{Msg: "refresh_token is required"}
	}
	return nil
}
