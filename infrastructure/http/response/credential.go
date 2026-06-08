package response

import (
	"time"

	"CredChain_Golang/domain"
)

// Credential is the response DTO for credential data. Embeddings are excluded
// to keep payloads small (a 768-float vector per credential adds substantial
// bytes).
//
// Holder, Issuer, and Revoker are optional user expansions loaded from the
// preloaded domain.Credential entity via FromDomainCredential.
type Credential struct {
	ID            string               `json:"id"`
	HolderUserID  string               `json:"holder_user_id"`
	IssuerUserID  string               `json:"issuer_user_id"`
	RevokerUserID *string              `json:"revoker_user_id"`
	Name          string               `json:"name"`
	Meta          map[string]any       `json:"meta"`
	TokenID       *string              `json:"token_id"`
	FileHash      string               `json:"file_hash"`
	FileURI       *string              `json:"file_uri"`
	ExtractStatus domain.ExtractStatus `json:"extract_status"`
	ExtractError  *string              `json:"extract_error"`
	ExtractedAt   *time.Time           `json:"extracted_at"`
	IssuedAt      time.Time            `json:"issued_at"`
	RevokedAt     *time.Time           `json:"revoked_at"`
	Holder        *User                `json:"holder,omitempty"`
	Issuer        *User                `json:"issuer,omitempty"`
	Revoker       *User                `json:"revoker,omitempty"`
}

// FromDomainCredential converts a domain Credential entity to a response DTO.
// Preloaded holder/issuer/revoker users are read directly from the domain
// entity (populated by the repository's GORM Preload).
func FromDomainCredential(c domain.Credential) Credential {
	out := Credential{
		ID:            c.ID,
		HolderUserID:  c.HolderUserID,
		IssuerUserID:  c.IssuerUserID,
		RevokerUserID: c.RevokerUserID,
		Name:          c.Name,
		Meta:          c.Meta,
		TokenID:       c.TokenID,
		FileHash:      c.FileHash,
		FileURI:       c.FileURI,
		ExtractStatus: c.ExtractStatus,
		ExtractError:  c.ExtractError,
		ExtractedAt:   c.ExtractedAt,
		IssuedAt:      c.IssuedAt,
		RevokedAt:     c.RevokedAt,
	}
	if c.Holder != nil {
		h := FromDomainUser(*c.Holder)
		out.Holder = &h
	}
	if c.Issuer != nil {
		i := FromDomainUser(*c.Issuer)
		out.Issuer = &i
	}
	if c.Revoker != nil {
		r := FromDomainUser(*c.Revoker)
		out.Revoker = &r
	}
	return out
}

// CredentialVerify is the response payload for POST /api/credentials/verify.
// VerdictCode is the 6-digit domain code for the verify outcome; Description
// is its localized message, resolved by the handler via the request's i18n
// localizer (Accept-Language). Credential is the matched credential, if any.
type CredentialVerify struct {
	VerdictCode       int         `json:"verdict_code"`
	SimilarityScore   *float64    `json:"similarity_score,omitempty"`
	SimilarityPercent *string     `json:"similarity_percent,omitempty"`
	Description       string      `json:"description"`
	Credential        *Credential `json:"credential,omitempty"`
}
