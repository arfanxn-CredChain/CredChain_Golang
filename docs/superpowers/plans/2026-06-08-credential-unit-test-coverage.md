# Credential Unit Test Coverage Improvement Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Raise unit-test coverage of the `feature/credential` package (currently 27.5%) and the River worker terminal-failure paths in `infrastructure/jobs` (26%) by covering all genuinely unit-testable code (pure logic + mockable collaborators), excluding code that requires a real MongoDB connection.

**Architecture:** White-box, in-package tests (`package credential`, `package jobs`) using `stretchr/testify/assert` + `mock`. Reuse the existing `testutil/mocks`, `testutil/fixtures`, and the `ctxWithAuth` helper already in `credential_service_test.go`. GORM repository tests use the in-memory SQLite harness (`testutil/db.OpenInMemorySQLite`) — the repo's established unit-test pattern. MongoDB repositories are explicitly out of scope (require a live Mongo server = integration).

**Tech Stack:** Go 1.25.1, testify, glebarez/sqlite (in-memory), go.uber.org/zap (NewNop in tests).

**Out of scope:** `mongo_credential_extraction_repository.go`, `mongo_credential_verification_repository.go` (need real Mongo); `infrastructure/jobs/river.go` `NewRiverClient`/`MigrateRiver` (need real Postgres + pgx pool).

---

## Task 1: Request DTO validation + ToDomain tests

**Files:**
- Modify: `feature/credential/credential_request_test.go` (file exists — has `TestCredentialReExtractRequest_Validate`)

Covers `credential_request.go`: `CredentialIssueInput.Validate`/`ToDomain`, `CredentialIssueRequest.Validate`/`ToDomain`, `CredentialRevokeRequest.Validate` (all currently 0%).

- [ ] **Step 1: Read the request DTOs**

Run: `sed -n '1,96p' feature/credential/credential_request.go`
Note the exact struct fields and validation rules so the test inputs match (e.g. `Name` length 1–256, `Ids` length 1–100, `HolderUserID` required).

- [ ] **Step 2: Add the tests**

Append to `feature/credential/credential_request_test.go`:

```go
func TestCredentialIssueInput_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		in := CredentialIssueInput{HolderUserID: "01J0000000000000000000000A", Name: "Degree"}
		assert.NoError(t, in.Validate())
	})
	t.Run("missing holder", func(t *testing.T) {
		in := CredentialIssueInput{Name: "Degree"}
		assert.Error(t, in.Validate())
	})
	t.Run("missing name", func(t *testing.T) {
		in := CredentialIssueInput{HolderUserID: "01J0000000000000000000000A"}
		assert.Error(t, in.Validate())
	})
	t.Run("name too long", func(t *testing.T) {
		in := CredentialIssueInput{HolderUserID: "h", Name: strings.Repeat("a", 257)}
		assert.Error(t, in.Validate())
	})
}

func TestCredentialIssueInput_ToDomain(t *testing.T) {
	meta := map[string]any{"k": "v"}
	in := CredentialIssueInput{HolderUserID: "holder-1", Name: "Degree", Meta: meta}
	got := in.ToDomain()
	assert.Equal(t, "holder-1", got.HolderUserID)
	assert.Equal(t, "Degree", got.Name)
	assert.Equal(t, meta, got.Meta)
}

func TestCredentialIssueRequest_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		r := CredentialIssueRequest{Items: []CredentialIssueInput{
			{HolderUserID: "h", Name: "n"},
		}}
		assert.NoError(t, r.Validate())
	})
	t.Run("empty items", func(t *testing.T) {
		r := CredentialIssueRequest{Items: []CredentialIssueInput{}}
		assert.Error(t, r.Validate())
	})
	t.Run("too many items", func(t *testing.T) {
		items := make([]CredentialIssueInput, 101)
		for i := range items {
			items[i] = CredentialIssueInput{HolderUserID: "h", Name: "n"}
		}
		r := CredentialIssueRequest{Items: items}
		assert.Error(t, r.Validate())
	})
	t.Run("invalid nested item", func(t *testing.T) {
		r := CredentialIssueRequest{Items: []CredentialIssueInput{{Name: "no-holder"}}}
		assert.Error(t, r.Validate())
	})
}

func TestCredentialIssueRequest_ToDomain(t *testing.T) {
	r := CredentialIssueRequest{Items: []CredentialIssueInput{
		{HolderUserID: "h1", Name: "n1"},
		{HolderUserID: "h2", Name: "n2"},
	}}
	got := r.ToDomain()
	assert.Len(t, got, 2)
	assert.Equal(t, "h1", got[0].HolderUserID)
	assert.Equal(t, "n2", got[1].Name)
}

func TestCredentialRevokeRequest_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		assert.NoError(t, CredentialRevokeRequest{Ids: []string{"01J0"}}.Validate())
	})
	t.Run("empty", func(t *testing.T) {
		assert.Error(t, CredentialRevokeRequest{Ids: []string{}}.Validate())
	})
	t.Run("too many", func(t *testing.T) {
		ids := make([]string, 101)
		for i := range ids {
			ids[i] = "x"
		}
		assert.Error(t, CredentialRevokeRequest{Ids: ids}.Validate())
	})
}
```

