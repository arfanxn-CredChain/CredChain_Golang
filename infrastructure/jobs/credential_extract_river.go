package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/ai/pyai"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// CredentialExtractArgs is the River job payload for async credential extraction.
type CredentialExtractArgs struct {
	CredentialID string `json:"credential_id"`
	FileURI      string `json:"file_uri"`
}

func (CredentialExtractArgs) Kind() string { return "credential_extract" }

// Enqueuer abstracts River job insertion for the feature layer.
type Enqueuer interface {
	EnqueueExtract(ctx context.Context, args CredentialExtractArgs) error
}

// CredentialExtractWorker performs credential extraction jobs via River.
type CredentialExtractWorker struct {
	river.WorkerDefaults[CredentialExtractArgs]
	credRepo       domain.CredentialRepository
	extractionRepo domain.CredentialExtractionRepository
	aiClient       pyai.PythonAIClient
	logger         *zap.Logger
}

type CredentialExtractWorkerParams struct {
	fx.In
	CredRepo       domain.CredentialRepository
	ExtractionRepo domain.CredentialExtractionRepository
	AIClient       pyai.PythonAIClient
	Logger         *zap.Logger
}

func NewCredentialExtractWorker(p CredentialExtractWorkerParams) *CredentialExtractWorker {
	return &CredentialExtractWorker{
		credRepo:       p.CredRepo,
		extractionRepo: p.ExtractionRepo,
		aiClient:       p.AIClient,
		logger:         p.Logger,
	}
}

// HandleError implements river.ErrorHandler. On terminal failure (max retries
// exhausted), stamps the credential extract_status = failed so reextract can act.
func (w *CredentialExtractWorker) HandleError(ctx context.Context, job *rivertype.JobRow, err error) *river.ErrorHandlerResult {
	w.handleTerminalFailure(ctx, job, err.Error())
	return &river.ErrorHandlerResult{}
}

// HandlePanic implements river.ErrorHandler. Treats a terminal panic the same
// as a terminal error: stamps the credential failed once retries are exhausted.
func (w *CredentialExtractWorker) HandlePanic(ctx context.Context, job *rivertype.JobRow, panicVal any, trace string) *river.ErrorHandlerResult {
	w.handleTerminalFailure(ctx, job, fmt.Sprintf("panic: %v", panicVal))
	return &river.ErrorHandlerResult{}
}

// handleTerminalFailure stamps the credential failed only when River has
// exhausted all retry attempts. The job row is untyped, so decode the args.
func (w *CredentialExtractWorker) handleTerminalFailure(ctx context.Context, job *rivertype.JobRow, errMsg string) {
	if job.Kind != (CredentialExtractArgs{}).Kind() || job.Attempt < job.MaxAttempts {
		return
	}
	var args CredentialExtractArgs
	if decodeErr := json.Unmarshal(job.EncodedArgs, &args); decodeErr != nil {
		w.logger.Error("failed to decode extract job args on terminal failure", zap.Error(decodeErr))
		return
	}
	w.logger.Error("extraction job permanently failed",
		zap.String("credential_id", args.CredentialID),
		zap.String("error", errMsg))
	if updateErr := w.workMarkFailed(ctx, args.CredentialID, errMsg); updateErr != nil {
		w.logger.Error("failed to mark credential extract_status=failed", zap.Error(updateErr))
	}
}

func (w *CredentialExtractWorker) Work(ctx context.Context, job *river.Job[CredentialExtractArgs]) error {
	return w.workExtract(ctx, job.Args)
}

// workExtract reads the file, calls Python, writes Mongo, updates Postgres lifecycle.
// Helper prefixed "work" (the method name).
func (w *CredentialExtractWorker) workExtract(ctx context.Context, args CredentialExtractArgs) error {
	data, err := os.ReadFile(args.FileURI)
	if err != nil {
		return fmt.Errorf("read file %q: %w", args.FileURI, err)
	}

	results, err := w.aiClient.Extract(ctx, pyai.ExtractFile{
		Filename: filepath.Base(args.FileURI),
		Data:     data,
	})
	if err != nil {
		return fmt.Errorf("python extract: %w", err)
	}
	if len(results) == 0 || len(results[0].Embedding) == 0 {
		return fmt.Errorf("python extract returned empty embedding for %s", args.CredentialID)
	}
	res := results[0]

	// Read credential's file_hash for the Mongo doc
	cred, err := w.credRepo.Find(ctx, args.CredentialID, nil)
	if err != nil {
		return fmt.Errorf("load credential %s: %w", args.CredentialID, err)
	}

	// Mongo write first (idempotent upsert)
	if err := w.extractionRepo.Store(ctx, domain.CredentialExtraction{
		CredentialID: args.CredentialID,
		FileHash:     cred.FileHash,
		Text:         res.Text,
		IDs:          workToDomainIDs(res.IDs),
		Embedding:    res.Embedding,
	}); err != nil {
		return fmt.Errorf("store extraction: %w", err)
	}

	// Postgres lifecycle update
	now := time.Now()
	if _, err := w.credRepo.Update(ctx, domain.Credential{
		ID:            args.CredentialID,
		ExtractStatus: domain.ExtractStatusSucceeded,
		ExtractedAt:   &now,
	}); err != nil {
		return fmt.Errorf("update credential: %w", err)
	}

	w.logger.Info("extraction job succeeded", zap.String("credential_id", args.CredentialID))
	return nil
}

// workMarkFailed stamps the credential as failed with the given error message.
// Used by HandleError when River permanently discards the job.
func (w *CredentialExtractWorker) workMarkFailed(ctx context.Context, credentialID string, errMsg string) error {
	_, updateErr := w.credRepo.Update(ctx, domain.Credential{
		ID:            credentialID,
		ExtractStatus: domain.ExtractStatusFailed,
		ExtractError:  &errMsg,
	})
	return updateErr
}

func workToDomainIDs(ids []pyai.ExtractedID) []domain.ExtractedID {
	out := make([]domain.ExtractedID, len(ids))
	for i, v := range ids {
		out[i] = domain.ExtractedID{Type: v.Type, Value: v.Value}
	}
	return out
}
