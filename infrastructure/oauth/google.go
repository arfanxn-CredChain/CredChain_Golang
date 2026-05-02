package oauth

import (
	"context"

	"go.uber.org/fx"
	"google.golang.org/api/idtoken"
)

// GoogleOAuthClient wraps the Google ID token validator
type GoogleOAuthClient struct {
	*idtoken.Validator
}

type GoogleOAuthParams struct {
	fx.In
	Ctx context.Context
}

// NewGoogleOAuthClient creates a Google ID token validator
func NewGoogleOAuthClient(p GoogleOAuthParams) (*GoogleOAuthClient, error) {
	validator, err := idtoken.NewValidator(p.Ctx)
	if err != nil {
		return nil, err
	}

	return &GoogleOAuthClient{Validator: validator}, nil
}

// ExtractEmailFromGoogleIdToken extracts email from Google ID token payload
func ExtractEmailFromGoogleIdToken(payload *idtoken.Payload) string {
	email, ok := payload.Claims["email"].(string)
	if !ok || email == "" {
		return ""
	}
	return email
}