- [ ] **Step 3: Ensure `strings` is imported**

If the test file doesn't import `strings`, add it to the import block.

- [ ] **Step 4: Run**

```bash
go test ./feature/credential/... -run "TestCredentialIssueInput|TestCredentialIssueRequest|TestCredentialRevokeRequest" -v
```
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
gofmt -w feature/credential/credential_request_test.go
git add feature/credential/credential_request_test.go
git commit -m "test(credential): request DTO validate + toDomain coverage"
```

---

## Task 2: Policy method coverage (Revoke + forbidden branches)

**Files:**
- Create: `feature/credential/credential_policy_test.go`

Covers `credential_policy.go`: `RevokePreFetch`, `RevokePostFetch`, plus the forbidden branch of `IssuePreFetch`, `VerifyPreFetch`, `ReExtractPreFetch` (all currently 66.7% — only happy path tested via service tests).

- [ ] **Step 1: Write the test file**

```go
package credential

import (
	"context"
	"testing"

	"CredChain_Golang/domain"
	httpContext "CredChain_Golang/infrastructure/http/context"
	"CredChain_Golang/infrastructure/testutil/fixtures"

	"github.com/stretchr/testify/assert"
)

func ctxWithRole(role domain.Role) context.Context {
	user := fixtures.NewDomainUser(fixtures.WithRole(role))
	return context.WithValue(context.Background(), httpContext.UserKey, &user)
}

func TestCredentialPolicy_IssuePreFetch_Forbidden(t *testing.T) {
	p := &credentialPolicy{}
	err := p.IssuePreFetch(ctxWithRole(domain.RoleHolder), nil)
	assert.Error(t, err)
}

func TestCredentialPolicy_VerifyPreFetch_Forbidden(t *testing.T) {
	p := &credentialPolicy{}
	err := p.VerifyPreFetch(ctxWithRole(domain.RoleHolder))
	assert.Error(t, err)
}

func TestCredentialPolicy_ReExtractPreFetch_Forbidden(t *testing.T) {
	p := &credentialPolicy{}
	err := p.ReExtractPreFetch(ctxWithRole(domain.RoleHolder))
	assert.Error(t, err)
}

func TestCredentialPolicy_RevokePreFetch(t *testing.T) {
	p := &credentialPolicy{}
	t.Run("issuer ok", func(t *testing.T) {
		assert.NoError(t, p.RevokePreFetch(ctxWithRole(domain.RoleIssuer), []string{"id"}))
	})
	t.Run("holder forbidden", func(t *testing.T) {
		assert.Error(t, p.RevokePreFetch(ctxWithRole(domain.RoleHolder), []string{"id"}))
	})
}

