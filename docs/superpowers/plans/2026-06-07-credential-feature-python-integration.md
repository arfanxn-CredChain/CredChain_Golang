# Credential Feature — Go ↔ Python AI Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Align the Go credential feature with the current Python AI service contract, re-home extraction data to MongoDB, and rebuild `/verify` as a cache → exact-hash → fuzzy-match pipeline. Replace the custom poll worker with riverqueue/river for async extraction jobs.

**Architecture:** DDD layering (domain interface → infrastructure impl → FX provider). Postgres keeps credential lifecycle; MongoDB stores extraction payloads + a TTL-bounded verification cache. River (Postgres-backed) runs async extraction jobs. On-chain issuance/revocation unchanged. Strict NO-N+1: all batch ops use a single query/aggregation.

**Tech Stack:** Go 1.25.1, Gin, Uber FX, GORM (Postgres), mongo-driver v2 (MongoDB), riverqueue/river + riverpgxv5 (job queue), go-ethereum, testify.

**Spec:** `docs/superpowers/specs/2026-06-07-credential-feature-python-integration-design.md`

**Phases (each independently testable):**
1. Wire-contract repair (`pyai` client + config)
2. MongoDB infrastructure (client, repos, migration command)
3. `/batch/issue` partial success + River job enqueue + Postgres schema
4. `/verify` redesign (cache → exact → fuzzy, locale description)
5. `/batch/reextract` endpoint
6. Tests (service + River worker + pyai client)
7. AGENTS.md + docs + Postman collection

**Cross-cutting conventions (apply everywhere):**
- Config defaults live in `config.go` only. Consumers dereference `*cfg.Field` — NO hardcoded fallback literals in services/workers/commands.
- A main method's private helpers carry that method's name as a prefix (e.g. `Verify`→`verify*`, `Extract`→`extract*`, `Issue`→`issue*`, `ReExtract`→`reExtract*`).
- `file_uri` stores a relative storage path only (e.g. `uploads/01J….pdf`), never a full URL.
- Strict NO-N+1 in every batch path.

**Verification command (run after each phase):**
```
go test ./... && go vet ./... && gofmt -l .
```
(`gofmt -l .` must produce zero output.)

---

## Phase 1: Wire-Contract Repair

### Task 1.1: Add `PythonAIAPIKey` config field

**Files:**
- Modify: `config/config.go`
- Modify: `.env.example`, `.env`, `.env.docker`

- [ ] **Step 1: Add the struct field**

In `config/config.go`, in the `Config` struct near `PythonAIBaseURL`, add:

```go
	PythonAIAPIKey                     *string
```

- [ ] **Step 2: Wire env loading**

In `NewConfig` (where `PythonAIBaseURL` is assigned), add:

```go
		PythonAIAPIKey:                     getEnv("PYTHON_AI_API_KEY", nil),
```

- [ ] **Step 3: Add env var to all env files**

Append to `.env.example`, `.env`, and `.env.docker`:

```
PYTHON_AI_API_KEY=
```

- [ ] **Step 4: Verify build**

Run: `go build ./...`
Expected: builds clean.

- [ ] **Step 5: Commit**

```bash
git add config/config.go .env.example .env .env.docker
git commit -m "feat(config): add PYTHON_AI_API_KEY"
```

### Task 1.2: Rewrite `pyai` client to match current Python contract

**Files:**
- Modify: `infrastructure/ai/pyai/client.go`

- [ ] **Step 1: Replace type definitions**

Replace the existing result/file types at the top of `client.go` (after imports). Delete `VerifyDescription` entirely — it is no longer surfaced (the verify description now comes from Go locales, see Phase 4):

```go
// ExtractedID is one identifier extracted from a credential document by Python.
type ExtractedID struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// ExtractFile is a single file to send to Python /extract, /verify, or /extract-ids.
// If MIMEType is empty, the client derives it from Filename.
type ExtractFile struct {
	Filename string
	MIMEType string
	Data     []byte
}

// PythonExtractResult is the per-file result from Python /extract.
// nil Embedding means the file failed extraction on the Python side.
type PythonExtractResult struct {
	Text      string
	IDs       []ExtractedID
	Embedding []float64
}

// VerifyResult is the result from Python /verify for a single file.
type VerifyResult struct {
	Verdict           string
	SimilarityScore   float64
	SimilarityPercent string
}
```

- [ ] **Step 2: Replace internal response structs**

```go
type extractData struct {
	Text      string        `json:"text"`
	IDs       []ExtractedID `json:"ids"`
	Embedding []float64     `json:"embedding"`
}

type verifyData struct {
	SimilarityScore   float64 `json:"similarity_score"`
	SimilarityPercent string  `json:"similarity_percent"`
	Verdict           string  `json:"verdict"`
}
```

(Note: Python returns `descriptions` too, but Go ignores it — the description is generated from Go locales.)

- [ ] **Step 3: Update struct + constructor to carry the API key**

```go
type pythonAIClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	mu         sync.Mutex
}

func NewPythonAIClient(cfg *config.Config) PythonAIClient {
	return &pythonAIClient{
		baseURL:    *cfg.PythonAIBaseURL,
		apiKey:     derefOrEmpty(cfg.PythonAIAPIKey),
		httpClient: &http.Client{Timeout: time.Duration(*cfg.PythonAITimeoutSeconds) * time.Second},
	}
}

func derefOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
```

Defaults for `PythonAIBaseURL` and `PythonAITimeoutSeconds` already live in `config.go` — do NOT re-default here.

- [ ] **Step 4: Add auth + MIME helpers (extract-prefixed where method-specific)**

```go
func (c *pythonAIClient) setAuthHeader(req *http.Request) {
	if c.apiKey != "" {
		req.Header.Set("X-Api-Key", c.apiKey)
	}
}

// resolveMIME derives the MIME type from the filename when not provided,
// using the Go standard library (mime.TypeByExtension) — no custom MIME table.
func (f ExtractFile) resolveMIME() string {
	if f.MIMEType != "" {
		return f.MIMEType
	}
	if t := mime.TypeByExtension(strings.ToLower(filepath.Ext(f.Filename))); t != "" {
		return t
	}
	return "application/octet-stream"
}
```

Add `"mime"`, `"path/filepath"`, and `"strings"` to the `pyai` imports. **Delete the bespoke `FileExtToMIME` function from `pyai`** — it is a general file utility that does not belong in the AI-client package, and `mime.TypeByExtension` (stdlib) replaces it. Grep for any remaining `pyai.FileExtToMIME` callers (e.g. the old worker) and remove them as part of the worker deletion in Task 3.3.

Callers (worker, verify) no longer build MIME externally — they pass `Filename` + `Data` and the client resolves MIME via `resolveMIME()` inside `buildMultipartFiles` / `Verify`.

- [ ] **Step 5: Rewrite `Extract` (new field names, auth header, all-failed code 500150)**

```go
func (c *pythonAIClient) Extract(ctx context.Context, files ...ExtractFile) ([]PythonExtractResult, error) {
	if len(files) == 0 {
		return nil, nil
	}
	body, contentType, err := buildMultipartFiles(files)
	if err != nil {
		return nil, fmt.Errorf("python extract: build multipart: %w", err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/extract", body)
	if err != nil {
		return nil, fmt.Errorf("python extract: build request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	c.setAuthHeader(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("python extract: http: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("python extract: read body: %w", err)
	}
	var parsed pythonResponse[*extractData]
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("python extract: decode: %w", err)
	}
	if parsed.Code == 500150 {
		return nil, fmt.Errorf("python extract: all files failed (code %d)", parsed.Code)
	}
	results := make([]PythonExtractResult, len(parsed.Data))
	for i, d := range parsed.Data {
		if d != nil {
			results[i] = PythonExtractResult{Text: d.Text, IDs: d.IDs, Embedding: d.Embedding}
		}
	}
	return results, nil
}
```

- [ ] **Step 6: Rewrite `Verify` (form field `embeddings` = `[[...]]`, new response)**

```go
func (c *pythonAIClient) Verify(ctx context.Context, file ExtractFile, storedEmbedding []float64) (*VerifyResult, error) {
	embJSON, err := json.Marshal([][]float64{storedEmbedding})
	if err != nil {
		return nil, fmt.Errorf("python verify: marshal embeddings: %w", err)
	}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("files", file.Filename)
	if err != nil {
		return nil, fmt.Errorf("python verify: create form file: %w", err)
	}
	if _, err := part.Write(file.Data); err != nil {
		return nil, fmt.Errorf("python verify: write file: %w", err)
	}
	if err := writer.WriteField("embeddings", string(embJSON)); err != nil {
		return nil, fmt.Errorf("python verify: write embeddings: %w", err)
	}
	writer.Close()
	c.mu.Lock()
	defer c.mu.Unlock()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/verify", body)
	if err != nil {
		return nil, fmt.Errorf("python verify: build request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	c.setAuthHeader(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("python verify: http: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("python verify: read body: %w", err)
	}
	var parsed pythonResponse[*verifyData]
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("python verify: decode: %w", err)
	}
	if len(parsed.Data) == 0 || parsed.Data[0] == nil {
		return nil, fmt.Errorf("python verify: empty result (code %d)", parsed.Code)
	}
	d := parsed.Data[0]
	return &VerifyResult{Verdict: d.Verdict, SimilarityScore: d.SimilarityScore, SimilarityPercent: d.SimilarityPercent}, nil
}
```

- [ ] **Step 7: Add `ExtractIDs` (interface + impl)**

Add to the `PythonAIClient` interface:

```go
	ExtractIDs(ctx context.Context, file ExtractFile) ([]ExtractedID, error)
```

Implement:

```go
type extractIdsData struct {
	IDs []ExtractedID `json:"ids"`
}

func (c *pythonAIClient) ExtractIDs(ctx context.Context, file ExtractFile) ([]ExtractedID, error) {
	body, contentType, err := buildMultipartFiles([]ExtractFile{file})
	if err != nil {
		return nil, fmt.Errorf("python extract-ids: build multipart: %w", err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/extract-ids", body)
	if err != nil {
		return nil, fmt.Errorf("python extract-ids: build request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	c.setAuthHeader(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("python extract-ids: http: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("python extract-ids: read body: %w", err)
	}
	var parsed pythonResponse[*extractIdsData]
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("python extract-ids: decode: %w", err)
	}
	if len(parsed.Data) == 0 || parsed.Data[0] == nil {
		return []ExtractedID{}, nil
	}
	return parsed.Data[0].IDs, nil
}
```

- [ ] **Step 8: Update `buildMultipartFiles` to resolve MIME**

Ensure `buildMultipartFiles` writes each part with the resolved MIME (so the client owns MIME detection, not callers). If the helper currently ignores MIME, set the part header `Content-Type` to `f.resolveMIME()`.

