# Credential Feature — Gap Fulfillment Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close 7 gaps identified between the implementation plan and the actual codebase after the main credential feature implementation. Fix atomicity issues, missing tests, documentation drift, and stale code.

**Architecture:** Three gaps cluster around data consistency: (1) River enqueue sits outside the GORM UoW transaction → jobs can be orphaned from credential state. The root constraint: GORM uses `database/sql`, River uses `pgx` — separate connection pools cannot share a transaction. Solution: correct as far as practical without deep refactoring. (2) MongoDB write + Postgres update in the worker cannot share a transaction (different databases). Solution: document the correct idempotent retry pattern. (3) ReExtract needs a compensating action on partial enqueue failure.

**Tech Stack:** Go 1.25.1, Gin, Uber FX, GORM (Postgres, pgx/v5 driver), mongo-driver v2, riverqueue/river, testify.

**Plan:** docs/superpowers/plans/2026-06-07-credential-feature-python-integration.md

---

## Gap 1: River Enqueue Atomicity — Document + Fix Issue ReExtract Compensating Action

### Task 1.1: Document the GORM/River transaction boundary

**Files:**
- Modify: `infrastructure/jobs/river.go`
- Modify: `feature/credential/credential_service.go`

**Context:** GORM uses `database/sql` + pgx/v5 driver. River uses direct pgx connections via `pgxpool`. They share the same Postgres server but use independent connection pools. A `*sql.Tx` and a `pgx.Tx` are incompatible wire-level structures — they cannot be bridged without extracting the underlying `pgx.Conn` from GORM's `sql.Tx` (which is fragile and not recommended).

**Impact on Issue:** The `issueEnqueueExtractJob` call IS inside the UoW closure body, but uses `client.Insert(ctx, args)` (non-transactional). If the server crashes between `uow.Credential().Update()` and the enqueue, the credential gets `token_id` set but no River job. This is a rare-but-possible window. Mitigation: the credential remains in `extract_status=pending` and the reextract endpoint exists to recover.

**Impact on ReExtract:** Enqueues happen AFTER the UoW commits. If enqueue fails for credential N after credentials 0..N-1 were enqueued, credential N is stuck in `pending` with no job. This is the `stuck-credential` risk.

The code already has comments documenting these tradeoffs. **No code change needed for Issue** — just add a clearer comment in `issueEnqueueExtractJob` and the Issue method's UoW body.

- [ ] **Step 1: Add cross-DB transaction comment to `issueEnqueueExtractJob`**

In `feature/credential/credential_service.go`, update the comment on `issueEnqueueExtractJob`:

```go
// issueEnqueueExtractJob enqueues a River extraction job.
// River jobs live in Postgres (river_jobs table) but use a separate connection
// pool (pgx) from GORM's (database/sql + pgx). They cannot share a transaction.
// This means a credential can be committed without its extraction job (rare:
// server crash between Update and Insert). Mitigation: the credential stays in
// extract_status=pending and the reextract endpoint can recover it.
func (s *credentialService) issueEnqueueExtractJob(ctx context.Context, credentialID, fileURI string) error {
	return s.enqueuer.EnqueueExtract(ctx, jobs.CredentialExtractArgs{CredentialID: credentialID, FileURI: fileURI})
}
```

- [ ] **Step 2: Add the same comment to the Issue method's enqueue call site**

Find the enqueue loop in `Issue` (line ~283) and add a brief inline comment referencing the compensatory documentation.

- [ ] **Step 3: Verify**

```bash
go build ./...
go vet ./...
```

- [ ] **Step 4: Commit**

```bash
git add feature/credential/credential_service.go
git commit -m "docs(credential): document GORM/River transaction boundary for enqueues"
```

### Task 1.2: Fix ReExtract stuck-credential risk + add dedicated ReExtractPreFetch policy

**Files:**
- Modify: `feature/credential/credential_policy.go`
- Modify: `feature/credential/credential_service.go`