func TestCredentialPolicy_RevokePostFetch(t *testing.T) {
	p := &credentialPolicy{}
	// Currently no rules — always nil. Documented test pins the contract.
	assert.NoError(t, p.RevokePostFetch(context.Background(), nil))
}

func TestNewCredentialPolicy(t *testing.T) {
	p := NewCredentialPolicy(CredentialPolicyParams{})
	assert.NotNil(t, p)
}
```

- [ ] **Step 2: Run**

```bash
go test ./feature/credential/... -run TestCredentialPolicy -v
```
Expected: all PASS.

- [ ] **Step 3: Commit**

```bash
gofmt -w feature/credential/credential_policy_test.go
git add feature/credential/credential_policy_test.go
git commit -m "test(credential): policy forbidden branches + Revoke pre/post coverage"
```

---

## Task 3: Service `Find` + `Revoke` + `reExtractCompensate` + `syncBlockchainRevoke` tests

**Files:**
- Modify: `feature/credential/credential_service_test.go`

Covers `credential_service.go` 0% / low-coverage methods: `Find` (not-found + error path), `Revoke` (happy path + already-revoked + chain rollback), `reExtractCompensate`, `syncBlockchainRevoke` (invalid token id + chain error + empty input).

- [ ] **Step 1: Add Find tests**

```go
func TestFind_NotFound(t *testing.T) {
	credRepo := &mocks.MockCredentialRepository{}
	credRepo.On("Find", mock.Anything, "missing", mock.Anything).Return(nil, gorm.ErrRecordNotFound)
	svc := &credentialService{repo: credRepo, logger: zap.NewNop()}
	_, err := svc.Find(context.Background(), "missing", nil)
	var domErr *domain.Error
	if assert.ErrorAs(t, err, &domErr) {
		assert.Equal(t, domain.CodeCredentialFetchNotFound, domErr.Code)
	}
}

func TestFind_HappyPath(t *testing.T) {
	credRepo := &mocks.MockCredentialRepository{}
	credRepo.On("Find", mock.Anything, "cred-1", mock.Anything).Return(&domain.Credential{ID: "cred-1"}, nil)
	svc := &credentialService{repo: credRepo, logger: zap.NewNop()}
	got, err := svc.Find(context.Background(), "cred-1", nil)
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, "cred-1", got.ID)
}

