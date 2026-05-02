package auth

import (
	"context"
	"time"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/crypto"
	"CredChain_Golang/infrastructure/oauth"
	"CredChain_Golang/infrastructure/security"

	"github.com/oklog/ulid/v2"
	"go.uber.org/fx"
)

// Service handles authentication business logic
type Service struct {
	uow         domain.UnitOfWork
	oauthClient *oauth.GoogleOAuthClient
	config      *config.Config
}

type AuthServiceParams struct {
	fx.In
	UoW         domain.UnitOfWork
	OAuthClient *oauth.GoogleOAuthClient
	Config      *config.Config
}

// NewAuthService creates a new auth service
func NewAuthService(p AuthServiceParams) *Service {
	return &Service{
		uow:         p.UoW,
		oauthClient: p.OAuthClient,
		config:      p.Config,
	}
}

// GoogleLogin validates a Google ID token and issues a new access/refresh token pair.
// Returns the authenticated user, the stored refresh token entity, the JWT access token string, and any error.
func (s *Service) GoogleLogin(ctx context.Context, idToken string) (domain.User, domain.UserToken, string, error) {
	// 1. Validate Google ID token
	payload, err := s.oauthClient.Validate(ctx, idToken, "")
	if err != nil {
		return domain.User{}, domain.UserToken{}, "", domain.NewError(domain.CodeAuthGoogleLoginInvalidToken, domain.WithError(err))
	}

	// 2. Extract email
	email := oauth.ExtractEmailFromGoogleIdToken(payload)
	if email == "" {
		return domain.User{}, domain.UserToken{}, "", domain.NewError(domain.CodeAuthGoogleLoginInvalidToken)
	}

	// 3. Find user by email (invite-only system)
	users, err := s.uow.User().FindByEmails(ctx, email)
	if err != nil || len(users) == 0 {
		return domain.User{}, domain.UserToken{}, "", domain.NewError(domain.CodeAuthGoogleLoginUserNotFound)
	}
	user := users[0]

	// 4. Generate access token
	accessToken, err := security.GenerateJWT(user.Id, []byte(s.config.JWTSecret), time.Duration(s.config.JWTAccessExpiryMinutes)*time.Minute)
	if err != nil {
		return domain.User{}, domain.UserToken{}, "", domain.NewError(domain.CodeAuthGoogleLoginJWTFailed, domain.WithError(err))
	}

	// 5. Generate refresh token
	refreshTokenStr := crypto.MustGenerateRandomToken()
	expiresAt := time.Now().Add(time.Duration(s.config.JWTRefreshExpiryHours) * time.Hour)

	// 6. Store refresh token in DB
	refreshToken := domain.UserToken{
		Id:        ulid.MustNew(ulid.Now(), nil).String(),
		UserId:    user.Id,
		Type:      domain.UserTokenTypeRefresh,
		Token:     refreshTokenStr,
		ExpiresAt: &expiresAt,
	}
	storedTokens, err := s.uow.UserToken().Store(ctx, refreshToken)
	if err != nil {
		return domain.User{}, domain.UserToken{}, "", domain.NewError(domain.CodeSystemInternal, domain.WithError(err))
	}

	return user, storedTokens[0], accessToken, nil
}

// GoogleRefresh validates an existing refresh token and rotates it with a new token pair.
// The old refresh token is revoked and a new one is issued atomically within a transaction.
// Returns the authenticated user, the new stored refresh token entity, the new JWT access token string, and any error.
func (s *Service) GoogleRefresh(ctx context.Context, refreshToken string) (domain.User, domain.UserToken, string, error) {
	// 1. Find refresh token in DB
	token, err := s.uow.UserToken().FindByToken(ctx, refreshToken)
	if err != nil {
		return domain.User{}, domain.UserToken{}, "", domain.NewError(domain.CodeAuthGoogleRefreshInvalidToken)
	}

	// 2. Check if revoked
	if token.RevokedAt != nil {
		return domain.User{}, domain.UserToken{}, "", domain.NewError(domain.CodeAuthGoogleRefreshTokenRevoked)
	}

	// 3. Check if expired
	if token.ExpiresAt != nil && time.Now().After(*token.ExpiresAt) {
		return domain.User{}, domain.UserToken{}, "", domain.NewError(domain.CodeAuthGoogleRefreshTokenExpired)
	}

	// 4. Find user
	user, err := s.uow.User().Find(ctx, token.UserId)
	if err != nil {
		return domain.User{}, domain.UserToken{}, "", domain.NewError(domain.CodeAuthGoogleRefreshUserNotFound)
	}

	// 5. Execute atomic token rotation (revoke old + store new)
	var newRefreshToken domain.UserToken
	var newAccessToken string

	err = s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		// 5a. Revoke old refresh token (prevents replay attacks)
		_, err = uow.UserToken().Revoke(ctx, token.Id)
		if err != nil {
			return domain.NewError(domain.CodeSystemInternal, domain.WithError(err))
		}

		// 5b. Generate new access token
		newAccessToken, err = security.GenerateJWT(user.Id, []byte(s.config.JWTSecret), time.Duration(s.config.JWTAccessExpiryMinutes)*time.Minute)
		if err != nil {
			return domain.NewError(domain.CodeAuthGoogleRefreshJWTFailed, domain.WithError(err))
		}

		// 5c. Generate new refresh token
		newRefreshTokenStr := crypto.MustGenerateRandomToken()
		expiresAt := time.Now().Add(time.Duration(s.config.JWTRefreshExpiryHours) * time.Hour)

		// 5d. Store new refresh token
		newRefreshToken = domain.UserToken{
			Id:        ulid.MustNew(ulid.Now(), nil).String(),
			UserId:    user.Id,
			Type:      domain.UserTokenTypeRefresh,
			Token:     newRefreshTokenStr,
			ExpiresAt: &expiresAt,
		}
		storedTokens, err := uow.UserToken().Store(ctx, newRefreshToken)
		if err != nil {
			return domain.NewError(domain.CodeSystemInternal, domain.WithError(err))
		}
		newRefreshToken = storedTokens[0]

		return nil
	})

	if err != nil {
		return domain.User{}, domain.UserToken{}, "", err
	}

	return *user, newRefreshToken, newAccessToken, nil
}
