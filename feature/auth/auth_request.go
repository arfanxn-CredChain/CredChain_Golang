package auth

import validation "github.com/go-ozzo/ozzo-validation/v4"

type AuthGoogleLoginRequest struct {
	IdToken string `json:"id_token"`
}

func (r AuthGoogleLoginRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.IdToken, validation.Required),
	)
}

type AuthRefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (r AuthRefreshRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.RefreshToken, validation.Required),
	)
}
