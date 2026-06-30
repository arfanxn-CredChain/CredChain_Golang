package domain

import (
	"context"
	"time"
)

// CredentialVerification is the MongoDB cache document for a verify result,
// keyed by the uploaded file's keccak256 hash. TTL-bounded via created_at
// MongoDB TTL index. VerdictCode is the 6-digit domain response code
// (e.g. 400401 verified_authentic) stored as a frozen snapshot of the last
// verify computation for this uploaded file.
type CredentialVerification struct {
	UploadedFileHash    string    `bson:"uploaded_file_hash"    json:"uploaded_file_hash"`
	VerdictCode         int       `bson:"verdict_code"          json:"verdict_code"`
	MatchedCredentialID *string   `bson:"matched_credential_id" json:"matched_credential_id"`
	SimilarityScore     *float64  `bson:"similarity_score"      json:"similarity_score"`
	SimilarityPercent   *string   `bson:"similarity_percent"    json:"similarity_percent"`
	CreatedAt           time.Time `bson:"created_at"            json:"created_at"`
}

// CredentialVerificationRepository is the MongoDB contract for the verify cache.
type CredentialVerificationRepository interface {
	// FindByUploadedFileHash returns the cached verification, or nil.
	FindByUploadedFileHash(ctx context.Context, hash string) (*CredentialVerification, error)
	// Store upserts the cache entry by uploaded_file_hash.
	Store(ctx context.Context, verification CredentialVerification) error
	// DeleteByUploadedFileHashes deletes cache entries whose uploaded_file_hash
	// is in the given list. No-op on empty input. Fails fast if any error.
	DeleteByUploadedFileHashes(ctx context.Context, hashes []string) error
}