**Problem A (policy):** `ReExtract` currently calls `s.policy.VerifyPreFetch(ctx)` — piggybacking on the verify policy. It should have its own `ReExtractPreFetch` method so future ReExtract-specific rules (e.g., only the original issuer may re-extract) can be added without coupling to verify.

**Problem B (atomicity):** ReExtract resets credentials to `pending` inside a UoW. If the River enqueue in the subsequent loop fails for item N, items 0..N-1 are in `pending` with jobs, but item N is in `pending` with no job. The next reextract call will fail with `CodeCredentialReExtractNotFailed` (status is now `pending`, not `failed`). The credential is stuck.

**Fix B:** If enqueue fails, perform a compensating update: re-stamp the failed-to-enqueue credential back to `failed` with the original error. The already-enqueued ones stay in `pending` (they have jobs).

- [ ] **Step 1: Add `ReExtractPreFetch` to the policy interface**

In `feature/credential/credential_policy.go`, add to the `CredentialPolicy` interface (after `VerifyPreFetch`):

```go
	// ReExtractPreFetch checks signer rank only (Issuer+).
	ReExtractPreFetch(ctx context.Context) error
```

- [ ] **Step 2: Implement `ReExtractPreFetch` on `credentialPolicy`**

Add the method (mirrors `VerifyPreFetch`, kept separate for future ReExtract-specific rules):

```go
func (p *credentialPolicy) ReExtractPreFetch(ctx context.Context) error {
	if !signerIsIssuerOrAbove(ctx) {
		return domain.NewError(domain.CodeAuthForbidden)
	}
	return nil
}
```

- [ ] **Step 3: Modify ReExtract to use ReExtractPreFetch + compensate on enqueue failure**

```go
func (s *credentialService) ReExtract(ctx context.Context, ids ...string) ([]domain.Credential, error) {
	if err := s.policy.ReExtractPreFetch(ctx); err != nil {
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
		for i, t := range targets {
			emptyErr := ""
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
	// Enqueue River jobs. If enqueue fails, compensate: re-stamp the failed
	// credential back to failed so the operator can retry the whole batch.
	for _, t := range toEnqueue {
		if err := s.enqueuer.EnqueueExtract(ctx, jobs.CredentialExtractArgs{
			CredentialID: t.ID, FileURI: *t.FileURI,
		}); err != nil {
			// Compensate: re-stamp this credential as failed so retry is possible.
			// Earlier credentials in this batch remain in pending with active jobs
			// and will be extracted normally by the River worker.
			if compErr := s.reExtractCompensate(ctx, t); compErr != nil {
				s.logger.Error("reextract compensate failed",
					zap.String("credential_id", t.ID), zap.Error(compErr))
			}
			return nil, err
		}
	}
	return updated, nil
}

// reExtractCompensate restamps a credential back to ExtractStatusFailed after
// a failed enqueue attempt so the operator can retry reextraction.
func (s *credentialService) reExtractCompensate(ctx context.Context, t domain.Credential) error {
	errMsg := "reenqueue failed"
	if t.ExtractError != nil {
		errMsg = *t.ExtractError
	}
	_, err := s.repo.Update(ctx, domain.Credential{
		ID:            t.ID,
		ExtractStatus: domain.ExtractStatusFailed,
		ExtractError:  &errMsg,
	})
	return err
}
```

- [ ] **Step 4: Update the policy mock**

`infrastructure/testutil/mocks/` does not currently have a `MockCredentialPolicy` (the service tests use the real `&credentialPolicy{}`). No mock change needed — but if one exists, add `ReExtractPreFetch`. Confirm by grepping `MockCredentialPolicy`.

```bash
go build ./...
go test ./... -run TestReExtract
```

- [ ] **Step 3: Commit**

```bash
git add feature/credential/credential_service.go
git commit -m "fix(credential): compensating re-stamp on ReExtract enqueue failure"
```

---