- [ ] **Step 9: Update stale comments**

In `client.go` interface doc + `domain/credential.go` line ~11, replace "LaBSE" with "EmbeddingGemma".

- [ ] **Step 10: Verify build and tests**

```bash
go build ./...
go test ./...
go vet ./...
gofmt -l .
```
Expected: all pass; `gofmt -l .` zero output.

- [ ] **Step 11: Commit**

```bash
git add infrastructure/ai/pyai/client.go domain/credential.go
git commit -m "feat(pyai): align client with current Python contract + extract-ids"
```

---

## Phase 2: MongoDB Infrastructure

### Task 2.1: Add Mongo + River config fields

**Files:**
- Modify: `config/config.go`
- Modify: `.env.example`, `.env`, `.env.docker`

- [ ] **Step 1: Add struct fields**

In `config/config.go` `Config` struct, near `MongoURI`, add:

```go
	MongoDatabase               *string
	AIVerificationCacheTTLHours *int
	RiverMaxWorkers             *int
```

- [ ] **Step 2: Wire env loading with defaults (defaults live HERE only)**

In `NewConfig`, add defaults:

```go
	defaultMongoDatabase := "credchain"
	defaultAIVerificationCacheTTLHours := 24
	defaultRiverMaxWorkers := 10
```

In the `Config{...}` literal:

```go
		MongoDatabase:               getEnv("MONGO_DATABASE", &defaultMongoDatabase),
		AIVerificationCacheTTLHours: getIntEnv("AI_VERIFICATION_CACHE_TTL_HOURS", &defaultAIVerificationCacheTTLHours),
		RiverMaxWorkers:             getIntEnv("RIVER_MAX_WORKERS", &defaultRiverMaxWorkers),
```

- [ ] **Step 3: Add env vars**

Append to `.env.example`, `.env`, `.env.docker`:

```
MONGO_DATABASE=credchain
AI_VERIFICATION_CACHE_TTL_HOURS=24
RIVER_MAX_WORKERS=10
```

- [ ] **Step 4: Verify + commit**

```bash
go build ./...
git add config/config.go .env.example .env .env.docker
git commit -m "feat(config): add mongo database, verification cache TTL, river workers"
```

### Task 2.2: Mongo client FX provider

**Files:**
- Create: `infrastructure/database/mongo/client.go`

- [ ] **Step 1: Write the client provider (no hardcoded db name — read from cfg)**

```go
package mongo

import (
	"context"
	"fmt"

	"CredChain_Golang/config"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/fx"
)

// NewClient connects to MongoDB and registers a lifecycle hook to disconnect.
func NewClient(lc fx.Lifecycle, cfg *config.Config) (*mongo.Client, error) {
	if cfg.MongoURI == nil || *cfg.MongoURI == "" {
		return nil, fmt.Errorf("MONGO_URI is required")
	}
	client, err := mongo.Connect(options.Client().ApplyURI(*cfg.MongoURI))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongo: %w", err)
	}
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error { return client.Disconnect(ctx) },
	})
	return client, nil
}

// NewDatabase returns the configured Mongo database handle. The default name
// lives in config.go (MongoDatabase); this provider only dereferences it.
func NewDatabase(client *mongo.Client, cfg *config.Config) *mongo.Database {
	return client.Database(*cfg.MongoDatabase)
}
```

- [ ] **Step 2: Verify + commit**

```bash
go build ./...
git add infrastructure/database/mongo/client.go
git commit -m "feat(mongo): add client + database FX providers"
```

### Task 2.3: Domain entities + repository interfaces

**Files:**
- Create: `domain/credential_extraction.go`
- Create: `domain/credential_verification.go`

- [ ] **Step 1: Write `credential_extraction.go`**

```go
package domain

import (
	"context"
	"time"
)

// ExtractedID is one identifier extracted from a credential document.
type ExtractedID struct {
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
	IDs          []ExtractedID `bson:"ids"           json:"ids"`
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
```

- [ ] **Step 2: Write `credential_verification.go`**

```go
package domain

import (
	"context"
	"time"
)

// CredentialVerification is the MongoDB cache document for a verify result,
// keyed by the uploaded file's keccak256 hash. TTL-bounded via created_at.
// VerdictCode is the 6-digit domain response code (e.g. 400401 verified_authentic)
// returned to the caller; the full doc is a frozen snapshot of the last verify
// computation for this uploaded file.
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
}
```

- [ ] **Step 3: Verify + commit**

```bash
go build ./...
git add domain/credential_extraction.go domain/credential_verification.go
git commit -m "feat(domain): add credential extraction + verification entities"
```

### Task 2.4: Mongo extraction repository implementation

**Files:**
- Create: `feature/credential/mongo_credential_extraction_repository.go`

- [ ] **Step 1: Write the repository**

```go
package credential

import (
	"context"
	"time"

	"CredChain_Golang/domain"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const credentialExtractionsCollection = "credential_extractions"

type mongoCredentialExtractionRepository struct {
	coll *mongo.Collection
}

// NewMongoCredentialExtractionRepository is the exported FX factory.
func NewMongoCredentialExtractionRepository(db *mongo.Database) domain.CredentialExtractionRepository {
	return &mongoCredentialExtractionRepository{coll: db.Collection(credentialExtractionsCollection)}
}

func (r *mongoCredentialExtractionRepository) Store(ctx context.Context, e domain.CredentialExtraction) error {
	now := time.Now()
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now
	}
	e.UpdatedAt = now
	_, err := r.coll.UpdateOne(ctx,
		bson.M{"credential_id": e.CredentialID},
		bson.M{
			"$set": bson.M{
				"file_hash":  e.FileHash,
				"text":       e.Text,
				"ids":        e.IDs,
				"embedding":  e.Embedding,
				"updated_at": e.UpdatedAt,
			},
			"$setOnInsert": bson.M{"credential_id": e.CredentialID, "created_at": e.CreatedAt},
		},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

func (r *mongoCredentialExtractionRepository) FindByCredentialId(ctx context.Context, credentialID string) (*domain.CredentialExtraction, error) {
	var out domain.CredentialExtraction
	err := r.coll.FindOne(ctx, bson.M{"credential_id": credentialID}).Decode(&out)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// FindRankedByIds runs a single aggregation: match docs sharing any id value,
// compute the intersection size, sort by it desc, limit. NO per-id queries.
func (r *mongoCredentialExtractionRepository) FindRankedByIds(ctx context.Context, values []string, limit int) ([]domain.CredentialExtraction, error) {
	if len(values) == 0 {
		return []domain.CredentialExtraction{}, nil
	}
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"ids.value": bson.M{"$in": values}}}},
		{{Key: "$addFields", Value: bson.M{
			"match_count": bson.M{"$size": bson.M{"$setIntersection": bson.A{"$ids.value", values}}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "match_count", Value: -1}}}},
		{{Key: "$limit", Value: int64(limit)}},
	}
	cur, err := r.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []domain.CredentialExtraction
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}
```

- [ ] **Step 2: Verify + commit**

```bash
go build ./...
git add feature/credential/mongo_credential_extraction_repository.go
git commit -m "feat(credential): add mongo extraction repository"
```

### Task 2.5: Mongo verification (cache) repository implementation

**Files:**
- Create: `feature/credential/mongo_credential_verification_repository.go`

- [ ] **Step 1: Write the repository**

```go
package credential

import (
	"context"
	"time"

	"CredChain_Golang/domain"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const credentialVerificationsCollection = "credential_verifications"

type mongoCredentialVerificationRepository struct {
	coll *mongo.Collection
}

// NewMongoCredentialVerificationRepository is the exported FX factory.
func NewMongoCredentialVerificationRepository(db *mongo.Database) domain.CredentialVerificationRepository {
	return &mongoCredentialVerificationRepository{coll: db.Collection(credentialVerificationsCollection)}
}

func (r *mongoCredentialVerificationRepository) FindByUploadedFileHash(ctx context.Context, hash string) (*domain.CredentialVerification, error) {
	var out domain.CredentialVerification
	err := r.coll.FindOne(ctx, bson.M{"uploaded_file_hash": hash}).Decode(&out)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *mongoCredentialVerificationRepository) Store(ctx context.Context, v domain.CredentialVerification) error {
	if v.CreatedAt.IsZero() {
		v.CreatedAt = time.Now()
	}
	_, err := r.coll.UpdateOne(ctx,
		bson.M{"uploaded_file_hash": v.UploadedFileHash},
		bson.M{"$set": bson.M{
			"verdict_code":          v.VerdictCode,
			"matched_credential_id": v.MatchedCredentialID,
			"similarity_score":      v.SimilarityScore,
			"similarity_percent":    v.SimilarityPercent,
			"created_at":            v.CreatedAt,
		}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}
```

- [ ] **Step 2: Verify + commit**

```bash
go build ./...
git add feature/credential/mongo_credential_verification_repository.go
git commit -m "feat(credential): add mongo verification cache repository"
```

### Task 2.6: Mongo migration CLI command

**Files:**
- Create: `cmd/migrate_mongo.go`
- Modify: `Makefile`

- [ ] **Step 1: Write the migration command**

Register a top-level `migrate-mongo` command with `up` / `down` subcommands. Helpers carry the `migrateMongo` prefix. Defaults (db name, TTL hours) are read from cfg — NOT hardcoded.