func TestFind_RepoError(t *testing.T) {
	credRepo := &mocks.MockCredentialRepository{}
	credRepo.On("Find", mock.Anything, "cred-x", mock.Anything).Return(nil, assert.AnError)
	svc := &credentialService{repo: credRepo, logger: zap.NewNop()}
	_, err := svc.Find(context.Background(), "cred-x", nil)
	assert.Error(t, err)
}
```

Add `"gorm.io/gorm"` to imports if not present.

- [ ] **Step 2: Add Revoke tests**

```go
func TestRevoke_HappyPath(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	tokID := "1"
	targets := []domain.Credential{{ID: "c1", TokenID: &tokID}}
	innerCredRepo := &mocks.MockCredentialRepository{}
	innerCredRepo.On("FindByIds", mock.Anything, []string{"c1"}).Return(targets, nil)
	innerCredRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(targets, nil)

	uow := mocks.NewPropagatingUnitOfWork()
	uow.On("Credential").Return(innerCredRepo)

	regSvc := &mocks.MockRegistryService{}
	regSvc.On("RevokeCredentials", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	svc := &credentialService{
		uow:             uow,
		registryService: regSvc,
		policy:          &credentialPolicy{},
		logger:          zap.NewNop(),
	}
	revoked, err := svc.Revoke(ctx, "c1")
	assert.NoError(t, err)
	assert.Len(t, revoked, 1)
}

func TestRevoke_NotFound(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	innerCredRepo := &mocks.MockCredentialRepository{}
	innerCredRepo.On("FindByIds", mock.Anything, []string{"missing"}).Return([]domain.Credential{}, nil)
	uow := mocks.NewPropagatingUnitOfWork()
	uow.On("Credential").Return(innerCredRepo)

	svc := &credentialService{
		uow:    uow,
		policy: &credentialPolicy{},
		logger: zap.NewNop(),
	}
	_, err := svc.Revoke(ctx, "missing")
	var domErr *domain.Error
	if assert.ErrorAs(t, err, &domErr) {
		assert.Equal(t, domain.CodeCredentialRevokeNotFound, domErr.Code)
	}
}

func TestRevoke_AlreadyRevoked(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	now := time.Now()
	innerCredRepo := &mocks.MockCredentialRepository{}
	innerCredRepo.On("FindByIds", mock.Anything, []string{"c1"}).Return(
		[]domain.Credential{{ID: "c1", RevokedAt: &now}}, nil)
	uow := mocks.NewPropagatingUnitOfWork()
	uow.On("Credential").Return(innerCredRepo)

	svc := &credentialService{
		uow:    uow,
		policy: &credentialPolicy{},
		logger: zap.NewNop(),
	}
	_, err := svc.Revoke(ctx, "c1")
	var domErr *domain.Error
	if assert.ErrorAs(t, err, &domErr) {
		assert.Equal(t, domain.CodeCredentialRevokeAlreadyRevoked, domErr.Code)
	}
}

func TestRevoke_ChainRollback(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	tokID := "1"
	innerCredRepo := &mocks.MockCredentialRepository{}
	innerCredRepo.On("FindByIds", mock.Anything, []string{"c1"}).Return(
		[]domain.Credential{{ID: "c1", TokenID: &tokID}}, nil)
	innerCredRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(
		[]domain.Credential{{ID: "c1"}}, nil)
	uow := mocks.NewPropagatingUnitOfWork()
	uow.On("Credential").Return(innerCredRepo)

	regSvc := &mocks.MockRegistryService{}
	regSvc.On("RevokeCredentials", mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError)

	svc := &credentialService{
		uow:             uow,
		registryService: regSvc,
		policy:          &credentialPolicy{},
		logger:          zap.NewNop(),
	}
	_, err := svc.Revoke(ctx, "c1")
	assert.Error(t, err)
}
```

- [ ] **Step 3: Add reExtractCompensate + syncBlockchainRevoke tests**

```go
func TestReExtractCompensate_Success(t *testing.T) {
	credRepo := &mocks.MockCredentialRepository{}
	credRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(
		[]domain.Credential{{ID: "c1"}}, nil)
	svc := &credentialService{repo: credRepo, logger: zap.NewNop()}
	err := svc.reExtractCompensate(context.Background(), domain.Credential{ID: "c1", ExtractError: lo.ToPtr("orig err")})
	assert.NoError(t, err)
}

func TestSyncBlockchainRevoke_EmptyInput(t *testing.T) {
	svc := &credentialService{logger: zap.NewNop()}
	err := svc.syncBlockchainRevoke(context.Background(), domain.Wallet{}, []string{})
	assert.NoError(t, err)
}

func TestSyncBlockchainRevoke_InvalidTokenID(t *testing.T) {
	svc := &credentialService{logger: zap.NewNop()}
	err := svc.syncBlockchainRevoke(context.Background(), domain.Wallet{}, []string{"not-a-number"})
	var domErr *domain.Error
	if assert.ErrorAs(t, err, &domErr) {
		assert.Equal(t, domain.CodeCredentialRevokeBlockchainSyncFailed, domErr.Code)
	}
}
```

- [ ] **Step 4: Run**

```bash
go test ./feature/credential/... -run "TestFind|TestRevoke|TestReExtractCompensate|TestSyncBlockchainRevoke" -v
```
Expected: all PASS.

- [ ] **Step 5: gofmt + commit**

```bash
gofmt -w feature/credential/credential_service_test.go
git add feature/credential/credential_service_test.go
git commit -m "test(credential): find not-found, revoke paths, compensate, syncBlockchainRevoke"
```

---

## Task 4: Handler helper unit tests (buildIssueItems, parseItemIndex, mapCredentialsToResponse)

**Files:**
- Create: `feature/credential/credential_handler_test.go`

Covers `credential_handler.go`: `buildIssueItems`, `parseItemIndex`, `mapCredentialsToResponse`, `readUploadedFile` (0% each).

- [ ] **Step 1: Write the tests**

```go
package credential

import (
	"mime/multipart"
	"testing"

	"CredChain_Golang/infrastructure/http/response"

	"github.com/stretchr/testify/assert"
)

func TestParseItemIndex(t *testing.T) {
	tests := []struct {
		key     string
		wantIdx int
		wantOK  bool
	}{
		{"items[0][holder_user_id]", 0, true},
		{"items[99][name]", 99, true},
		{"items[-1][x]", 0, false},
		{"items[abc][x]", 0, false},
		{"not_items[0][x]", 0, false},
		{"items[]", 0, false},
	}
	for _, tt := range tests {
		got, ok := parseItemIndex(tt.key)
		assert.Equal(t, tt.wantOK, ok, "key=%s", tt.key)
		assert.Equal(t, tt.wantIdx, got, "key=%s", tt.key)
	}
}

func TestMapCredentialsToResponse(t *testing.T) {
	creds := []domain.Credential{
		{ID: "c1", Name: "n1"},
		{ID: "c2", Name: "n2"},
	}
	out := mapCredentialsToResponse(creds)
	assert.Len(t, out, 2)
	assert.Equal(t, "c1", out[0].ID)
	assert.Equal(t, "n2", out[1].Name)
}

func TestMapCredentialsToResponse_Empty(t *testing.T) {
	out := mapCredentialsToResponse([]domain.Credential{})
	assert.Len(t, out, 0)
}
```

Note: `buildIssueItems` and `readUploadedFile` take multipart form data (`*multipart.Form`, `*multipart.FileHeader`). These are harder to unit-test without building real multipart data in the test. For coverage, the handler integration test via Postman covers them. Include them here only if the setup time is minimal. Otherwise skip — the `mapCredentialsToResponse` and `parseItemIndex` pure functions are good coverage wins.

- [ ] **Step 2: Run**

```bash
go test ./feature/credential/... -run "TestParseItemIndex|TestMapCredentialsToResponse" -v
```
Expected: all PASS.

- [ ] **Step 3: gofmt + commit**

```bash
gofmt -w feature/credential/credential_handler_test.go
git add feature/credential/credential_handler_test.go
git commit -m "test(credential): handler helper unit tests (parseItemIndex, mapCredentialsToResponse)"
```

---

## Task 5: GORM CredentialRepository tests via in-memory SQLite

**Files:**
- Create: `feature/credential/gorm_credential_repository_test.go`

Covers `gorm_credential_repository.go` methods that are testable via SQLite (currently 0%): `Store`, `FindByIds`, `FindByFileHashes`, `FindByHolderId`, `Find`, `Get` (paginated), `Update` (batch CASE).

Uses the existing in-memory SQLite harness: `infrastructure/testutil/db.OpenInMemorySQLite(t)`. This auto-migrates the GORM models (including `Credential`) into an in-memory SQLite database. Some Postgres-specific features (JSONB operations on `Meta`, ENUM types) round-trip as TEXT in SQLite — the tests exercise the GORM repository layer with those caveats.

- [ ] **Step 1: Write the test file**

```go
package credential

import (
	"context"
	"testing"
	"time"

	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/testutil/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openSQLite(t *testing.T) *gormCredentialRepository {
	t.Helper()
	gdb := db.OpenInMemorySQLite(t)
	return &gormCredentialRepository{db: gdb}
}

func TestGormCredentialStore_FindByIds(t *testing.T) {
	repo := openSQLite(t)
	ctx := context.Background()
	creds := []domain.Credential{
		{ID: "c1", HolderUserID: "h1", IssuerUserID: "iss", Name: "a", FileHash: "0xaa"},
		{ID: "c2", HolderUserID: "h2", IssuerUserID: "iss", Name: "b", FileHash: "0xbb"},
	}
	stored, err := repo.Store(ctx, creds...)
	require.NoError(t, err)
	assert.Len(t, stored, 2)

	found, err := repo.FindByIds(ctx, "c1", "c2")
	require.NoError(t, err)
	assert.Len(t, found, 2)

	partial, err := repo.FindByIds(ctx, "c1", "nonexistent")
	require.NoError(t, err)
	assert.Len(t, partial, 1)
	assert.Equal(t, "c1", partial[0].ID)
}

func TestGormCredentialFindByFileHashes(t *testing.T) {
	repo := openSQLite(t)
	ctx := context.Background()
	_, err := repo.Store(ctx, domain.Credential{ID: "c1", HolderUserID: "h1", IssuerUserID: "iss", Name: "a", FileHash: "0xaa"})
	require.NoError(t, err)

	found, err := repo.FindByFileHashes(ctx, "0xaa", "0xbb")
	require.NoError(t, err)
	assert.Len(t, found, 1)
	assert.Equal(t, "c1", found[0].ID)
}

func TestGormCredentialFindByHolderId(t *testing.T) {
	repo := openSQLite(t)
	ctx := context.Background()
	_, err := repo.Store(ctx,
		domain.Credential{ID: "c1", HolderUserID: "h1", IssuerUserID: "iss", Name: "a", FileHash: "0xaa"},
		domain.Credential{ID: "c2", HolderUserID: "h2", IssuerUserID: "iss", Name: "b", FileHash: "0xbb"},
	)
	require.NoError(t, err)

	held, err := repo.FindByHolderId(ctx, "h1")
	require.NoError(t, err)
	assert.Len(t, held, 1)
	assert.Equal(t, "c1", held[0].ID)
}

func TestGormCredentialFind(t *testing.T) {
	repo := openSQLite(t)
	ctx := context.Background()
	_, err := repo.Store(ctx, domain.Credential{ID: "c1", HolderUserID: "h1", IssuerUserID: "iss", Name: "a", FileHash: "0xaa"})
	require.NoError(t, err)

	found, err := repo.Find(ctx, "c1", nil)
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, "c1", found.ID)

	missing, err := repo.Find(ctx, "nonexistent", nil)
	assert.Nil(t, missing)
	assert.Error(t, err)
}

func TestGormCredentialUpdate(t *testing.T) {
	repo := openSQLite(t)
	ctx := context.Background()
	_, err := repo.Store(ctx, domain.Credential{ID: "c1", HolderUserID: "h1", IssuerUserID: "iss", Name: "a", FileHash: "0xaa"})
	require.NoError(t, err)

	now := time.Now()
	updated, err := repo.Update(ctx, domain.Credential{ID: "c1", Name: "updated", RevokedAt: &now})
	require.NoError(t, err)
	assert.Len(t, updated, 1)
	assert.Equal(t, "updated", updated[0].Name)
	assert.NotNil(t, updated[0].RevokedAt)
}

func TestGormCredentialGet(t *testing.T) {
	repo := openSQLite(t)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		id := fmt.Sprintf("c%d", i+1)
		_, err := repo.Store(ctx, domain.Credential{ID: id, HolderUserID: "h1", IssuerUserID: "iss", Name: id, FileHash: "0x" + id})
		require.NoError(t, err)
	}
	all, total, err := repo.Get(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, all, 5)
}
```

- [ ] **Step 2: Add `fmt` import** — the test uses `fmt.Sprintf`. Read the top of the test file — if `fmt` isn't imported, don't add it. Instead use simpler ID generation (e.g. `"c"+strconv.Itoa(i+1)`) to avoid a new import. Or add `"fmt"` to the import block — check which is cleaner. Prefer `strconv.Itoa` over `fmt.Sprintf` to avoid adding an import.

- [ ] **Step 3: Run**

```bash
go test ./feature/credential/... -run "TestGormCredential" -v
```
Expected: all PASS.

- [ ] **Step 4: gofmt + commit**

```bash
gofmt -w feature/credential/gorm_credential_repository_test.go
git add feature/credential/gorm_credential_repository_test.go
git commit -m "test(credential): GORM repository in-memory SQLite coverage"
```

---

## Task 6: River worker ErrorHandler + HandlePanic tests

**Files:**
- Modify: `infrastructure/jobs/credential_extract_river_test.go`

Covers `credential_extract_river.go`: `HandleError`, `HandlePanic`, `handleTerminalFailure` (currently not directly tested).

- [ ] **Step 1: Read the existing worker test file**

Check the current test structure and `rivertype.JobRow` fields: `Attempt`, `MaxAttempts`, `Kind`, `EncodedArgs`.

- [ ] **Step 2: Add terminal-failure tests**

```go
func TestHandleError_TerminalFailure(t *testing.T) {
	credRepo := &mocks.MockCredentialRepository{}
	credRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(
		[]domain.Credential{{ID: "cred-id"}}, nil)

	args, err := json.Marshal(CredentialExtractArgs{CredentialID: "cred-id", FileURI: "/tmp/x"})
	require.NoError(t, err)

	w := &CredentialExtractWorker{credRepo: credRepo, logger: zap.NewNop()}
	job := &rivertype.JobRow{
		Kind:         "credential_extract",
		Attempt:      5,
		MaxAttempts:  5,
		EncodedArgs:  args,
	}
	w.HandleError(context.Background(), job, assert.AnError)
	credRepo.AssertCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
}