## Gap 2: Document MongoDB Worker Write Ordering

### Task 2.1: Document the idempotent retry pattern in the worker

**Files:**
- Modify: `infrastructure/jobs/credential_extract_river.go`

**Context:** The worker's `workExtract` does:
1. Read file from storage (idempotent)
2. Call Python `/extract` (idempotent for same file content)
3. Mongo `extractionRepo.Store` — idempotent upsert
4. Postgres `credRepo.Update` — single-row update

Cross-database atomicity is impossible: Mongo and Postgres cannot share a transaction. The pattern is correct because:
- Step 3 is an upsert by `credential_id` — re-running overwrites the same document harmlessly
- If step 4 fails → River retries the entire job → step 3 re-runs safely
- Each step is individually atomic (single-document upsert is atomic in MongoDB; single-row UPDATE is atomic in Postgres)

- [ ] **Step 1: Add documentation comment to `workExtract`**

Update the comment on `workExtract` and add inline notes:

```go
// workExtract reads the file, calls Python, writes Mongo, updates Postgres lifecycle.
//
// Atomicity notes:
// - MongoDB single-document upsert (Store) is atomically executed by MongoDB itself.
// - Postgres single-row Update is atomically executed by Postgres.
// - Cross-database atomicity is not possible (Mongo + Postgres cannot share a
//   transaction). The design uses an idempotent pattern: Mongo write FIRST, then
//   Postgres. If Postgres fails, River retries the whole job. Since the Mongo
//   upsert is keyed by credential_id, re-running is safe (same data overwrites).
// - Each step before Postgres is idempotent: os.ReadFile, aiClient.Extract,
//   credRepo.Find, extractionRepo.Store.
// Helper prefixed "work" (the method name).
func (w *CredentialExtractWorker) workExtract(ctx context.Context, args CredentialExtractArgs) error {
```

- [ ] **Step 2: Verify**

```bash
go build ./...
go vet ./...
```

- [ ] **Step 3: Commit**

```bash
git add infrastructure/jobs/credential_extract_river.go
git commit -m "docs(worker): document cross-DB atomicity pattern and idempotent retry design"
```

---

## Gap 3: Missing Tests — AllFailed, ChainRollback, Real Partial Success

### Task 3.1: Add TestIssue_AllFailed

**Files:**
- Modify: `feature/credential/credential_service_test.go`

- [ ] **Step 1: Write the test**

```go
func TestIssue_AllFailed(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)
	enq := &localMockEnqueuer{}
	userRepo := &mocks.MockUserRepository{}
	stor := &storage.Storage{BaseDir: t.TempDir()}

	// Return empty holders — every item's holder will be "not found".
	userRepo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{}, nil)

	svc := &credentialService{
		storage:  stor,
		policy:   &credentialPolicy{},
		userRepo: userRepo,
		logger:   zap.NewNop(),
		enqueuer: enq,
	}
	items := []CredentialIssuance{
		{HolderUserID: "holder-1", Name: "a", Filename: "x.pdf", FileBytes: []byte("x")},
		{HolderUserID: "holder-2", Name: "b", Filename: "x.pdf", FileBytes: []byte("y")},
	}
	results, errs, err := svc.Issue(ctx, items)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "", results[0].ID)
	assert.Equal(t, "", results[1].ID)
	assert.Contains(t, errs, "credentials.0")
	assert.Contains(t, errs, "credentials.1")
	enq.AssertNotCalled(t, "EnqueueExtract", mock.Anything, mock.Anything)
}
```

- [ ] **Step 2: Run**

```bash
go test ./feature/credential/... -v -run TestIssue_AllFailed
```
Expected: PASS.

- [ ] **Step 3: Commit (with Task 3.2).**

### Task 3.2: Add TestIssue_ChainRollback

- [ ] **Step 1: Write the test**