```go
package cmd

import (
	"context"
	"fmt"

	"CredChain_Golang/config"
	infraLogger "CredChain_Golang/infrastructure/logger"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func init() {
	migrateMongoCmd.AddCommand(migrateMongoUpCmd)
	migrateMongoCmd.AddCommand(migrateMongoDownCmd)
	rootCmd.AddCommand(migrateMongoCmd)
}

var migrateMongoCmd = &cobra.Command{
	Use:   "migrate-mongo",
	Short: "MongoDB collection + index migration tools",
}

var migrateMongoUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Creates Mongo collections and indexes (idempotent)",
	Run: func(cmd *cobra.Command, args []string) {
		fx.New(infraLogger.Module, fx.Provide(NewConfigFromCmd(cmd)), fx.Invoke(migrateMongoUp)).Run()
	},
}

func migrateMongoConnect(cfg *config.Config) (*mongo.Database, func(context.Context) error, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(*cfg.MongoURI))
	if err != nil {
		return nil, nil, fmt.Errorf("connect mongo: %w", err)
	}
	return client.Database(*cfg.MongoDatabase), client.Disconnect, nil
}

func migrateMongoUp(cfg *config.Config, logger *zap.Logger) error {
	ctx := context.Background()
	db, disconnect, err := migrateMongoConnect(cfg)
	if err != nil {
		return err
	}
	defer disconnect(ctx)

	ttlSeconds := int32(*cfg.AIVerificationCacheTTLHours * 3600)

	if _, err := db.Collection("credential_extractions").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "credential_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "file_hash", Value: 1}}},
		{Keys: bson.D{{Key: "ids.value", Value: 1}}},
	}); err != nil {
		return fmt.Errorf("create extraction indexes: %w", err)
	}
	if _, err := db.Collection("credential_verifications").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "uploaded_file_hash", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "created_at", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(ttlSeconds)},
	}); err != nil {
		return fmt.Errorf("create verification indexes: %w", err)
	}
	logger.Info("successfully ran mongo migrations up")
	return nil
}

var migrateMongoDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Drops Mongo collections (destructive)",
	Run: func(cmd *cobra.Command, args []string) {
		fx.New(infraLogger.Module, fx.Provide(NewConfigFromCmd(cmd)), fx.Invoke(migrateMongoDown)).Run()
	},
}

func migrateMongoDown(cfg *config.Config, logger *zap.Logger) error {
	ctx := context.Background()
	db, disconnect, err := migrateMongoConnect(cfg)
	if err != nil {
		return err
	}
	defer disconnect(ctx)
	for _, name := range []string{"credential_extractions", "credential_verifications"} {
		if err := db.Collection(name).Drop(ctx); err != nil {
			return fmt.Errorf("drop %s: %w", name, err)
		}
	}
	logger.Info("successfully ran mongo migrations down")
	return nil
}
```

- [ ] **Step 2: Add Makefile targets (named `migrate-up-mongo` / `migrate-down-mongo`)**