func TestHandleError_NonTerminal(t *testing.T) {
	credRepo := &mocks.MockCredentialRepository{}
	args, err := json.Marshal(CredentialExtractArgs{CredentialID: "cred-id"})
	require.NoError(t, err)
	w := &CredentialExtractWorker{credRepo: credRepo, logger: zap.NewNop()}
	job := &rivertype.JobRow{
		Kind:        "credential_extract",
		Attempt:     1,
		MaxAttempts: 5,
		EncodedArgs: args,
	}
	w.HandleError(context.Background(), job, assert.AnError)
	credRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestHandlePanic_Terminal(t *testing.T) {
	credRepo := &mocks.MockCredentialRepository{}
	credRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(
		[]domain.Credential{{ID: "cred-id"}}, nil)
	args, err := json.Marshal(CredentialExtractArgs{CredentialID: "cred-id"})
	require.NoError(t, err)
	w := &CredentialExtractWorker{credRepo: credRepo, logger: zap.NewNop()}
	job := &rivertype.JobRow{
		Kind:        "credential_extract",
		Attempt:     5,
		MaxAttempts: 5,
		EncodedArgs: args,
	}
	w.HandlePanic(context.Background(), job, "ouch", "stacktrace")
	credRepo.AssertCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
}
```

- [ ] **Step 3: Add imports** — the test file needs `"encoding/json"`, `"github.com/riverqueue/river/rivertype"`, `"github.com/stretchr/testify/require"`. Check existing imports and add.

- [ ] **Step 4: Run**

```bash
go test ./infrastructure/jobs/... -run "TestHandleError|TestHandlePanic" -v
```
Expected: all PASS.

- [ ] **Step 5: gofmt + commit**

```bash
gofmt -w infrastructure/jobs/credential_extract_river_test.go
git add infrastructure/jobs/credential_extract_river_test.go
git commit -m "test(jobs): river worker HandleError/HandlePanic terminal-failure coverage"
```

---

## Task 7: Final coverage measurement + verify gate

- [ ] **Step 1: Measure credential feature coverage**

```bash
go test -cover ./feature/credential/... 2>&1
```

- [ ] **Step 2: Measure jobs coverage**

```bash
go test -cover ./infrastructure/jobs/... 2>&1
```

- [ ] **Step 3: Full gate**

```bash
go test ./... && go vet ./... && gofmt -l .
```
All must be clean.

- [ ] **Step 4: Report**

```bash
git log --oneline HEAD~10..HEAD
```
