package oauth

import (
	"context"

	"go.uber.org/fx"
	"google.golang.org/api/idtoken"
)

// GoogleOAuthClient is the interface for Google ID token validation.
type GoogleOAuthClient interface {
	Validate(ctx context.Context, idToken, audience string) (*idtoken.Payload, error)
}

// googleOAuthClient is the concrete implementation wrapping the Google ID token validator.
type googleOAuthClient struct {
	*idtoken.Validator
}

type GoogleOAuthParams struct {
	fx.In
	Ctx context.Context
}

// NewGoogleOAuthClient creates a Google ID token validator
func NewGoogleOAuthClient(p GoogleOAuthParams) (GoogleOAuthClient, error) {
	validator, err := idtoken.NewValidator(p.Ctx)
	if err != nil {
		return nil, err
	}

	return &googleOAuthClient{Validator: validator}, nil
}

// ExtractEmailFromGoogleIdToken extracts email from Google ID token payload
func ExtractEmailFromGoogleIdToken(payload *idtoken.Payload) string {
	email, ok := payload.Claims["email"].(string)
	if !ok || email == "" {
		return ""
	}
	return email
}
