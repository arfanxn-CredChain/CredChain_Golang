package domain

import (
	"context"
	"time"

	domainQuery "CredChain_Golang/domain/query"
)

// Credential represents a row in the credentials table
type Credential struct {
	ID            string         `db:"id" json:"id"`
	HolderUserID  string         `db:"holder_user_id" json:"holder_user_id"`
	IssuerUserID  string         `db:"issuer_user_id" json:"issuer_user_id"`
	RevokerUserID *string        `db:"revoker_user_id" json:"revoker_user_id"`
	Name          string         `db:"name" json:"name"`
	Meta          map[string]any `db:"meta" json:"meta"`
	TokenID       *string        `db:"token_id" json:"token_id"`
	FileHash      string         `db:"file_hash" json:"file_hash"`
	IssuedAt      time.Time      `db:"issued_at" json:"issued_at"`
	RevokedAt     *time.Time     `db:"revoked_at" json:"revoked_at"`
}

// CredentialRepository defines the database contract for the Credential Domain
type CredentialRepository interface {
	Get(ctx context.Context, query *domainQuery.Query) ([]Credential, int, error)
	Find(ctx context.Context, id string) (*Credential, error)
	FindByHolder(ctx context.Context, holderID string) ([]Credential, error)
}
