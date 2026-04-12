package credential

import (
	"context"

	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/database"

	"go.uber.org/fx"
)

type PostgresCredRepository struct {
	db *database.DB
}

type CredRepoParams struct {
	fx.In
	DB *database.DB
}

func NewRepository(p CredRepoParams) domain.CredentialRepository {
	return &PostgresCredRepository{db: p.DB}
}

func (r *PostgresCredRepository) GetCredentials(ctx context.Context) ([]domain.Credential, error) {
	return nil, nil // Phase stub
}

func (r *PostgresCredRepository) GetCredentialByID(ctx context.Context, id string) (*domain.Credential, error) {
	return nil, nil // Phase stub
}

func (r *PostgresCredRepository) GetCredentialsByHolder(ctx context.Context, holderID string) ([]domain.Credential, error) {
	var creds []domain.Credential
	query := `SELECT * FROM credentials WHERE holder_user_id = $1 ORDER BY issued_at DESC`
	err := r.db.SelectContext(ctx, &creds, query, holderID)
	return creds, err
}
