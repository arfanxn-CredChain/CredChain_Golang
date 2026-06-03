package credential

import (
	"context"
	"time"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	"CredChain_Golang/infrastructure/database/gorm/model"

	"github.com/oklog/ulid/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ── Repository struct & factory ──────────────────────────────────────────

type gormCredentialExtractJobRepository struct {
	db *gorm.DB
}

// NewGormCredentialExtractJobRepository is the exported factory for FX injection.
func NewGormCredentialExtractJobRepository(db *gorm.DB) domain.CredentialExtractJobRepository {
	return &gormCredentialExtractJobRepository{db: db}
}

// ── Pagination ────────────────────────────────────────────────────────────

// Get returns a paginated list of extract jobs for administrative dashboards.
func (r *gormCredentialExtractJobRepository) Get(ctx context.Context, query *domainQuery.Query) ([]domain.CredentialExtractJob, int, error) {
	db := r.db.WithContext(ctx).Model(&model.CredentialExtractJob{})
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if query.HasPagination() {
		db = db.Limit(query.Limit).Offset(query.Offset())
	}
	db = db.Order("created_at ASC")
	var rows []model.CredentialExtractJob
	if err := db.Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.CredentialExtractJob, len(rows))
	for i, r := range rows {
		out[i] = r.ToDomain()
	}
	return out, int(total), nil
}

// ── Claim ─────────────────────────────────────────────────────────────────

// FindPending claims a single pending job with FOR UPDATE SKIP LOCKED,
// ordered by created_at ASC (FIFO). Returns nil if no eligible job exists.
func (r *gormCredentialExtractJobRepository) FindPending(ctx context.Context, query *domainQuery.Query) (*domain.CredentialExtractJob, error) {
	return r.FindPendingTx(ctx, r.db, query)
}

// FindPendingTx is the transactional variant — the caller supplies the
// *gorm.DB (e.g. from a UoW transaction).
func (r *gormCredentialExtractJobRepository) FindPendingTx(ctx context.Context, tx *gorm.DB, query *domainQuery.Query) (*domain.CredentialExtractJob, error) {
	var row model.CredentialExtractJob
	err := tx.WithContext(ctx).Model(&model.CredentialExtractJob{}).
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("status = ? AND available_at <= ?", "pending", time.Now()).
		Order("created_at ASC").
		First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	d := row.ToDomain()
	return &d, nil
}

// ── Mutation ──────────────────────────────────────────────────────────────

// Store inserts a new job row. The caller is responsible for generating the
// ULID (typically done by the feature layer).
func (r *gormCredentialExtractJobRepository) Store(ctx context.Context, job *domain.CredentialExtractJob) error {
	if job.ID == "" {
		job.ID = ulid.Make().String()
	}
	m := model.FromDomainCredentialExtractJob(*job)
	return r.db.WithContext(ctx).Create(&m).Error
}

// MarkRunning transitions a job to running and stamps reserved_at.
func (r *gormCredentialExtractJobRepository) MarkRunning(ctx context.Context, id string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.CredentialExtractJob{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":      "running",
			"reserved_at": now,
		}).Error
}

// MarkSucceeded marks a job as completed.
func (r *gormCredentialExtractJobRepository) MarkSucceeded(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&model.CredentialExtractJob{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status": "succeeded",
		}).Error
}

// MarkFailed records an error attempt. If attempt_count has reached
// maxAttempts, the job transitions to terminal "failed"; otherwise it resets
// to "pending" with exponential backoff on available_at.
func (r *gormCredentialExtractJobRepository) MarkFailed(ctx context.Context, id string, errMsg string, maxAttempts int) error {
	var job model.CredentialExtractJob
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&job).Error; err != nil {
		return err
	}
	newCount := job.AttemptCount + 1
	if newCount >= maxAttempts {
		return r.db.WithContext(ctx).Model(&model.CredentialExtractJob{}).
			Where("id = ?", id).
			Updates(map[string]any{
				"status":        "failed",
				"attempt_count": newCount,
				"errors":        gorm.Expr("array_append(errors, ?)", errMsg),
				"reserved_at":   nil,
			}).Error
	}
	backoff := time.Duration(newCount*newCount) * time.Second
	return r.db.WithContext(ctx).Model(&model.CredentialExtractJob{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":        "pending",
			"attempt_count": newCount,
			"errors":        gorm.Expr("array_append(errors, ?)", errMsg),
			"available_at":  time.Now().Add(backoff),
			"reserved_at":   nil,
		}).Error
}

var _ domain.CredentialExtractJobRepository = (*gormCredentialExtractJobRepository)(nil)
