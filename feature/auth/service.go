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
	userRepo      domain.UserRepository
	userTokenRepo domain.UserTokenRepository
	uow           domain.UnitOfWork
	oauthClient   *oauth.GoogleOAuthClient
	config        *config.Config
}

type AuthServiceParams struct {
	fx.In
	UserRepo      domain.UserRepository
	UserTokenRepo domain.UserTokenRepository
	UoW           domain.UnitOfWork
	OAuthClient   *oauth.GoogleOAuthClient
	Config        *config.Config
}

// NewAuthService creates a new auth service
func NewAuthService(p AuthServiceParams) *Service {
	return &Service{
		userRepo:      p.UserRepo,
		userTokenRepo: p.UserTokenRepo,
		uow:           p.UoW,
		oauthClient:   p.OAuthClient,
		config:        p.Config,
	}
}

// GoogleLogin validates a Google ID token and issues a new access/refresh token pair.
// Returns the authenticated user, the stored refresh token entity, the JWT access token string, and any error.
func (s *Service) GoogleLogin(ctx context.Context, idToken string) (domain.User, domain.UserToken, string, error) {
	// 1. Validate Google ID token (with timeout to prevent hanging)
	validateCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	payload, err := s.oauthClient.Validate(validateCtx, idToken, "")
	if err != nil {
		return domain.User{}, domain.UserToken{}, "", domain.NewError(domain.CodeAuthGoogleLoginInvalidToken, domain.WithError(err))
	}

	// 2. Extract email
	email := oauth.ExtractEmailFromGoogleIdToken(payload)
	if email == "" {
		return domain.User{}, domain.UserToken{}, "", domain.NewError(domain.CodeAuthGoogleLoginInvalidToken)
	}

	// 3. Find user by email (invite-only system)
	users, err := s.userRepo.FindByEmails(ctx, email)
	if err != nil || len(users) == 0 {
		return domain.User{}, domain.UserToken{}, "", domain.NewError(domain.CodeAuthGoogleLoginUserNotFound)
	}
	user := users[0]

	// 3b. Revoke all existing refresh tokens (prevents token accumulation)
	_, err = s.userTokenRepo.RevokeByUserIdAndType(ctx, user.Id, domain.UserTokenTypeRefresh)
	if err != nil {
		return domain.User{}, domain.UserToken{}, "", domain.NewError(domain.CodeSystemInternal, domain.WithError(err))
	}

	// 4. Generate access token
	accessToken, err := security.GenerateJWT(user.Id, []byte(s.config.JWTSecret), time.Duration(s.config.JWTAccessExpiryMinutes)*time.Minute)
	if err != nil {
		return domain.User{}, domain.UserToken{}, "", domain.NewError(domain.CodeAuthGoogleLoginJWTFailed, domain.WithError(err))
	}

	// 5. Generate refresh token
	refreshTokenStr := crypto.MustGenerateRandomHex32()
	expiresAt := time.Now().Add(time.Duration(s.config.JWTRefreshExpiryHours) * time.Hour)

	// 6. Store refresh token in DB
	refreshToken := domain.UserToken{
		Id:        ulid.MustNew(ulid.Now(), nil).String(),
		UserId:    user.Id,
		Type:      domain.UserTokenTypeRefresh,
		Token:     refreshTokenStr,
		ExpiresAt: &expiresAt,
	}
	storedTokens, err := s.userTokenRepo.Store(ctx, refreshToken)
	if err != nil {
		return domain.User{}, domain.UserToken{}, "", domain.NewError(domain.CodeSystemInternal, domain.WithError(err))
	}

	return user, storedTokens[0], accessToken, nil
}

// Refresh validates an existing refresh token and rotates it with a new token pair.
// The old refresh token is revoked and a new one is issued atomically within a transaction.
// Returns the authenticated user, the new stored refresh token entity, the new JWT access token string, and any error.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (domain.User, domain.UserToken, string, error) {
	// 1. Find refresh token in DB
	token, err := s.userTokenRepo.Find(ctx, refreshToken)
	if err != nil {
		return domain.User{}, domain.UserToken{}, "", domain.NewError(domain.CodeAuthRefreshInvalidToken)
	}

	// 2. Check if revoked
	if token.RevokedAt != nil {
		return domain.User{}, domain.UserToken{}, "", domain.NewError(domain.CodeAuthRefreshTokenRevoked)
	}

	// 3. Check if expired
	if token.ExpiresAt != nil && time.Now().After(*token.ExpiresAt) {
		return domain.User{}, domain.UserToken{}, "", domain.NewError(domain.CodeAuthRefreshTokenExpired)
	}

	// 4. Find user
	user, err := s.userRepo.Find(ctx, token.UserId)
	if err != nil {
		return domain.User{}, domain.UserToken{}, "", domain.NewError(domain.CodeAuthRefreshUserNotFound)
	}

	// 5. Execute atomic token rotation (revoke old + store new)
	var newRefreshToken domain.UserToken
	var newAccessToken string

	err = s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		// 5a. Mark token as used and revoke it in a single update
		now := time.Now()
		token.LastUsedAt = &now
		token.RevokedAt = &now
		_, err = uow.UserToken().Update(ctx, *token)
		if err != nil {
			return domain.NewError(domain.CodeSystemInternal, domain.WithError(err))
		}

		// 5b. Generate new access token
		newAccessToken, err = security.GenerateJWT(user.Id, []byte(s.config.JWTSecret), time.Duration(s.config.JWTAccessExpiryMinutes)*time.Minute)
		if err != nil {
			return domain.NewError(domain.CodeAuthRefreshJWTFailed, domain.WithError(err))
		}

		// 5d. Generate new refresh token
		newRefreshTokenStr := crypto.MustGenerateRandomHex32()
		expiresAt := time.Now().Add(time.Duration(s.config.JWTRefreshExpiryHours) * time.Hour)

		// 5e. Store new refresh token
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

// Logout revokes all refresh tokens for the given user.
func (s *Service) Logout(ctx context.Context, userId string) error {
	_, err := s.userTokenRepo.RevokeByUserIdAndType(ctx, userId, domain.UserTokenTypeRefresh)
	return err
}