```go
func TestIssue_ChainRollback(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)
	enq := &localMockEnqueuer{}
	userRepo := &mocks.MockUserRepository{}
	stor := &storage.Storage{BaseDir: t.TempDir()}

	userRepo.On("FindByIds", mock.Anything, mock.Anything).Return(
		[]domain.User{{Id: "holder-valid"}}, nil)

	// Simulate chain sync failure: RegistryService.IssueCredentials returns error
	regSvc := &mocks.MockRegistryService{}
	regSvc.On("IssueCredentials", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, assert.AnError)

	uow := mocks.NewPropagatingUnitOfWork()
	innerCredRepo := &mocks.MockCredentialRepository{}
	innerCredRepo.On("Store", mock.Anything, mock.Anything, mock.Anything).Return(
		[]domain.Credential{{ID: "stored-1", FileURI: lo.ToPtr("up/test.pdf")}}, nil)
	uow.On("Credential").Return(innerCredRepo)
	uow.On("CredentialExtractJob").Return(&mocks.MockCredentialExtractJobRepository{})

	svc := &credentialService{
		repo:            &mocks.MockCredentialRepository{},
		uow:             uow,
		registryService: regSvc,
		storage:         stor,
		policy:          &credentialPolicy{},
		userRepo:        userRepo,
		logger:          zap.NewNop(),
		enqueuer:        enq,
	}
	items := []CredentialIssuance{
		{HolderUserID: "holder-valid", Name: "doc", Filename: "x.pdf", FileBytes: []byte("test")},
	}
	_, _, err := svc.Issue(ctx, items)
	assert.Error(t, err)
}
```

- [ ] **Step 2: Run**

```bash
go test ./feature/credential/... -v -run TestIssue_ChainRollback
```
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add feature/credential/credential_service_test.go
git commit -m "test(credential): issue all-failed + chain-rollback coverage"
```

### Task 3.3: Fix TestIssue_PartialSuccess to test actual partial scenario

**Problem:** The current test only has one valid item. It should test a mix: bad holder + valid, to verify the partial-success path (some items fail, survivors succeed).

- [ ] **Step 1: Rewrite TestIssue_PartialSuccess**

Replace the body with a version that has 3 items: holder-1 (bad, not returned by FindByIds), holder-2 (valid). Assert that only holder-2's credential is committed and the error map contains the bad-holder entry.

```go
func TestIssue_PartialSuccess(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)
	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}
	enq := &localMockEnqueuer{}

	stor := &storage.Storage{BaseDir: t.TempDir()}

	// Only return holder-2 — holder-1 is "bad" (not found).
	userRepo := &mocks.MockUserRepository{}
	userRepo.On("FindByIds", mock.Anything, mock.Anything).
		Return([]domain.User{{Id: "holder-2"}}, nil)
	m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything).Return(nil, nil)

	uow := mocks.NewPropagatingUnitOfWork()
	innerCredRepo := &mocks.MockCredentialRepository{}
	innerCredRepo.On("Store", mock.Anything, mock.Anything, mock.Anything).Return(
		[]domain.Credential{{ID: "stored-2", FileURI: lo.ToPtr("up/test.pdf")}}, nil)
	innerCredRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(
		[]domain.Credential{{ID: "stored-2", TokenID: lo.ToPtr("1")}}, nil)
	uow.On("Credential").Return(innerCredRepo)
	uow.On("CredentialExtractJob").Return(&mocks.MockCredentialExtractJobRepository{})
	m.regSvc.On("IssueCredentials", mock.Anything, mock.Anything, mock.Anything).
		Return([]*big.Int{big.NewInt(1)}, nil)
	enq.On("EnqueueExtract", mock.Anything, mock.Anything).Return(nil)

	svc := &credentialService{
		repo:            m.credRepo,
		uow:             uow,
		registryService: m.regSvc,
		aiClient:        m.aiClient,
		storage:         stor,
		policy:          &credentialPolicy{},
		userRepo:        userRepo,
		logger:          zap.NewNop(),
		enqueuer:        enq,
	}

	items := []CredentialIssuance{
		{HolderUserID: "holder-1", Name: "bad", Filename: "x.pdf", FileBytes: []byte("x")},
		{HolderUserID: "holder-2", Name: "valid", Filename: "x.pdf", FileBytes: []byte("b")},
	}
	results, errs, err := svc.Issue(ctx, items)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Contains(t, errs, "credentials.0")
	assert.Equal(t, "", results[0].ID)          // failed item — zero value
	assert.Equal(t, "stored-2", results[1].ID)   // survivor committed
	enq.AssertNumberOfCalls(t, "EnqueueExtract", 1)
}
```

- [ ] **Step 2: Run**

```bash
go test ./feature/credential/... -v -run TestIssue_PartialSuccess
```
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add feature/credential/credential_service_test.go
git commit -m "test(credential): real partial-success case (bad holder + valid item)"
```

