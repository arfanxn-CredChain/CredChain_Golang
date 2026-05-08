package credential

import (
	"context"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	"gorm.io/gorm"
)

type gormCredentialRepository struct {
	db *gorm.DB
}

func NewGormCredentialRepository(db *gorm.DB) domain.CredentialRepository {
	return &gormCredentialRepository{db: db}
}

func (r *gormCredentialRepository) Get(ctx context.Context, query *domainQuery.Query) ([]domain.Credential, int, error) {
	return nil, 0, nil // TODO: Implement later
}

func (r *gormCredentialRepository) Find(ctx context.Context, id string) (*domain.Credential, error) {
	return nil, nil // TODO: Implement later
}

func (r *gormCredentialRepository) FindByHolder(ctx context.Context, holderID string) ([]domain.Credential, error) {
	return nil, nil // TODO: Implement later
}