In `Makefile`, after the existing `migrate-down` target, add (match the file's existing `ENV ?= .env` convention):

```makefile
migrate-up-mongo:
	go run main.go migrate-mongo up --env $(ENV)

migrate-down-mongo:
	go run main.go migrate-mongo down --env $(ENV)
```

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: builds clean.

- [ ] **Step 4: Commit**

```bash
git add cmd/migrate_mongo.go Makefile
git commit -m "feat(cmd): add migrate-mongo up/down (make migrate-up-mongo / migrate-down-mongo)"
```

### Task 2.7: Wire Mongo providers into FX

**Files:**
- Modify: `cmd/server.go`

- [ ] **Step 1: Add import**

```go
	infraMongo "CredChain_Golang/infrastructure/database/mongo"
```

- [ ] **Step 2: Add providers** (in `fx.Provide(...)`, after `gormInfra.NewGorm`)

```go
				infraMongo.NewClient,
				infraMongo.NewDatabase,
				credential.NewMongoCredentialExtractionRepository,
				credential.NewMongoCredentialVerificationRepository,
```

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: builds clean (providers unused yet — FX allows it).

- [ ] **Step 4: Commit**

```bash
git add cmd/server.go
git commit -m "feat(fx): wire mongo client + credential repositories"
```

---

## Phase 3: `/batch/issue` Partial Success + River Job Enqueue

### Task 3.1: Drop `embeddings` column from Postgres schema

**Files:**
- Modify: `infrastructure/database/migrations/000001_initial_schema.up.sql`
- Modify: `infrastructure/database/migrations/000001_initial_schema.down.sql`
- Modify: `infrastructure/database/gorm/model/credential.go`
- Modify: `domain/credential.go`

- [ ] **Step 1: Remove the column from up.sql**

In `000001_initial_schema.up.sql`, delete the line:

```sql
    embeddings JSONB,
```

- [ ] **Step 2: Keep down.sql consistent**

Confirm `000001_initial_schema.down.sql` only drops tables/types. If it recreates the `credentials` table, remove any `embeddings` reference there too.

- [ ] **Step 3: Remove `Embeddings` from the GORM model**

In `infrastructure/database/gorm/model/credential.go`, delete the struct field:

```go
	Embeddings    []float64            `gorm:"type:jsonb;serializer:json;column:embeddings"`
```

Remove `Embeddings: m.Embeddings,` from `ToDomain()` and `Embeddings: c.Embeddings,` from `FromDomainCredential()`.

- [ ] **Step 4: Remove `Embeddings` from the domain entity**

In `domain/credential.go`, delete the field:

```go
	Embeddings    []float64      `db:"embeddings"      json:"embeddings,omitempty"`
```

- [ ] **Step 5: Verify build (expect breakages to fix later)**

Run: `go build ./...`
Expected: compile errors only in `credential_service.go` (Verify reads `cred.Embeddings`) and the old worker (writes `Embeddings`). These are fixed in Task 3.3 (worker → River) and Phase 4 (verify). Confirm ONLY those references break.

- [ ] **Step 6: Defer commit** — bundle with Task 3.4 (build stays red until then).

### Task 3.2: Issue service — partial success

**Files:**
- Modify: `feature/credential/credential_service.go`

**Goal:** Per-item pre-chain validation (hash, storage, holder, dup-hash) drops failed items individually; survivors go to one chain tx. Return positional results + per-index errors. `EnqueueExtractJob` lives inline in the service (not in the worker file).

- [ ] **Step 1: Change the `Issue` interface signature**

```go
Issue(ctx context.Context, items []CredentialIssuance) ([]domain.Credential, map[string][]string, error)
```

- [ ] **Step 2: Rewrite `Issue` implementation**

```go
func (s *credentialService) Issue(ctx context.Context, items []CredentialIssuance) ([]domain.Credential, map[string][]string, error) {
	if len(items) == 0 {
		return []domain.Credential{}, nil, nil
	}
	authUser := httpContext.MustGetUser(ctx)

	if err := s.policy.IssuePreFetch(ctx, lo.Map(items, func(it CredentialIssuance, _ int) domain.Credential {
		return domain.Credential{HolderUserID: it.HolderUserID}
	})); err != nil {
		return nil, nil, err
	}

	// Single IN-query for all holders — NO per-item lookup.
	holderIDs := lo.Map(items, func(it CredentialIssuance, _ int) string { return it.HolderUserID })
	holders, err := s.userRepo.FindByIds(ctx, holderIDs...)
	if err != nil {
		return nil, nil, err
	}
	holderByID := lo.SliceToMap(holders, func(h domain.User) (string, domain.User) { return h.Id, h })

	// Single batch dup-hash lookup.
	hashes := make([]string, len(items))
	for i, it := range items {
		hashes[i] = "0x" + hex.EncodeToString(ethCrypto.Keccak256(it.FileBytes))
	}
	existing, err := s.repo.FindByFileHashes(ctx, hashes...)
	if err != nil {
		return nil, nil, err
	}
	activeDup := map[string]bool{}
	for _, e := range existing {
		if e.RevokedAt == nil {
			activeDup[e.FileHash] = true
		}
	}

	type prepared struct {
		idx  int
		cred domain.Credential
	}
	errs := map[string][]string{}
	results := make([]domain.Credential, len(items))
	var survivors []prepared
	var fileURIs []string

	for i, it := range items {
		key := fmt.Sprintf("credentials.%d", i)
		if _, ok := holderByID[it.HolderUserID]; !ok {
			errs[key] = append(errs[key], "holder not found")
			continue
		}
		if activeDup[hashes[i]] {
			errs[key] = append(errs[key], "duplicate file hash")
			continue
		}
		ext := strings.ToLower(filepath.Ext(it.Filename))
		if ext == "" {
			ext = ".bin"
		}
		// file_uri stores the relative path only (e.g. uploads/01J....pdf)
		path, err := s.storage.SaveBytes(it.FileBytes, ext)
		if err != nil {
			errs[key] = append(errs[key], "storage failed")
			continue
		}
		if path == "" {
			s.cleanupOrphanFiles(fileURIs)
			return nil, nil, domain.NewError(domain.CodeCredentialIssueStorageFailed,
				domain.WithError(errors.New("storage returned empty path")))
		}
		fileURIs = append(fileURIs, path)
		id := ulid.Make().String()
		cred := domain.Credential{
			ID:            id,
			HolderUserID:  it.HolderUserID,
			IssuerUserID:  authUser.Id,
			Name:          it.Name,
			Meta:          it.Meta,
			FileHash:      hashes[i],
			FileURI:       &path,
			ExtractStatus: domain.ExtractStatusPending,
		}
		survivors = append(survivors, prepared{idx: i, cred: cred})
	}

	if len(survivors) == 0 {
		return results, errs, nil
	}

	survCreds := lo.Map(survivors, func(p prepared, _ int) domain.Credential { return p.cred })
	if err := s.policy.IssuePostFetch(ctx, survCreds, holders); err != nil {
		s.cleanupOrphanFiles(fileURIs)
		return nil, nil, err
	}

	signerWallet := domain.WalletFromUser(*authUser)
	var committed []domain.Credential
	err = s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		stored, err := uow.Credential().Store(ctx, survCreds...)
		if err != nil {
			return err
		}
		issuances := make([]chain.CredentialIssuance, len(stored))
		for i, c := range stored {
			issuances[i] = chain.CredentialIssuance{
				HolderAddress: holderByID[c.HolderUserID].WalletAddress,
				Hash:          c.FileHash,
				URI:           c.ID, // ULID on-chain, not storage path
			}
		}
		tokenIds, err := s.syncBlockchainIssue(ctx, signerWallet, issuances)
		if err != nil {
			return err
		}
		updates := make([]domain.Credential, len(stored))
		for i, c := range stored {
			tok := tokenIds[i].String()
			updates[i] = domain.Credential{ID: c.ID, TokenID: &tok}
			stored[i].TokenID = &tok
		}
		if _, err := uow.Credential().Update(ctx, updates...); err != nil {
			return err
		}
		for _, c := range stored {
			if c.FileURI == nil {
				// A credential without file_uri is broken — fail hard.
				return domain.NewError(domain.CodeCredentialIssueStorageFailed,
					domain.WithMetadata("credential_id", c.ID))
			}
			if err := s.issueEnqueueExtractJob(ctx, c.ID, *c.FileURI); err != nil {
				return err
			}
		}
		committed = stored
		return nil
	})
	if err != nil {
		s.cleanupOrphanFiles(fileURIs)
		return nil, nil, err
	}

	for i, p := range survivors {
		results[p.idx] = committed[i]
	}
	if len(errs) == 0 {
		errs = nil
	}
	return results, errs, nil
}

// issueEnqueueExtractJob enqueues an extraction job for a just-issued credential
// via the River enqueuer. Unexported internal helper, prefixed with the method
// name "issue". The enqueuer is wired in Task 3.5.
func (s *credentialService) issueEnqueueExtractJob(ctx context.Context, credentialID, fileURI string) error {
	return s.enqueuer.EnqueueExtract(ctx, jobs.CredentialExtractArgs{
		CredentialID: credentialID,
		FileURI:      fileURI,
	})
}
```

- [ ] **Step 3: Add the River `enqueuer` dependency to the service struct and params**

In `credentialService` struct:
```go
	enqueuer jobs.Enqueuer
```
In `CredentialServiceParams`:
```go
	Enqueuer jobs.Enqueuer
```
Set in `NewCredentialService` from params. The enqueuer is provided by the River client (Task 3.5). `jobRepo` is NOT needed on the service — River handles job persistence.

- [ ] **Step 4: Defer commit** — build still red until Task 3.3 + 3.4.

### Task 3.3: Replace custom worker with River

**Files:**
- Delete: `infrastructure/jobs/credential_extract_worker.go` (old poll worker)
- Create: `infrastructure/jobs/credential_extract_river.go`
- Modify: `go.mod` (add deps)

**Goal:** River (Postgres-backed) replaces the custom poll loop. The job carries `credential_id` + `file_uri` as args. The worker performs: read file → Python `/extract` → write Mongo extraction → update Postgres lifecycle, with the Postgres writes wrapped in a UoW transaction. Worker concurrency comes from `cfg.RiverMaxWorkers`. No hardcoded worker/poll/attempt literals.

- [ ] **Step 1: Add dependencies**

```bash
go get github.com/riverqueue/river
go get github.com/riverqueue/river/riverdriver/riverpgxv5
go get github.com/riverqueue/river/rivermigrate
go get github.com/jackc/pgx/v5/pgxpool
```

- [ ] **Step 2: Define job args + worker**

```go
package jobs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	pyai "CredChain_Golang/infrastructure/ai/pyai"

	"github.com/riverqueue/river"
	"go.uber.org/zap"
)

// CredentialExtractArgs is the River job payload for async credential extraction.
type CredentialExtractArgs struct {
	CredentialID string `json:"credential_id"`
	FileURI      string `json:"file_uri"`
}

func (CredentialExtractArgs) Kind() string { return "credential_extract" }

// CredentialExtractWorker performs credential extraction jobs.
type CredentialExtractWorker struct {
	river.WorkerDefaults[CredentialExtractArgs]
	uow            domain.UnitOfWork
	credRepo       domain.CredentialRepository
	extractionRepo domain.CredentialExtractionRepository
	aiClient       pyai.PythonAIClient
	logger         *zap.Logger
}

type CredentialExtractWorkerParams struct {
	UoW            domain.UnitOfWork
	CredRepo       domain.CredentialRepository
	ExtractionRepo domain.CredentialExtractionRepository
	AIClient       pyai.PythonAIClient
	Logger         *zap.Logger
}

func NewCredentialExtractWorker(p CredentialExtractWorkerParams) *CredentialExtractWorker {
	return &CredentialExtractWorker{
		uow:            p.UoW,
		credRepo:       p.CredRepo,
		extractionRepo: p.ExtractionRepo,
		aiClient:       p.AIClient,
		logger:         p.Logger,
	}
}

// Work is the River entrypoint. Returning an error makes River retry per the
// queue's retry policy (exponential backoff by default).
func (w *CredentialExtractWorker) Work(ctx context.Context, job *river.Job[CredentialExtractArgs]) error {
	return w.workExtract(ctx, job.Args)
}

// workExtract reads the file, calls Python, writes Mongo, and updates Postgres
// lifecycle inside a transaction. Helper prefixed with the method name "work".
func (w *CredentialExtractWorker) workExtract(ctx context.Context, args CredentialExtractArgs) error {
	data, err := os.ReadFile(args.FileURI)
	if err != nil {
		return fmt.Errorf("read file %q: %w", args.FileURI, err)
	}

	// MIME is resolved inside the pyai client from the filename.
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

	// Single read for file_hash (needed in the Mongo doc).
	cred, err := w.credRepo.Find(ctx, args.CredentialID, nil)
	if err != nil {
		return fmt.Errorf("load credential %s: %w", args.CredentialID, err)
	}

	// Mongo write is idempotent (upsert by credential_id); do it first.
	if err := w.extractionRepo.Store(ctx, domain.CredentialExtraction{
		CredentialID: args.CredentialID,
		FileHash:     cred.FileHash,
		Text:         res.Text,
		IDs:          workToDomainIDs(res.IDs),
		Embedding:    res.Embedding,
	}); err != nil {
		return fmt.Errorf("store extraction: %w", err)
	}

	// Postgres lifecycle update inside a transaction.
	now := time.Now()
	return w.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		_, err := uow.Credential().Update(ctx, domain.Credential{
			ID:            args.CredentialID,
			ExtractStatus: domain.ExtractStatusSucceeded,
			ExtractedAt:   &now,
		})
		return err
	})
}

func workToDomainIDs(ids []pyai.ExtractedID) []domain.ExtractedID {
	out := make([]domain.ExtractedID, len(ids))
	for i, v := range ids {
		out[i] = domain.ExtractedID{Type: v.Type, Value: v.Value}
	}
	return out
}

var _ = domainQuery.Query{} // keep import if Find signature needs *domainQuery.Query; else remove
var _ = config.Config{}     // remove if unused after wiring
```

Note: `credRepo.Find` currently takes `*domainQuery.Query`; pass `nil` (the repo tolerates a nil query for a plain by-id lookup — confirm and, if not, pass `&domainQuery.Query{}`). Remove the two `var _ =` lines once imports settle.

- [ ] **Step 3: On permanent failure**

River marks a job `discarded` after its max attempts. Add a separate concern: when River exhausts retries, the credential's `extract_status` should become `failed`. Implement via a River `JobInsertMiddleware` or an error handler that, on final failure, updates Postgres. Simplest approach: in `workExtract`, River retries automatically; to record terminal failure, register a River error handler in Task 3.5 wiring that updates `extract_status=failed` + `extract_error` when `river.Job.Attempt >= MaxAttempts`.

- [ ] **Step 4: Defer commit** — wiring in Task 3.5, build red until then.

### Task 3.4: Issue handler — partial-success response

**Files:**
- Modify: `feature/credential/credential_handler.go`
- Modify: `infrastructure/http/responder/responder.go`

- [ ] **Step 1: Update the `Issue` handler tail**

```go
	created, fieldErrs, err := h.credSvc.Issue(c.Request.Context(), serviceItems)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	out := make([]*response.Credential, len(created))
	successCount := 0
	for i, cred := range created {
		if cred.ID == "" {
			continue
		}
		dto := response.FromDomainCredential(cred)
		out[i] = &dto
		successCount++
	}
	code := domain.CodeCredentialIssueSuccess
	if successCount == 0 {
		code = domain.CodeCredentialIssueFailed
	}
	responder.SendPartial(c, code, out, fieldErrs)
```

- [ ] **Step 2: Add `SendPartial` responder helper**

In `infrastructure/http/responder/responder.go` add (reuse whatever message-resolution helper `Send` already uses internally — confirm its name first):

```go
// SendPartial emits a partial-success envelope: data array + per-field errors.
func SendPartial(c *gin.Context, code int, data any, fieldErrors map[string][]string) {
	status := HttpCodes[code]
	if status == 0 {
		status = http.StatusOK
	}
	c.JSON(status, gin.H{
		"code":    code,
		"message": resolveMessage(c, code),
		"data":    data,
		"errors":  fieldErrors,
	})
}
```

- [ ] **Step 3: Verify build + tests**

```bash
go build ./...
go test ./...
go vet ./...
gofmt -l .
```

- [ ] **Step 4: Defer commit** — bundle with 3.5.

### Task 3.5: River client + worker FX wiring

**Files:**
- Create: `infrastructure/jobs/river.go` (River client provider + start hook + enqueuer)
- Modify: `cmd/server.go`
- Modify: `feature/credential/credential_service.go` (enqueue via River instead of jobRepo)

**Goal:** Provide a River `*river.Client[pgx.Tx]` (workers attached), start it on FX `OnStart`, stop on `OnStop`. Expose an `Enqueuer` interface the service uses to insert jobs. Run River's own schema migration.

- [ ] **Step 1: River migration (one-time schema for River tables)**

Add to `cmd/migrate.go` `migrateUp` (after the golang-migrate run) OR document running `river migrate-up` via the CLI. Preferred: programmatic — create `infrastructure/jobs/river_migrate.go`:

```go
package jobs

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

// MigrateRiver runs River's own table migrations (idempotent).
func MigrateRiver(ctx context.Context, pool *pgxpool.Pool) error {
	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return err
	}
	_, err = migrator.Migrate(ctx, rivermigrate.DirectionUp, nil)
	return err
}
```

Call `MigrateRiver` from `migrateUp` in `cmd/migrate.go` so `make migrate-up` also provisions River tables (needs a pgx pool — open one from `*cfg.PostgresDSN`, run, close).

- [ ] **Step 2: River client provider + Enqueuer**

```go
package jobs

import (
	"context"

	"CredChain_Golang/config"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"go.uber.org/fx"
)

// Enqueuer abstracts River job insertion for the feature layer.
type Enqueuer interface {
	EnqueueExtract(ctx context.Context, args CredentialExtractArgs) error
}

type riverEnqueuer struct {
	client *river.Client[pgx.Tx]
}

func (e *riverEnqueuer) EnqueueExtract(ctx context.Context, args CredentialExtractArgs) error {
	_, err := e.client.Insert(ctx, args, nil)
	return err
}

// NewRiverClient builds a pgx pool + River client with the extraction worker
// registered, and wires lifecycle start/stop. MaxWorkers comes from config.
func NewRiverClient(lc fx.Lifecycle, cfg *config.Config, worker *CredentialExtractWorker) (*river.Client[pgx.Tx], Enqueuer, error) {
	pool, err := pgxpool.New(context.Background(), *cfg.PostgresDSN)
	if err != nil {
		return nil, nil, err
	}
	workers := river.NewWorkers()
	river.AddWorker(workers, worker)
	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues:  map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: *cfg.RiverMaxWorkers}},
		Workers: workers,
	})
	if err != nil {
		return nil, nil, err
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error { return client.Start(ctx) },
		OnStop: func(ctx context.Context) error {
			pool.Close()
			return client.Stop(ctx)
		},
	})
	return client, &riverEnqueuer{client: client}, nil
}
```

Note: import `pgx` as `github.com/jackc/pgx/v5`. Adjust the generic type param to match riverpgxv5's expected driver type (`pgx.Tx`).

- [ ] **Step 3: Confirm the service consumes the Enqueuer**

The `enqueuer jobs.Enqueuer` field and `issueEnqueueExtractJob` helper were already defined in Task 3.2. No code change here — just confirm `NewCredentialService` receives `jobs.Enqueuer` via `CredentialServiceParams` and that the River provider (Step 2) supplies it. The service never touches `jobRepo` or the `credential_extract_jobs` table directly.

- [ ] **Step 4: Wire providers in `cmd/server.go`**

Add providers: `jobs.NewCredentialExtractWorker`, `jobs.NewRiverClient`. Remove the old `infraJobs.NewCredentialExtractWorker` poll-worker provider and its `OnStart` `w.Start(ctx)` invoke block.

- [ ] **Step 5: Verify build + tests, then commit Phase 3 bundle**

```bash
go build ./... && go test ./... && go vet ./... && gofmt -l .
git add infrastructure/database/migrations/ infrastructure/database/gorm/model/credential.go domain/credential.go feature/credential/credential_service.go feature/credential/credential_handler.go infrastructure/http/responder/responder.go infrastructure/jobs/ cmd/server.go cmd/migrate.go go.mod go.sum
git commit -m "feat(credential): partial-success issue + river-based async extraction"
```

---

## Phase 4: `/verify` Redesign

### Task 4.1: Add verify verdict domain codes

**Files:**
- Modify: `domain/codes.go`
- Modify: `infrastructure/http/responder/mapper.go`
- Modify: `infrastructure/http/responder/mapper_test.go`
- Modify: `locales/en.json`, `locales/id.json`

- [ ] **Step 1: Add constants to `domain/codes.go`** (after `CodeCredentialVerifyCredentialNotFound = 400445`)

```go
	CodeCredentialVerifyAuthentic        = 400401
	CodeCredentialVerifyRevoked          = 400402
	CodeCredentialVerifyIntegrityWarning = 400403
	CodeCredentialVerifyTampered         = 400404
	CodeCredentialVerifySuspicious       = 400405
	CodeCredentialVerifyLowSimilarity    = 400406
	CodeCredentialVerifyNotSimilar       = 400407
	CodeCredentialVerifyNoIdentifiers    = 400408
	CodeCredentialVerifyNoMatch          = 400409
```

- [ ] **Step 2: `mapper.go` — `CodeToMessageKey`**

```go
	domain.CodeCredentialVerifyAuthentic:        "success_credential_verify_authentic",
	domain.CodeCredentialVerifyRevoked:          "success_credential_verify_revoked",
	domain.CodeCredentialVerifyIntegrityWarning: "warning_credential_verify_integrity",
	domain.CodeCredentialVerifyTampered:         "success_credential_verify_tampered",
	domain.CodeCredentialVerifySuspicious:       "success_credential_verify_suspicious",
	domain.CodeCredentialVerifyLowSimilarity:    "success_credential_verify_low_similarity",
	domain.CodeCredentialVerifyNotSimilar:       "success_credential_verify_not_similar",
	domain.CodeCredentialVerifyNoIdentifiers:    "success_credential_verify_no_identifiers",
	domain.CodeCredentialVerifyNoMatch:          "success_credential_verify_no_match",
```

- [ ] **Step 3: `mapper.go` — `HttpCodes`**

```go
	domain.CodeCredentialVerifyAuthentic:        http.StatusOK,
	domain.CodeCredentialVerifyRevoked:          http.StatusOK,
	domain.CodeCredentialVerifyIntegrityWarning: http.StatusConflict,
	domain.CodeCredentialVerifyTampered:         http.StatusOK,
	domain.CodeCredentialVerifySuspicious:       http.StatusOK,
	domain.CodeCredentialVerifyLowSimilarity:    http.StatusOK,
	domain.CodeCredentialVerifyNotSimilar:       http.StatusOK,
	domain.CodeCredentialVerifyNoIdentifiers:    http.StatusOK,
	domain.CodeCredentialVerifyNoMatch:          http.StatusOK,
```

- [ ] **Step 4: `mapper_test.go` — add all nine to `allDomainCodes`.**

- [ ] **Step 5: `locales/en.json`**

```json
"success_credential_verify_authentic": "The document is authentic and matches an issued credential.",
"success_credential_verify_revoked": "The document matches an issued credential that has been revoked.",
"warning_credential_verify_integrity": "The document matches a credential in the database but not on the blockchain — integrity warning.",
"success_credential_verify_tampered": "The document is suspiciously similar to an issued credential and may have been tampered.",
"success_credential_verify_suspicious": "The document is highly similar to an issued credential.",
"success_credential_verify_low_similarity": "The document has low similarity to an issued credential.",
"success_credential_verify_not_similar": "The document is not similar to the matched credential.",
"success_credential_verify_no_identifiers": "No identifiable IDs were found in the document.",
"success_credential_verify_no_match": "No matching issued credential was found for the IDs in this document."
```

- [ ] **Step 6: `locales/id.json`**

```json
"success_credential_verify_authentic": "Dokumen otentik dan cocok dengan kredensial yang telah diterbitkan.",
"success_credential_verify_revoked": "Dokumen cocok dengan kredensial yang telah dicabut.",
"warning_credential_verify_integrity": "Dokumen cocok di database tetapi tidak di blockchain — peringatan integritas.",
"success_credential_verify_tampered": "Dokumen sangat mirip dengan kredensial yang diterbitkan dan mungkin telah dimanipulasi.",
"success_credential_verify_suspicious": "Dokumen sangat mirip dengan kredensial yang diterbitkan.",
"success_credential_verify_low_similarity": "Dokumen memiliki kemiripan rendah dengan kredensial yang diterbitkan.",
"success_credential_verify_not_similar": "Dokumen tidak mirip dengan kredensial yang cocok.",
"success_credential_verify_no_identifiers": "Tidak ada ID yang dapat diidentifikasi dalam dokumen.",
"success_credential_verify_no_match": "Tidak ada kredensial yang cocok ditemukan untuk ID dalam dokumen ini."
```

- [ ] **Step 7: Test + commit**

```bash
go test ./infrastructure/http/responder/...
git add domain/codes.go infrastructure/http/responder/mapper.go infrastructure/http/responder/mapper_test.go locales/en.json locales/id.json
git commit -m "feat(codes): add verify verdict codes 400401-400409"
```

### Task 4.2: Add `FindCredentialByHash` read to RegistryService

**Files:**
- Modify: `infrastructure/chain/registry_service.go`

- [ ] **Step 1: Add interface method**

```go
	// FindCredentialByHash reads the on-chain credential whose token id is
	// derived from the file hash. Returns (hashOnChain, found, error).
	FindCredentialByHash(ctx context.Context, hash string) (string, bool, error)
```

- [ ] **Step 2: Implement**

```go
func (s *registryService) FindCredentialByHash(ctx context.Context, hash string) (string, bool, error) {
	id := tokenIdFromHash(hash)
	cred, err := s.client.Registry.FindCredential(&bind.CallOpts{Context: ctx}, id)
	if err != nil {
		return "", false, fmt.Errorf("find credential on-chain: %w", err)
	}
	if cred.Hash == "" {
		return "", false, nil
	}
	return cred.Hash, true, nil
}
```

- [ ] **Step 3: Build + commit**

```bash
go build ./...
git add infrastructure/chain/registry_service.go
git commit -m "feat(chain): add FindCredentialByHash on-chain read"
```

### Task 4.3: Verify service — cache → exact → fuzzy pipeline with locale description

**Files:**
- Modify: `feature/credential/credential_service.go`

- [ ] **Step 1: Add Mongo repo fields to struct + params (same pattern as Task 3.2 for extractionRepo, verificationRepo).**

- [ ] **Step 2: Change `Verify` interface signature**

```go
	Verify(ctx context.Context, file pyai.ExtractFile) (int, *domain.Credential, *float64, *string, error)
```

Returns `(verdictCode, matchedCredential, score, percent, error)`. Description is NOT returned by the service — the handler resolves it from the verdict code via the request's i18n localizer (see Task 4.4). This keeps the service free of HTTP/i18n concerns.

- [ ] **Step 3: Implement the pipeline**

```go
func (s *credentialService) Verify(ctx context.Context, file pyai.ExtractFile) (int, *domain.Credential, *float64, *string, error) {
	if err := s.policy.VerifyPreFetch(ctx); err != nil {
		return 0, nil, nil, nil, err
	}

	uploadedHash := "0x" + hex.EncodeToString(ethCrypto.Keccak256(file.Data))

	// 3. CACHE LOOKUP
	cached, err := s.verificationRepo.FindByUploadedFileHash(ctx, uploadedHash)
	if err != nil {
		return 0, nil, nil, nil, err
	}
	if cached != nil {
		var cred *domain.Credential
		if cached.MatchedCredentialID != nil {
			cred, _ = s.repo.Find(ctx, *cached.MatchedCredentialID, nil)
		}
		return cached.VerdictCode, cred, cached.SimilarityScore, cached.SimilarityPercent, nil
	}

	// 4. EXACT-HASH PATH
	existing, err := s.repo.FindByFileHashes(ctx, uploadedHash)
	if err != nil {
		return 0, nil, nil, nil, err
	}
	if len(existing) > 0 {
		cred := existing[0]
		_, onChain, chainErr := s.registryService.FindCredentialByHash(ctx, uploadedHash)
		code := domain.CodeCredentialVerifyAuthentic
		if chainErr != nil || !onChain {
			code = domain.CodeCredentialVerifyIntegrityWarning
		} else if cred.RevokedAt != nil {
			code = domain.CodeCredentialVerifyRevoked
		}
		s.verifyCacheVerdict(ctx, uploadedHash, code, &cred.ID, nil, nil)
		return code, &cred, nil, nil, nil
	}

	// 5. FUZZY PATH
	ids, err := s.aiClient.ExtractIDs(ctx, file)
	if err != nil {
		return 0, nil, nil, nil, domain.NewError(domain.CodeCredentialVerifyAiServiceFailed, domain.WithError(err))
	}
	if len(ids) == 0 {
		code := domain.CodeCredentialVerifyNoIdentifiers
		s.verifyCacheVerdict(ctx, uploadedHash, code, nil, nil, nil)
		return code, nil, nil, nil, nil
	}

	values := lo.Map(ids, func(id pyai.ExtractedID, _ int) string { return id.Value })
	ranked, err := s.extractionRepo.FindRankedByIds(ctx, values, 10)
	if err != nil {
		return 0, nil, nil, nil, err
	}
	if len(ranked) == 0 {
		code := domain.CodeCredentialVerifyNoMatch
		s.verifyCacheVerdict(ctx, uploadedHash, code, nil, nil, nil)
		return code, nil, nil, nil, nil
	}

	best := s.verifyPickBestMatch(ctx, ranked, values)
	result, err := s.aiClient.Verify(ctx, file, best.Embedding)
	if err != nil {
		return 0, nil, nil, nil, domain.NewError(domain.CodeCredentialVerifyAiServiceFailed, domain.WithError(err))
	}

	code := s.verifyVerdictToCode(result.Verdict)
	cred, _ := s.repo.Find(ctx, best.CredentialID, nil)
	s.verifyCacheVerdict(ctx, uploadedHash, code, &best.CredentialID, &result.SimilarityScore, &result.SimilarityPercent)
	return code, cred, &result.SimilarityScore, &result.SimilarityPercent, nil
}
```

- [ ] **Step 4: Add helpers**

```go
func (s *credentialService) verifyPickBestMatch(ctx context.Context, ranked []domain.CredentialExtraction, values []string) domain.CredentialExtraction {
	maxCount := s.verifyCountIntersection(ranked[0].IDs, values)
	var tied []domain.CredentialExtraction
	for _, r := range ranked {
		if s.verifyCountIntersection(r.IDs, values) == maxCount {
			tied = append(tied, r)
		}
	}
	if len(tied) == 1 {
		return tied[0]
	}
	ids := lo.Map(tied, func(e domain.CredentialExtraction, _ int) string { return e.CredentialID })
	creds, err := s.repo.FindByIds(ctx, ids...)
	if err != nil {
		return tied[0]
	}
	credByID := lo.SliceToMap(creds, func(c domain.Credential) (string, domain.Credential) { return c.ID, c })
	best, bestCred, bestOK := tied[0], credByID[tied[0].CredentialID], true
	for _, t := range tied[1:] {
		tc, ok := credByID[t.CredentialID]
		if !ok {
			continue
		}
		if !bestOK {
			best, bestCred, bestOK = t, tc, true; continue
		}
		bestRevoked, tRevoked := bestCred.RevokedAt != nil, tc.RevokedAt != nil
		if bestRevoked && !tRevoked {
			best, bestCred = t, tc
		} else if bestRevoked == tRevoked && tc.IssuedAt.After(bestCred.IssuedAt) {
			best, bestCred = t, tc
		}
	}
	return best
}

func (s *credentialService) verifyCountIntersection(ids []domain.ExtractedID, values []string) int {
	set := lo.SliceToMap(values, func(v string) (string, struct{}) { return v, struct{}{} })
	count := 0
	for _, id := range ids {
		if _, ok := set[id.Value]; ok {
			count++
		}
	}
	return count
}

func (s *credentialService) verifyVerdictToCode(verdict string) int {
	switch verdict {
	case "tampered":  return domain.CodeCredentialVerifyTampered
	case "suspicious": return domain.CodeCredentialVerifySuspicious
	case "low_similarity": return domain.CodeCredentialVerifyLowSimilarity
	default: return domain.CodeCredentialVerifyNotSimilar
	}
}

func (s *credentialService) verifyCacheVerdict(ctx context.Context, hash string, code int, credID *string, score *float64, percent *string) {
	if err := s.verificationRepo.Store(ctx, domain.CredentialVerification{
		UploadedFileHash: hash, VerdictCode: code, MatchedCredentialID: credID,
		SimilarityScore: score, SimilarityPercent: percent,
	}); err != nil {
		s.logger.Warn("failed to cache verification", zap.String("hash", hash), zap.Error(err))
	}
}
```

Note: the service does NOT resolve the description — it returns only the verdict code. The handler (Task 4.4) resolves the localized description from that code using the request's i18n localizer. This keeps the service free of HTTP/i18n concerns.

- [ ] **Step 5: Defer commit** — handler in Task 4.4.

### Task 4.4: Verify handler + response DTO (description resolved handler-side)

**Files:**
- Modify: `feature/credential/credential_handler.go`
- Modify: `infrastructure/http/response/credential.go`

**Decision (from Task 4.3):** the service returns `(code, cred, score, percent, error)` WITHOUT description. The handler resolves the localized description from the verdict code using the request's i18n localizer (already set by `I18nMiddleware` from `Accept-Language`). Description is generated by Go locales, never Python.

- [ ] **Step 1: Confirm the service `Verify` signature is**

```go
	Verify(ctx context.Context, file pyai.ExtractFile) (int, *domain.Credential, *float64, *string, error)
```

(Revert the `lang`/description additions from Task 4.3 Step 2 — keep the service pure. The `verifyResolveDescription` helper is REMOVED from the service.)

- [ ] **Step 2: Rewrite the `Verify` handler**

```go
func (h *credentialHandler) Verify(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		responder.SendError(c, domain.NewError(domain.CodeCredentialVerifyValidation, domain.WithError(err)))
		return
	}
	fileBytes, mime, filename, err := readUploadedFile(fileHeader)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if !allowedMIMETypes[mime] {
		responder.SendError(c, domain.NewError(domain.CodeCredentialVerifyValidation,
			domain.WithMetadata("file_mime", mime)))
		return
	}
	if int64(len(fileBytes)) > maxFileBytes {
		responder.SendError(c, domain.NewError(domain.CodeCredentialVerifyValidation,
			domain.WithMetadata("file_size", len(fileBytes))))
		return
	}
	code, cred, score, percent, err := h.credSvc.Verify(c.Request.Context(), pyai.ExtractFile{
		Filename: filename, MIMEType: mime, Data: fileBytes,
	})
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}

	// Resolve the localized description from the verdict code via the request
	// localizer (set by I18nMiddleware from Accept-Language). Go-locale driven.
	desc := responder.ResolveMessage(c, code)

	out := response.CredentialVerify{
		VerdictCode:       code,
		SimilarityScore:   score,
		SimilarityPercent: percent,
		Description:       desc,
	}
	if cred != nil {
		dto := response.FromDomainCredential(*cred)
		out.Credential = &dto
	}
	responder.Send(c, code, out)
}
```

Note: `responder.ResolveMessage(c, code)` should be an exported wrapper around whatever internal message-resolution `Send` uses. If `Send` already resolves+sends, add a thin exported `ResolveMessage(c, code) string` that returns the localized string for a code so the handler can embed it in `data` too. Confirm/extract this helper in `responder.go`.

- [ ] **Step 3: Update `CredentialVerify` response DTO**

In `infrastructure/http/response/credential.go`:

```go
type CredentialVerify struct {
	VerdictCode       int         `json:"verdict_code"`
	SimilarityScore   *float64    `json:"similarity_score,omitempty"`
	SimilarityPercent *string     `json:"similarity_percent,omitempty"`
	Description       string      `json:"description"`
	Credential        *Credential `json:"credential,omitempty"`
}
```

- [ ] **Step 4: Verify build + tests**

```bash
go build ./... && go test ./... && go vet ./... && gofmt -l .
```
Expected: green.

- [ ] **Step 5: Commit Phase 4 bundle**

```bash
git add feature/credential/credential_service.go feature/credential/credential_handler.go infrastructure/http/response/credential.go infrastructure/http/responder/responder.go infrastructure/chain/registry_service.go
git commit -m "feat(credential): redesigned /verify pipeline (cache + exact + fuzzy) with locale description"
```

---

## Phase 5: `/batch/reextract` Endpoint

### Task 5.1: Re-extract domain codes

**Files:**
- Modify: `domain/codes.go`
- Modify: `infrastructure/http/responder/mapper.go`
- Modify: `infrastructure/http/responder/mapper_test.go`
- Modify: `locales/en.json`, `locales/id.json`

- [ ] **Step 1: Add constants (new feature `05` under credential) to `domain/codes.go`**

```go
	CodeCredentialReExtractSuccess   = 400500
	CodeCredentialReExtractNotFound  = 400540
	CodeCredentialReExtractNotFailed = 400541
```

- [ ] **Step 2: `mapper.go` — `CodeToMessageKey`**

```go
	domain.CodeCredentialReExtractSuccess:   "success_credential_reextract",
	domain.CodeCredentialReExtractNotFound:  "error_credential_reextract_not_found",
	domain.CodeCredentialReExtractNotFailed: "error_credential_reextract_not_failed",
```

- [ ] **Step 3: `mapper.go` — `HttpCodes`**

```go
	domain.CodeCredentialReExtractSuccess:   http.StatusOK,
	domain.CodeCredentialReExtractNotFound:  http.StatusNotFound,
	domain.CodeCredentialReExtractNotFailed: http.StatusConflict,
```

- [ ] **Step 4: `mapper_test.go` — add the three to `allDomainCodes`.**

- [ ] **Step 5: `locales/en.json`**

```json
"success_credential_reextract": "Re-extraction enqueued successfully.",
"error_credential_reextract_not_found": "One or more credentials were not found: {{.credential_ids}}",
"error_credential_reextract_not_failed": "One or more credentials are not in a failed extraction state: {{.credential_ids}}"
```

- [ ] **Step 6: `locales/id.json`**

```json
"success_credential_reextract": "Ekstraksi ulang berhasil diantrekan.",
"error_credential_reextract_not_found": "Satu atau lebih kredensial tidak ditemukan: {{.credential_ids}}",
"error_credential_reextract_not_failed": "Satu atau lebih kredensial tidak dalam status ekstraksi gagal: {{.credential_ids}}"
```

- [ ] **Step 7: Test + commit**

```bash
go test ./infrastructure/http/responder/...
git add domain/codes.go infrastructure/http/responder/ locales/
git commit -m "feat(codes): add credential reextract codes 400500/540/541"
```

### Task 5.2: Re-extract request DTO

**Files:**
- Modify: `feature/credential/credential_request.go`
- Modify: `feature/credential/credential_request_test.go`

- [ ] **Step 1: Add the DTO**

```go
// CredentialReExtractRequest is the JSON body for POST /api/credentials/batch/reextract.
type CredentialReExtractRequest struct {
	Ids []string `json:"ids"`
}

func (r CredentialReExtractRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Ids, validation.Required, validation.Length(1, 100)),
	)
}
```

- [ ] **Step 2: Add a validation test**

```go
func TestCredentialReExtractRequest_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		r := CredentialReExtractRequest{Ids: []string{"01J0000000000000000000000A"}}
		assert.NoError(t, r.Validate())
	})
	t.Run("empty", func(t *testing.T) {
		assert.Error(t, CredentialReExtractRequest{Ids: []string{}}.Validate())
	})
	t.Run("too many", func(t *testing.T) {
		ids := make([]string, 101)
		for i := range ids { ids[i] = "x" }
		assert.Error(t, CredentialReExtractRequest{Ids: ids}.Validate())
	})
}
```

- [ ] **Step 3: Test + commit**

```bash
go test ./feature/credential/...
git add feature/credential/credential_request.go feature/credential/credential_request_test.go
git commit -m "feat(credential): add reextract request DTO + validation"
```

### Task 5.3: Re-extract service method

**Files:**
- Modify: `feature/credential/credential_service.go`

**Goal:** All-or-nothing within one UoW. Only `failed` credentials with a `file_uri` can be re-extracted. Reset `extract_status=pending`, clear `extract_error`, re-enqueue via the River enqueuer.

- [ ] **Step 1: Add to the interface**

```go
	ReExtract(ctx context.Context, ids ...string) ([]domain.Credential, error)
```

- [ ] **Step 2: Implement (helpers prefixed `reExtract*`)**

```go
func (s *credentialService) ReExtract(ctx context.Context, ids ...string) ([]domain.Credential, error) {
	if err := s.policy.VerifyPreFetch(ctx); err != nil { // Issuer+ rank reused
		return nil, err
	}
	var updated []domain.Credential
	var toEnqueue []domain.Credential
	err := s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		targets, err := uow.Credential().FindByIds(ctx, ids...)
		if err != nil {
			return err
		}
		if err := s.reExtractValidate(ids, targets); err != nil {
			return err
		}
		updates := make([]domain.Credential, len(targets))
		emptyErr := ""
		for i, t := range targets {
			updates[i] = domain.Credential{
				ID:            t.ID,
				ExtractStatus: domain.ExtractStatusPending,
				ExtractError:  &emptyErr,
			}
		}
		updated, err = uow.Credential().Update(ctx, updates...)
		if err != nil {
			return err
		}
		toEnqueue = targets
		return nil
	})
	if err != nil {
		return nil, err
	}
	// Enqueue AFTER the DB tx commits (River insert is not part of the UoW tx).
	for _, t := range toEnqueue {
		if err := s.enqueuer.EnqueueExtract(ctx, jobs.CredentialExtractArgs{
			CredentialID: t.ID, FileURI: *t.FileURI,
		}); err != nil {
			return nil, err
		}
	}
	return updated, nil
}

// reExtractValidate ensures every target exists and is in a failed state with a file.
func (s *credentialService) reExtractValidate(ids []string, targets []domain.Credential) error {
	targetIds := lo.Map(targets, func(c domain.Credential, _ int) string { return c.ID })
	if missing, _ := lo.Difference(ids, targetIds); len(missing) > 0 {
		return domain.NewError(domain.CodeCredentialReExtractNotFound,
			domain.WithMetadata("credential_ids", missing))
	}
	notFailed := []string{}
	for _, t := range targets {
		if t.ExtractStatus != domain.ExtractStatusFailed || t.FileURI == nil {
			notFailed = append(notFailed, t.ID)
		}
	}
	if len(notFailed) > 0 {
		return domain.NewError(domain.CodeCredentialReExtractNotFailed,
			domain.WithMetadata("credential_ids", notFailed))
	}
	return nil
}
```

- [ ] **Step 2b: Confirm `Update` clears `extract_error`**

Read `feature/credential/gorm_credential_repository.go` `Update`. If the per-column CASE builder skips empty-string pointers, add an `extract_error` CASE branch that emits whenever the pointer is non-nil (even if empty), so `ExtractError = &""` actually clears the column. Adjust the repo in this step and note the change in the commit.

- [ ] **Step 3: Build (handler red until 5.4); defer commit.**

### Task 5.4: Re-extract handler + route

**Files:**
- Modify: `feature/credential/credential_handler.go`
- Modify: `infrastructure/http/router.go`

- [ ] **Step 1: Add interface method + handler**

In the `CredentialHandler` interface add `ReExtract(c *gin.Context)`. Implement:

```go
func (h *credentialHandler) ReExtract(c *gin.Context) {
	var req CredentialReExtractRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	updated, err := h.credSvc.ReExtract(c.Request.Context(), req.Ids...)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responder.Send(c, domain.CodeCredentialReExtractSuccess, mapCredentialsToResponse(updated))
}
```

- [ ] **Step 2: Register the route** (match the file's existing group var + issuer middleware var names)

```go
		credentials.POST("/batch/reextract", issuerMW, credHandler.ReExtract)
```

- [ ] **Step 3: Build + test**

```bash
go build ./... && go test ./... && go vet ./... && gofmt -l .
```

- [ ] **Step 4: Commit Phase 5 bundle**

```bash
git add feature/credential/credential_service.go feature/credential/credential_handler.go infrastructure/http/router.go feature/credential/gorm_credential_repository.go
git commit -m "feat(credential): add /batch/reextract endpoint"
```

---

## Phase 6: Tests

### Task 6.1: Mongo repository mocks

**Files:**
- Create: `infrastructure/testutil/mocks/credential_extraction_repository.go`
- Create: `infrastructure/testutil/mocks/credential_verification_repository.go`
- Modify: existing mocks if `MockPythonAIClient`/`MockRegistryService` need new methods

- [ ] **Step 1: `MockCredentialExtractionRepository`**

```go
package mocks

import (
	"context"
	"CredChain_Golang/domain"
	"github.com/stretchr/testify/mock"
)

type MockCredentialExtractionRepository struct{ mock.Mock }

func (m *MockCredentialExtractionRepository) Store(ctx context.Context, e domain.CredentialExtraction) error {
	return m.Called(ctx, e).Error(0)
}
func (m *MockCredentialExtractionRepository) FindByCredentialId(ctx context.Context, id string) (*domain.CredentialExtraction, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).(*domain.CredentialExtraction), args.Error(1)
}
func (m *MockCredentialExtractionRepository) FindRankedByIds(ctx context.Context, values []string, limit int) ([]domain.CredentialExtraction, error) {
	args := m.Called(ctx, values, limit)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).([]domain.CredentialExtraction), args.Error(1)
}
```

- [ ] **Step 2: `MockCredentialVerificationRepository`**

```go
package mocks

import (
	"context"
	"CredChain_Golang/domain"
	"github.com/stretchr/testify/mock"
)

type MockCredentialVerificationRepository struct{ mock.Mock }

func (m *MockCredentialVerificationRepository) FindByUploadedFileHash(ctx context.Context, hash string) (*domain.CredentialVerification, error) {
	args := m.Called(ctx, hash)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).(*domain.CredentialVerification), args.Error(1)
}
func (m *MockCredentialVerificationRepository) Store(ctx context.Context, v domain.CredentialVerification) error {
	return m.Called(ctx, v).Error(0)
}
```

- [ ] **Step 3: Update `MockPythonAIClient`** (check `infrastructure/testutil/mocks/` first)

Add `ExtractIDs` method if `MockPythonAIClient` exists:
```go
func (m *MockPythonAIClient) ExtractIDs(ctx context.Context, file pyai.ExtractFile) ([]pyai.ExtractedID, error) {
	args := m.Called(ctx, file)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).([]pyai.ExtractedID), args.Error(1)
}
```

- [ ] **Step 4: Update `MockRegistryService`** (if it exists)

Add `FindCredentialByHash`:
```go
func (m *MockRegistryService) FindCredentialByHash(ctx context.Context, hash string) (string, bool, error) {
	args := m.Called(ctx, hash)
	return args.String(0), args.Bool(1), args.Error(2)
}
```

- [ ] **Step 5: Build + commit**

```bash
go build ./... && go test ./...
git add infrastructure/testutil/mocks/
git commit -m "test(mocks): add mongo repo mocks + pyai/registry new method mocks"
```

### Task 6.2: Verify service tests

**Files:**
- Create or modify: `feature/credential/credential_service_test.go`

Cover every path with a `newTestCredentialService(t, deps)` helper that wires all mocks:

- [ ] **Cache hit** — `verificationRepo.FindByUploadedFileHash` returns non-nil; assert the cached code is returned, `credRepo.Find` is called for the matched credential, Python is NOT called.
- [ ] **Exact authentic** — cache miss; `repo.FindByFileHashes` returns non-nil, not revoked; `registryService.FindCredentialByHash` returns `found=true`; assert `CodeCredentialVerifyAuthentic`.
- [ ] **Exact revoked** — same as above but `cred.RevokedAt` non-nil; assert `CodeCredentialVerifyRevoked`.
- [ ] **Exact integrity warning** — `FindCredentialByHash` returns `found=false`; assert `CodeCredentialVerifyIntegrityWarning`.
- [ ] **Fuzzy no-identifiers** — cache miss, no exact match, `aiClient.ExtractIDs` returns `[]`; assert `CodeCredentialVerifyNoIdentifiers`.
- [ ] **Fuzzy no-match** — `ExtractIDs` returns ids, `extractionRepo.FindRankedByIds` returns `[]`; assert `CodeCredentialVerifyNoMatch`.
- [ ] **Fuzzy tampered** — ranked returns one result, `aiClient.Verify` returns `verdict="tampered"`; assert `CodeCredentialVerifyTampered`.
- [ ] **Fuzzy suspicious** — `verdict="suspicious"` → `CodeCredentialVerifySuspicious`.
- [ ] **Fuzzy low_similarity** → `CodeCredentialVerifyLowSimilarity`.
- [ ] **Fuzzy not_similar** → `CodeCredentialVerifyNotSimilar`.
- [ ] **Tie-break: non-revoked preferred** — two ranked results same intersection count; one has `revoked_at`, one doesn't; assert non-revoked is chosen (Python called with that embedding).
- [ ] **Tie-break: newest issued_at** — both non-revoked; assert the newer one is chosen.

Example skeleton:

```go
func TestVerify_CacheHit(t *testing.T) {
	credRepo := &mocks.MockCredentialRepository{}
	verRepo := &mocks.MockCredentialVerificationRepository{}
	extRepo := &mocks.MockCredentialExtractionRepository{}
	aiClient := &mocks.MockPythonAIClient{}
	registrySvc := &mocks.MockRegistryService{}

	cid := "01JTEST0000000000000000001"
	verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).
		Return(&domain.CredentialVerification{
			VerdictCode:         domain.CodeCredentialVerifyAuthentic,
			MatchedCredentialID: &cid,
		}, nil)
	credRepo.On("Find", mock.Anything, cid, mock.Anything).
		Return(&domain.Credential{ID: cid}, nil)
	verRepo.On("Store", mock.Anything, mock.Anything).Return(nil).Maybe()

	svc := newTestCredentialService(t, testCredentialServiceDeps{
		credRepo: credRepo, verRepo: verRepo, extRepo: extRepo,
		aiClient: aiClient, registrySvc: registrySvc,
	})
	ctx := gintest.NewContext(t, gintest.WithUser(fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))))
	code, cred, _, _, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test-file")})
	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyAuthentic, code)
	assert.NotNil(t, cred)
	aiClient.AssertNotCalled(t, "ExtractIDs", mock.Anything, mock.Anything)
	aiClient.AssertNotCalled(t, "Verify", mock.Anything, mock.Anything, mock.Anything)
}
```

- [ ] **Run + commit**

```bash
go test ./feature/credential/... -v -run TestVerify
git add feature/credential/credential_service_test.go
git commit -m "test(credential): verify pipeline path coverage"
```

### Task 6.3: Issue + reextract service tests

**Files:**
- Modify: `feature/credential/credential_service_test.go`

- [ ] **Step 1: Issue partial-success** — items: [bad holder, dup hash, valid]. Wire `userRepo.FindByIds` to omit the bad holder; `repo.FindByFileHashes` returns an active row for the dup; `storage.SaveBytes` returns a path for the valid one; `uow` via `mocks.RunUnitOfWorkFn`; chain + `Store` + `Update` + enqueuer mocked. Assert: `results[0]`/`results[1]` zero-value, `results[2]` committed, `errs["credentials.0"]`/`errs["credentials.1"]` set, enqueuer called once.
- [ ] **Step 2: Issue all-failed** — all bad holders. Assert all `results` zero-value, `errs` populated, chain NOT called, enqueuer NOT called.
- [ ] **Step 3: Issue chain rollback** — valid items, `syncBlockchainIssue` returns error; use `mocks.NewPropagatingUnitOfWork`. Assert error returned, orphan files cleaned (storage delete attempted).
- [ ] **Step 4: Reextract happy path** — all targets `failed` with `file_uri`. Assert `Update` called with `pending`+cleared error, enqueuer called per target.
- [ ] **Step 5: Reextract not-found** — `FindByIds` missing one → `CodeCredentialReExtractNotFound`, enqueuer NOT called.
- [ ] **Step 6: Reextract not-failed** — a target is `succeeded` → `CodeCredentialReExtractNotFailed`, enqueuer NOT called.

- [ ] **Step 7: Run + commit**

```bash
go test ./feature/credential/... -v
git add feature/credential/credential_service_test.go
git commit -m "test(credential): issue partial-success + reextract coverage"
```

### Task 6.4: River worker test

**Files:**
- Create: `infrastructure/jobs/credential_extract_river_test.go`

- [ ] **Step 1: Success path** — write a temp file under `tmp/`; mock `aiClient.Extract` to return `{Text, IDs, Embedding}`; mock `credRepo.Find` (returns file_hash); mock `extractionRepo.Store`; mock `uow` (via `RunUnitOfWorkFn`) so `credRepo.Update` sets `succeeded`+`extracted_at`. Call `w.workExtract(ctx, args)` directly (not through River). Assert no error, `Store` + `Update` invoked.
- [ ] **Step 2: File-not-found** — args point at a missing path → error returned, no Mongo/PG writes.
- [ ] **Step 3: Empty embedding** — `Extract` returns empty embedding → error returned.
- [ ] **Step 4: Mongo store fails** — `extractionRepo.Store` returns error → error returned, PG update NOT called.

- [ ] **Step 5: Run + commit**

```bash
go test ./infrastructure/jobs/... -v
git add infrastructure/jobs/credential_extract_river_test.go
git commit -m "test(jobs): river extraction worker coverage"
```

### Task 6.5: pyai client tests

**Files:**
- Create: `infrastructure/ai/pyai/client_test.go`

- [ ] **Step 1: Extract parsing + auth header** — `httptest.Server` returns `{code:500100,data:[{text,ids,embedding}]}`; assert mapped fields and that the request carried `X-Api-Key` when configured.
- [ ] **Step 2: Extract all-failed** — server returns `code:500150` → client returns error.
- [ ] **Step 3: Verify request/response** — assert the multipart contains an `embeddings` field equal to `[[...]]` (single file), and the response `{verdict,similarity_score,similarity_percent}` maps to `VerifyResult`.
- [ ] **Step 4: ExtractIDs empty** — server returns `data:[{ids:[]}]` → returns empty slice, no error.
- [ ] **Step 5: MIME resolution** — pass `ExtractFile{Filename:"x.pdf"}` with empty MIMEType; assert the multipart part Content-Type is `application/pdf`.

- [ ] **Step 6: Run + commit**

```bash
go test ./infrastructure/ai/pyai/... -v
git add infrastructure/ai/pyai/client_test.go
git commit -m "test(pyai): contract parsing, auth header, mime resolution"
```
---

## Phase 7: AGENTS.md + Docs + Postman Collection

### Task 7.1: Update CredChain_Golang/AGENTS.md

**Files:**
- Modify: `CredChain_Golang/AGENTS.md`

- [ ] **Step 1: MongoDB section** — update the Database subsection: MongoDB is now actively used with `credential_extractions` (text, ids, embedding, file_hash) and `credential_verifications` (TTL cache). Add `infrastructure/database/mongo/client.go` as the FX provider.

- [ ] **Step 2: Critical Commands** — add `make migrate-up-mongo` and `make migrate-down-mongo`. Note that `make migrate-up` now ALSO runs River migrations programmatically.

- [ ] **Step 3: Env vars table** — add `PYTHON_AI_API_KEY`, `MONGO_DATABASE` (default `credchain`), `AI_VERIFICATION_CACHE_TTL_HOURS` (default 24), `RIVER_MAX_WORKERS` (default 10).

- [ ] **Step 4: API Routes table** — add `POST /api/credentials/batch/reextract` (Issuer+, handler `ReExtract`). Update `/api/credentials/verify`: accepts one `file`, no `credential_id`, returns verdict code 400401-400409, response includes `description` (locale-resolved from `Accept-Language`).

- [ ] **Step 5: Schema note** — `credentials.embeddings` column removed; extraction data (text, ids, embedding) lives in MongoDB `credential_extractions`.

- [ ] **Step 6: Issue/revoke semantics** — Issue is partial-success (`credentials.<index>` error keys); revoke and reextract are all-or-nothing.

- [ ] **Step 7: River job queue** — document River as the async job queue replacing the custom poll worker. `CredentialExtractArgs{CredentialID, FileURI}` is the job kind. `make migrate-up` provisions River tables. `RIVER_MAX_WORKERS` controls concurrency.

- [ ] **Step 8: Add strict NO-N+1 rule (new subsection under Key Constraints)**

Add verbatim:

```
### Strict NO-N+1 Rule

All batch repository operations (Postgres and MongoDB) MUST execute a
single query/aggregation regardless of batch size. NEVER issue queries
inside a loop over input items.

- Postgres batch updates use CASE statements (updateBatchCase).
- Mongo id-search uses ONE aggregation pipeline (FindRankedByIds:
  $match $in -> $addFields $setIntersection -> $sort -> $limit).
- Relation/tie-break lookups use a single IN-clause / $in query
  (FindByIds), never per-row fetches.
- The verify fuzzy path is bounded: 1 extract-ids call + 1 aggregation
  + 1 verify call + 1 cache write. No per-candidate queries.

Reviewers MUST reject any code path that issues queries inside a loop.
```

- [ ] **Step 9: Verdict-code taxonomy** — document codes 400401-400409 (verify verdicts, HTTP statuses: integrity-warning=409, rest=200) and 400500/540/541 (reextract, HTTP: 200/404/409).

- [ ] **Step 10: Commit**

```bash
git add CredChain_Golang/AGENTS.md
git commit -m "docs(agents): mongo, verify, reextract, river, NO-N+1 rule"
```

### Task 7.2: Postman collection update

**Files:**
- Modify: `CredChain_postman_collection.json`

- [ ] **Step 1: Update Verify request** — under `Credential > Verify`: remove the `credential_id` form field; keep only a `file` (File type) form-data field. Update auth to inherit Issuer bearer.

- [ ] **Step 2: Add Verify response examples** — one example per verdict code: 400401 authentic (200, with credential), 400402 revoked (200), 400403 integrity-warning (409), 400404 tampered (200, with credential + score), 400405 suspicious, 400406 low_similarity, 400407 not_similar, 400408 no_identifiers (200, no credential), 400409 no_match (200, no credential). Each body shows `{code, message, data:{verdict_code, similarity_score?, similarity_percent?, description, credential?}}`.

- [ ] **Step 3: Add Re-Extract request** — new request under `Credential`: `POST {{BASE_URL}}/api/credentials/batch/reextract`, JSON body `{"ids": ["{{TARGET_CREDENTIAL_ID}}"]}`, Issuer bearer auth.

- [ ] **Step 4: Add Re-Extract response examples** — 400500 success (200), 400540 not-found (404), 400541 not-failed (409).

- [ ] **Step 5: Update Issue examples** — show the partial-success envelope `{code, message, data:[...nulls for failed...], errors:{"credentials.0":[...]}}`.

- [ ] **Step 6: Commit**

```bash
git add CredChain_postman_collection.json
git commit -m "docs(postman): verify verdicts, reextract, partial-success issue"
```

### Task 7.3: Final full verification

- [ ] **Step 1: Run the full gate**

```bash
go test ./... && go vet ./... && gofmt -l .
```
Expected: all tests pass, vet clean, gofmt zero output.

- [ ] **Step 2: Manual smoke (optional; needs Postgres + Mongo + Python + RPC)**

```bash
make migrate-up        # also provisions River tables
make migrate-up-mongo
make serve
```
Exercise: issue -> (River worker extracts to Mongo) -> verify (exact + fuzzy) -> reextract a failed one.

---

## Self-Review Notes (for the executor)

- **Spec coverage:** Section 1 -> Phase 1; Section 2 -> Phase 2; Section 3 -> Phase 3; Section 4 (revoke) -> unchanged code, documented in Phase 7; Section 5 (verify) -> Phase 4; Section 6 (reextract) -> Phase 5; Section 7 (testing) -> Phase 6; Section 8 (docs) -> Phase 7.
- **River replaces the custom poll worker** — `infrastructure/jobs/credential_extract_worker.go` is deleted; `credential_extract_river.go` + `river.go` + `river_migrate.go` replace it. The `credential_extract_jobs` Postgres table and `gorm_credential_extract_job_repository.go` become unused by the new flow; LEAVE them in place (out of scope to remove) but the service no longer writes to them — it enqueues via the River `Enqueuer`. Note this in the commit.
- **Commit strategy:** one commit per task (or per phase bundle where builds are interdependent), each passing `go build` at minimum; the phase-closing commit passes the full gate `go test ./... && go vet ./... && gofmt -l .`.
- **Executor MUST confirm before coding:**
  - `responder` message-resolution helper name (Task 3.4 / 4.4) — extract an exported `ResolveMessage(c, code) string`.
  - `gorm_credential_repository.go` Update nil-vs-empty handling for `extract_error` (Task 5.3 Step 2b).
  - Exact group/middleware var names in `router.go` (Task 5.4).
  - `credRepo.Find` accepts `nil` query or needs `&domainQuery.Query{}` (Task 3.3).
  - riverpgxv5 generic type param (`*river.Client[pgx.Tx]`) matches installed version.
  - Existing `MockPythonAIClient` / `MockRegistryService` presence + method sets (Task 6.1).