---

## Gap 4: AGENTS.md Route Auth — All Credential Routes = Issuer+

### Task 4.1: Fix AGENTS.md route auth column

**Files:**
- Modify: `AGENTS.md`

**Context:** All credential routes (`GET /api/credentials`, `GET /api/credentials/:id`, `POST /batch/issue`, `POST /batch/revoke`, `POST /batch/reextract`, `POST /verify`) are registered with the `IssuerRoleMiddleware`. AGENTS.md incorrectly listed the two GET routes as `Admin+`.

- [ ] **Step 1: Update AGENTS.md route table**

Edit lines 364-365 in AGENTS.md — change both `Admin+` entries to `Issuer+`. The full credential route block should read:

```
| GET | `/api/credentials` | Issuer+ | List credentials |
| GET | `/api/credentials/:id` | Issuer+ | Single credential |
| POST | `/api/credentials/batch/issue` | Issuer+ | Issue credentials |
| POST | `/api/credentials/batch/revoke` | Issuer+ | Revoke credentials |
| POST | `/api/credentials/batch/reextract` | Issuer+ | Re-extract failed credentials |
| POST | `/api/credentials/verify` | Issuer+ | Returns verdict code (400401-400409) + locale description |
```

- [ ] **Step 2: Commit**

```bash
git add AGENTS.md
git commit -m "docs(agents): fix credential route auth to Issuer+ for list and find"
```

---

## Gap 5: Postman Verify Verdict Examples

### Task 5.1: Add remaining verify verdict response examples

**Files:**
- Modify: `CredChain_postman_collection.json`

- [ ] **Step 1: Count existing examples**

Check how many `verdict_code` examples exist in the Postman collection:

```bash
grep -c "verdict_code" CredChain_postman_collection.json
```

Expected: fewer than 9.

- [ ] **Step 2: Add the missing verdict code examples**

For verdicts 400401-400407 (authentic through not_similar), each example should have:
- HTTP status (200 or 409 for integrity_warning)
- Body: `{"code": <verdict_code>, "message": "<locale message from en.json>", "data": {"verdict_code": <code>, "similarity_score"?: ..., "similarity_percent"?: ..., "description": "...", "credential"?: {...}}}`.

For verdicts 400408 (no_identifiers) and 400409 (no_match), `data` is omitted per spec.

Add missing examples to the Postman collection JSON. Each example must be in a `response` array on the `/verify` request.

- [ ] **Step 3: Validate JSON**

```bash
python3 -m json.tool CredChain_postman_collection.json > /dev/null
```
Expected: exit 0.

- [ ] **Step 4: Commit**

```bash
git add CredChain_postman_collection.json
git commit -m "docs(postman): add all 9 verify verdict response examples"
```

---

## Gap 6: Remove Dead Code — `credential_extract_jobs` Table/Repository (Full Removal)

