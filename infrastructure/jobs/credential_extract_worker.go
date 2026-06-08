package jobs

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	pyai "CredChain_Golang/infrastructure/ai/pyai"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ── Worker ────────────────────────────────────────────────────────────────

// CredentialExtractWorker polls the credential_extract_jobs table and calls
// the Python AI service /extract endpoint for each pending job. It uses
// FOR UPDATE SKIP LOCKED so multiple workers never double-process.
//
// The extract results (embeddings) are written back to the credentials table
// via domain.CredentialRepository.Update.
type CredentialExtractWorker struct {
	jobRepo     domain.CredentialExtractJobRepository
	credRepo    domain.CredentialRepository
	aiClient    pyai.PythonAIClient
	logger      *zap.Logger
	cfg         *config.Config
	workerCount int
	pollSeconds int
	maxAttempts int
}

// CredentialExtractWorkerParams is the FX-provided constructor input.
type CredentialExtractWorkerParams struct {
	fx.In
	JobRepo  domain.CredentialExtractJobRepository
	CredRepo domain.CredentialRepository
	AIClient pyai.PythonAIClient
	Logger   *zap.Logger
	Config   *config.Config
}

// NewCredentialExtractWorker creates a new worker. Worker count and poll
// interval are read from config.
func NewCredentialExtractWorker(p CredentialExtractWorkerParams) *CredentialExtractWorker {
	workerCount := 1
	if p.Config.CredentialExtractWorkerCount != nil {
		workerCount = *p.Config.CredentialExtractWorkerCount
	}
	pollSeconds := 2
	if p.Config.CredentialExtractWorkerPollSeconds != nil {
		pollSeconds = *p.Config.CredentialExtractWorkerPollSeconds
	}
	maxAttempts := 3
	if p.Config.CredentialExtractWorkerMaxAttempts != nil {
		maxAttempts = *p.Config.CredentialExtractWorkerMaxAttempts
	}
	return &CredentialExtractWorker{
		jobRepo:     p.JobRepo,
		credRepo:    p.CredRepo,
		aiClient:    p.AIClient,
		logger:      p.Logger,
		cfg:         p.Config,
		workerCount: workerCount,
		pollSeconds: pollSeconds,
		maxAttempts: maxAttempts,
	}
}

// Start launches worker goroutines. Called from the FX OnStart lifecycle hook.
func (w *CredentialExtractWorker) Start(ctx context.Context) {
	for i := 0; i < w.workerCount; i++ {
		go w.runLoop(ctx, i)
	}
}

func (w *CredentialExtractWorker) runLoop(ctx context.Context, id int) {
	ticker := time.NewTicker(time.Duration(w.pollSeconds) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.processOne(ctx); err != nil {
				w.logger.Error("credential extract worker error",
					zap.Int("worker", id),
					zap.Error(err))
			}
		}
	}
}

// processOne claims a pending job, processes the file through Python /extract,
// and records the outcome on both the job row and the parent credential row.
func (w *CredentialExtractWorker) processOne(ctx context.Context) error {
	query := &domainQuery.Query{}
	job, err := w.jobRepo.FindPending(ctx, query)
	if err != nil {
		return err
	}
	if job == nil {
		return nil
	}
	if err := w.jobRepo.MarkRunning(ctx, job.ID); err != nil {
		return err
	}
	w.execute(ctx, job)
	return nil
}

// execute reads the persisted file, calls Python /extract, and updates
// the credential and job rows with the result.
func (w *CredentialExtractWorker) execute(ctx context.Context, job *domain.CredentialExtractJob) {
	data, err := os.ReadFile(job.FileURI)
	if err != nil {
		w.markFailed(ctx, job.ID, "read file: "+err.Error())
		return
	}

	filename := filepath.Base(job.FileURI)
	results, err := w.aiClient.Extract(ctx, pyai.ExtractFile{
		Filename: filename,
		Data:     data,
	})
	if err != nil {
		w.markFailed(ctx, job.ID, err.Error())
		return
	}
	if len(results) == 0 || len(results[0].Embedding) == 0 {
		w.markFailed(ctx, job.ID, "python extract returned empty embeddings")
		return
	}

	now := time.Now()
	// TODO(Task 3.3): this worker is replaced by the River worker; embeddings now persist to MongoDB.
	cred := domain.Credential{
		ID:            job.CredentialID,
		ExtractStatus: domain.ExtractStatusSucceeded,
		ExtractedAt:   &now,
	}
	if _, err := w.credRepo.Update(ctx, cred); err != nil {
		w.markFailed(ctx, job.ID, "update credential: "+err.Error())
		return
	}
	if err := w.jobRepo.MarkSucceeded(ctx, job.ID); err != nil {
		w.logger.Error("mark succeeded", zap.String("job_id", job.ID), zap.Error(err))
	}
}

// markFailed records a failed attempt. If maxAttempts is reached, also stamps
// the credential with extract_status=failed and the error message.
func (w *CredentialExtractWorker) markFailed(ctx context.Context, jobID string, errMsg string) {
	w.logger.Warn("credential extract job failed",
		zap.String("job_id", jobID),
		zap.String("error", errMsg))

	if err := w.jobRepo.MarkFailed(ctx, jobID, errMsg, w.maxAttempts); err != nil {
		w.logger.Error("mark failed", zap.String("job_id", jobID), zap.Error(err))
	}
}

// ── Enqueue helper ────────────────────────────────────────────────────────

// EnqueueExtractJob creates a new credential_extract_jobs row. Caller
// typically passes the domain.CredentialExtractJobRepository scoped to the
// current UoW transaction so the enqueue is atomic with the credential INSERT.
func EnqueueExtractJob(ctx context.Context, jobRepo domain.CredentialExtractJobRepository, credentialID, fileURI string) error {
	job := &domain.CredentialExtractJob{
		CredentialID: credentialID,
		FileURI:      fileURI,
		Status:       "pending",
		AvailableAt:  time.Now(),
	}
	return jobRepo.Store(ctx, job)
}
