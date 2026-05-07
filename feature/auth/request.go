package auth

import validation "github.com/go-ozzo/ozzo-validation/v4"

// GoogleLoginRequest represents the incoming Google OAuth payload
type GoogleLoginRequest struct {
	IdToken string `json:"id_token"`
}

// Validate performs structural validation
func (r GoogleLoginRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.IdToken, validation.Required),
	)
}

// RefreshRequest represents the refresh token payload
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Validate performs structural validation
func (r RefreshRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.RefreshToken, validation.Required),
	)
}
