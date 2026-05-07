package user

import (
	"context"
	"time"

	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/gorm/model"

	"gorm.io/gorm"
)

// GormUserTokenRepository implements domain.UserTokenRepository using GORM
type GormUserTokenRepository struct {
	db *gorm.DB
}

// NewGormUserTokenRepository creates a new GORM-based user token repository
func NewGormUserTokenRepository(db *gorm.DB) domain.UserTokenRepository {
	return &GormUserTokenRepository{db: db}
}

// Store creates new user tokens
func (r *GormUserTokenRepository) Store(ctx context.Context, tokens ...domain.UserToken) ([]domain.UserToken, error) {
	gormTokens := make([]model.UserToken, len(tokens))
	for i, token := range tokens {
		gormTokens[i] = model.FromDomainUserToken(token)
	}

	if err := r.db.WithContext(ctx).Create(&gormTokens).Error; err != nil {
		return nil, err
	}

	stored := make([]domain.UserToken, len(gormTokens))
	for i, gormToken := range gormTokens {
		stored[i] = gormToken.ToDomain()
	}
	return stored, nil
}

// FindByToken retrieves a token by its token string
func (r *GormUserTokenRepository) FindByToken(ctx context.Context, token string) (*domain.UserToken, error) {
	var gormToken model.UserToken
	if err := r.db.WithContext(ctx).Where("token = ?", token).First(&gormToken).Error; err != nil {
		return nil, err
	}
	domainToken := gormToken.ToDomain()
	return &domainToken, nil
}

// FindByUserId retrieves all tokens for a user
func (r *GormUserTokenRepository) FindByUserId(ctx context.Context, userId string) ([]domain.UserToken, error) {
	var gormTokens []model.UserToken
	if err := r.db.WithContext(ctx).Where("user_id = ?", userId).Find(&gormTokens).Error; err != nil {
		return nil, err
	}

	tokens := make([]domain.UserToken, len(gormTokens))
	for i, gormToken := range gormTokens {
		tokens[i] = gormToken.ToDomain()
	}
	return tokens, nil
}

// Revoke marks tokens as revoked, returns number of revoked tokens
func (r *GormUserTokenRepository) Revoke(ctx context.Context, tokenIDs ...string) (int64, error) {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&model.UserToken{}).
		Where("id IN ?", tokenIDs).
		Update("revoked_at", now)
	return result.RowsAffected, result.Error
}

// RevokeByUserIdAndType revokes all tokens of a specific type for a user
func (r *GormUserTokenRepository) RevokeByUserIdAndType(ctx context.Context, userId string, tokenType domain.UserTokenType) (int64, error) {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&model.UserToken{}).
		Where("user_id = ? AND type = ?", userId, tokenType).
		Update("revoked_at", now)
	return result.RowsAffected, result.Error
}