**Context:** Since River replaced the custom poll worker, the entire `credential_extract_jobs` stack is dead code: the Postgres table + ENUM, GORM model, domain entity + repository interface, GORM repository impl, UoW accessor, mock, and SQLite test auto-migrate entry. Remove it fully across all 12 referencing files. Split into 4 sub-tasks so the build is restored at the end of the sequence (it will be red in the middle — these sub-tasks land as ONE commit at the end of Task 6.4).

Referencing files (from `grep -rln "CredentialExtractJob\|credential_extract_job\|EnqueueExtractJob" --include="*.go"`):
`cmd/server.go`, `feature/credential/credential_service_test.go`, `feature/credential/credential_service.go`, `feature/credential/gorm_credential_extract_job_repository.go`, `infrastructure/database/gorm/uow.go`, `infrastructure/database/gorm/model/credential_extract_job.go`, `infrastructure/database/gorm/uow_test.go`, `infrastructure/testutil/mocks/mock_credential_extract_job_repository.go`, `infrastructure/testutil/mocks/mock_unit_of_work.go`, `infrastructure/testutil/db/sqlite.go`, `domain/uow.go`, `domain/credential_extract_job.go`. Plus migrations.

### Task 6.1: Delete the model, migration, and SQLite auto-migrate entries

**Files:**
- Delete: `infrastructure/database/gorm/model/credential_extract_job.go`
- Modify: `infrastructure/database/migrations/000001_initial_schema.up.sql` (remove the `credential_extract_jobs` table + its indexes + the `credential_extract_job_status` ENUM type)
- Modify: `infrastructure/database/migrations/000001_initial_schema.down.sql` (remove the matching `DROP TABLE credential_extract_jobs` + `DROP TYPE credential_extract_job_status`)
- Modify: `infrastructure/testutil/db/sqlite.go` (remove `model.CredentialExtractJob` from the `AutoMigrate(...)` list)

- [ ] **Step 1: Delete the model file**

```bash
git rm infrastructure/database/gorm/model/credential_extract_job.go
```

- [ ] **Step 2: Remove SQL table + ENUM**

In `000001_initial_schema.up.sql`, delete the `CREATE TYPE credential_extract_job_status AS ENUM (...)` block, the `CREATE TABLE credential_extract_jobs (...)` block, and any `CREATE INDEX ... ON credential_extract_jobs (...)` statements. In `000001_initial_schema.down.sql`, delete the matching `DROP` statements for that table and ENUM.

- [ ] **Step 3: Remove SQLite auto-migrate entry**

In `infrastructure/testutil/db/sqlite.go`, remove `&model.CredentialExtractJob{}` from the `db.AutoMigrate(...)` argument list.

- [ ] **Step 4: Defer commit (build is red until 6.4).**

### Task 6.2: Delete the domain entity, repository interface, GORM impl, and mock

**Files:**
- Delete: `domain/credential_extract_job.go`
- Delete: `feature/credential/gorm_credential_extract_job_repository.go`
- Delete: `infrastructure/testutil/mocks/mock_credential_extract_job_repository.go`

- [ ] **Step 1: Delete the three files**

```bash
git rm domain/credential_extract_job.go
git rm feature/credential/gorm_credential_extract_job_repository.go
git rm infrastructure/testutil/mocks/mock_credential_extract_job_repository.go
```

- [ ] **Step 2: Defer commit (build is red until 6.4).**

### Task 6.3: Remove the UoW accessor + factory wiring

