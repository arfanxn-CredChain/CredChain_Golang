package domain

import (
	"context"
	"time"

	domainQuery "CredChain_Golang/domain/query"

	"gorm.io/gorm"
)

// CredentialExtractJob is the domain entity for the credential_extract_jobs
// Postgres-backed work queue. Workers claim pending jobs with FOR UPDATE SKIP
// LOCKED, call the Python AI service /extract, and update both the job status
// and the credential's embeddings.
type CredentialExtractJob struct {
	ID           string
	CredentialID string
	FileURI      string
	Status       string
	AttemptCount int
	Errors       []string
	AvailableAt  time.Time
	ReservedAt   *time.Time
	CreatedAt    time.Time
}

// CredentialExtractJobRepository defines the database contract for the
// credential_extract_jobs work queue.
type CredentialExtractJobRepository interface {
	// Get is the standard paginated list endpoint. Primarily used by
	// administrative dashboards to inspect the queue.
	Get(ctx context.Context, query *domainQuery.Query) ([]CredentialExtractJob, int, error)

	// FindPending claims a single pending job with FOR UPDATE SKIP LOCKED
	// ordered by created_at ASC (FIFO). Returns nil if no eligible job exists.
	FindPending(ctx context.Context, query *domainQuery.Query) (*CredentialExtractJob, error)

	// FindPendingTx is the transactional variant — the caller supplies the
	// *gorm.DB (from a UoW transaction) so the claim and subsequent status
	// updates share the same connection.
	FindPendingTx(ctx context.Context, tx *gorm.DB, query *domainQuery.Query) (*CredentialExtractJob, error)

	// Store inserts a new job. Used inside the credential issue UoW
	// transaction so the job is committed atomically with the credential row.
	Store(ctx context.Context, job *CredentialExtractJob) error

	// MarkRunning transitions a job from pending to running and stamps
	// reserved_at.
	MarkRunning(ctx context.Context, id string) error

	// MarkSucceeded marks a job as successfully completed.
	MarkSucceeded(ctx context.Context, id string) error

	// MarkFailed marks a job as failed, appending the error message to the
	// errors array and incrementing attempt_count. If attempt_count reaches
	// the configured max, status switches to terminal "failed"; otherwise
	// it resets to "pending" with a backoff on available_at.
	MarkFailed(ctx context.Context, id string, errMsg string, maxAttempts int) error
}
