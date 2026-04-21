package credential

import (
	"context"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	"CredChain_Golang/infrastructure/gorm"
)

type GormCredRepository struct {
	db *gorm.GormDB
}

func NewGormCredentialRepository(db *gorm.GormDB) domain.CredentialRepository {
	return &GormCredRepository{db: db}
}

func (r *GormCredRepository) Get(ctx context.Context, query *domainQuery.Query) ([]domain.Credential, int, error) {
	return nil, 0, nil // TODO: Implement later
}

func (r *GormCredRepository) Find(ctx context.Context, id string) (*domain.Credential, error) {
	return nil, nil // TODO: Implement later
}

func (r *GormCredRepository) FindByHolder(ctx context.Context, holderID string) ([]domain.Credential, error) {
	return nil, nil // TODO: Implement later
}
