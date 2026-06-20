package domain

import (
	"context"
	"time"
)

// CredentialExtractedID is one identifier extracted from a credential document.
//
// Distinct from pyai.ExtractedID (the HTTP wire type returned by the Python
// AI client): this domain copy carries BSON tags for MongoDB persistence.
// The service/worker layer maps between the two — keep them separate so the
// domain package stays free of infrastructure dependencies.
type CredentialExtractedID struct {
	Type  string `bson:"type"  json:"type"`
	Value string `bson:"value" json:"value"`
}

// CredentialExtraction is the MongoDB document holding the heavy extraction
// payload for a credential (text, ids, embedding). Lives in Mongo so the
// Postgres credentials table stays lean; searchable by ids.value.
type CredentialExtraction struct {
	CredentialID string        `bson:"credential_id" json:"credential_id"`
	FileHash     string        `bson:"file_hash"     json:"file_hash"`
	Text         string        `bson:"text"          json:"text"`
	IDs          []CredentialExtractedID `bson:"ids"           json:"ids"`
	Embedding    []float64     `bson:"embedding"     json:"embedding"`
	CreatedAt    time.Time     `bson:"created_at"    json:"created_at"`
	UpdatedAt    time.Time     `bson:"updated_at"    json:"updated_at"`
}

// CredentialExtractionRepository is the MongoDB contract for extraction docs.
type CredentialExtractionRepository interface {
	// Store upserts the extraction by credential_id.
	Store(ctx context.Context, extraction CredentialExtraction) error
	// FindByCredentialId returns the extraction for a credential, or nil.
	FindByCredentialId(ctx context.Context, credentialID string) (*CredentialExtraction, error)
	// FindRankedByIds returns extractions whose ids.value intersect the given
	// values, ranked by intersection count desc, capped at limit. Single
	// aggregation pipeline — NO per-id queries.
	FindRankedByIds(ctx context.Context, values []string, limit int) ([]CredentialExtraction, error)
}