**Files:**
- Modify: `domain/uow.go` (remove `CredentialExtractJob() CredentialExtractJobRepository` from the `UnitOfWork` interface)
- Modify: `infrastructure/database/gorm/uow.go` (remove the `CredentialExtractJob()` method implementation AND the factory function parameter that constructs the extract-job repo; check `NewGormUnitOfWork`'s signature and the struct field)
- Modify: `infrastructure/testutil/mocks/mock_unit_of_work.go` (remove the `CredentialExtractJob()` mock method)
- Modify: `cmd/server.go` (remove `credential.NewGormCredentialExtractJobRepository` from `fx.Provide(...)` AND remove its factory argument from the `NewGormUnitOfWork(...)` closure)

- [ ] **Step 1: Remove from the domain interface**

In `domain/uow.go`, delete the line:
```go
	CredentialExtractJob() CredentialExtractJobRepository
```

- [ ] **Step 2: Remove the GORM UoW implementation + factory param**

In `infrastructure/database/gorm/uow.go`: delete the `CredentialExtractJob()` method, the struct field holding the extract-job repo factory, and the corresponding parameter in `NewGormUnitOfWork`. Read the file first — the factory takes repo constructors as parameters; remove only the extract-job one.

- [ ] **Step 3: Remove the mock method**

In `infrastructure/testutil/mocks/mock_unit_of_work.go`, delete the `func (m *MockUnitOfWork) CredentialExtractJob() ...` method.

- [ ] **Step 4: Remove FX wiring**

In `cmd/server.go`: remove `credential.NewGormCredentialExtractJobRepository,` from the `fx.Provide(...)` list, and remove its argument from the `gormInfra.NewGormUnitOfWork(...)` factory closure (the closure that builds the UoW passes repo constructors — drop the extract-job one).

- [ ] **Step 5: Defer commit (build is red until 6.4).**

### Task 6.4: Clean up remaining callers, build, and commit the whole removal

**Files:**
- Modify: `feature/credential/credential_service.go` (if any `uow.CredentialExtractJob()` calls remain — they should NOT after Phase 3's River migration, but grep to confirm)
- Modify: `feature/credential/credential_service_test.go` (remove all `uow.On("CredentialExtractJob").Return(...)` mock setup lines)
- Modify: `infrastructure/database/gorm/uow_test.go` (remove any test asserting the `CredentialExtractJob()` accessor)
- Modify: `AGENTS.md` (remove any mention of `credential_extract_jobs`)

- [ ] **Step 1: Grep for all remaining references**

```bash
grep -rn "CredentialExtractJob\|credential_extract_job\|EnqueueExtractJob" --include="*.go" .
```
Remove every remaining reference (mock setups in tests, UoW test assertions, any service call sites).

- [ ] **Step 2: Remove AGENTS.md references**

Grep AGENTS.md for `credential_extract_job` and remove any line/sentence referencing the old table or repository.

- [ ] **Step 3: Build, test, vet, gofmt**

```bash
go build ./...
go test ./...
go vet ./...
gofmt -l .
```
All must pass; gofmt zero output.

- [ ] **Step 4: Confirm zero references remain**

```bash
grep -rn "CredentialExtractJob\|credential_extract_job\|EnqueueExtractJob" --include="*.go" .
```
Expected: zero results.

- [ ] **Step 5: Commit the full removal as one commit**

```bash
git add -A
git commit -m "refactor(credential): remove dead credential_extract_jobs table, repo, model, UoW accessor"
```

---

## Gap 7: Verify Final Gate

### Task 7.1: Full verification

- [ ] **Step 1: Run full gate**

```bash
go test ./... && go vet ./... && gofmt -l .
```
Expected: all pass, zero gofmt output.

- [ ] **Step 2: Manual check** — read the final implementation for consistency

```bash
git log --oneline HEAD~10..HEAD
```

---

## Self-Review

1. **Spec coverage:** The original plan covered all major features. Gaps found during post-implementation review fall into documentation, test completeness, and edge-case handling (not missing features).

2. **Placeholder scan:** No TBD/TODO in this plan. Every task has concrete code.

3. **Type consistency:** The compensating update function uses `s.repo.Update` (the unscoped credential repository), which is distinct from the UoW-scoped `uow.Credential().Update` used inside the transaction. This is intentional — the compensation happens OUTSIDE the UoW (after it committed), so it uses the repository directly. Same signature.

4. **Tasks are bite-sized:** Each task touches 1-2 files, has ≤3 steps, and produces a single commit.
