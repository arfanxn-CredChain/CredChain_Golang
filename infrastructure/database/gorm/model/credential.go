package model

import (
	"time"

	"CredChain_Golang/domain"
)

// Credential is the GORM model for the credentials table.
//
// Meta uses serializer:json so SQLite tests round-trip JSONB columns through
// TEXT (matching model.User.Meta). Extraction data (text, ids, embedding)
// lives in MongoDB (credential_extractions), not on this row.
// ExtractStatus is the credential_extract_status Postgres ENUM stored as
// TEXT in SQLite.
//
// HolderUser / IssuerUser / RevokerUser are GORM relationship fields.
// They are populated by Preload in the repository when the query's Includes
// contains the corresponding key.
type Credential struct {
	Id            string               `gorm:"primaryKey;type:char(26);column:id"`
	HolderUserId  string               `gorm:"type:char(26);column:holder_user_id;index"`
	IssuerUserId  string               `gorm:"type:char(26);column:issuer_user_id;index"`
	RevokerUserId *string              `gorm:"type:char(26);column:revoker_user_id"`
	Name          string               `gorm:"type:varchar(256);column:name;not null"`
	Meta          map[string]any       `gorm:"type:jsonb;serializer:json;column:meta"`
	TokenID       *string              `gorm:"type:varchar(256);column:token_id;uniqueIndex"`
	FileHash      string               `gorm:"type:char(66);column:file_hash;index"`
	FileURI       *string              `gorm:"type:text;column:file_uri"`
	ExtractStatus domain.ExtractStatus `gorm:"type:credential_extract_status;column:extract_status;not null;default:pending"`
	ExtractError  *string              `gorm:"type:text;column:extract_error"`
	ExtractedAt   *time.Time           `gorm:"column:extracted_at"`
	IssuedAt      time.Time            `gorm:"column:issued_at;not null;default:CURRENT_TIMESTAMP"`
	RevokedAt     *time.Time           `gorm:"column:revoked_at"`

	// GORM relations — populated by db.Preload("HolderUser") etc. from the
	// repository layer when the caller requests includes of "holder",
	// "issuer", or "revoker". A single batch IN-clause query runs per
	// Preload regardless of result size (no N+1).
	HolderUser  User `gorm:"foreignKey:Id;references:HolderUserId"`
	IssuerUser  User `gorm:"foreignKey:Id;references:IssuerUserId"`
	RevokerUser User `gorm:"foreignKey:Id;references:RevokerUserId"`
}

func (Credential) TableName() string { return "credentials" }

// ToDomain converts a GORM credential to its domain entity. Preloaded
// user relations (HolderUser, IssuerUser, RevokerUser) are mapped to the
// corresponding domain.User pointers.
func (m Credential) ToDomain() domain.Credential {
	c := domain.Credential{
		ID:            m.Id,
		HolderUserID:  m.HolderUserId,
		IssuerUserID:  m.IssuerUserId,
		RevokerUserID: m.RevokerUserId,
		Name:          m.Name,
		Meta:          m.Meta,
		TokenID:       m.TokenID,
		FileHash:      m.FileHash,
		FileURI:       m.FileURI,
		ExtractStatus: m.ExtractStatus,
		ExtractError:  m.ExtractError,
		ExtractedAt:   m.ExtractedAt,
		IssuedAt:      m.IssuedAt,
		RevokedAt:     m.RevokedAt,
	}
	if m.HolderUserId != "" {
		u := m.HolderUser.ToDomain()
		c.Holder = &u
	}
	if m.IssuerUserId != "" {
		u := m.IssuerUser.ToDomain()
		c.Issuer = &u
	}
	if m.RevokerUserId != nil && *m.RevokerUserId != "" {
		u := m.RevokerUser.ToDomain()
		c.Revoker = &u
	}
	return c
}

// FromDomainCredential converts a domain.Credential into a GORM model.
// A zero-value ExtractStatus defaults to ExtractStatusPending.
func FromDomainCredential(c domain.Credential) Credential {
	status := c.ExtractStatus
	if status == "" {
		status = domain.ExtractStatusPending
	}
	return Credential{
		Id:            c.ID,
		HolderUserId:  c.HolderUserID,
		IssuerUserId:  c.IssuerUserID,
		RevokerUserId: c.RevokerUserID,
		Name:          c.Name,
		Meta:          c.Meta,
		TokenID:       c.TokenID,
		FileHash:      c.FileHash,
		FileURI:       c.FileURI,
		ExtractStatus: status,
		ExtractError:  c.ExtractError,
		ExtractedAt:   c.ExtractedAt,
		IssuedAt:      c.IssuedAt,
		RevokedAt:     c.RevokedAt,
	}
}
