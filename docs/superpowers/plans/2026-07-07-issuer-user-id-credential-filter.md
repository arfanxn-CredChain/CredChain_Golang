# Add `issuer_user_id` to Credential Filter Allowlist

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enable the frontend to filter credentials by `issuer_user_id` so the User Detail page can show credentials an Issuer+ user issued, not just credentials they hold.

**Architecture:** Add `issuer_user_id` to the `allowedFilterColumns` allowlist in the GORM credential repository, then add a test function mirroring the existing `holder_user_id` filter test. The shared `ApplyFilters` helper already handles all 14 operators generically — only the allowlist gating prevents `issuer_user_id` from being used.

**Tech Stack:** Go 1.25.1, GORM, testify, in-memory SQLite

## Global Constraints

- Go module path is `CredChain_Golang` (with underscore)
- Repository tests run against in-memory SQLite via `github.com/glebarez/sqlite`
- `go test ./... && go vet ./... && gofmt -l .` must pass before push
- Commit directly to master — no feature/bugfix branches

---

### Task 1: Add `issuer_user_id` to allowed filter columns

**Files:**
- Modify: `feature/credential/gorm_credential_repository.go:35-44`

**Interfaces:**
- Consumes: nothing new
- Produces: `allowedFilterColumns` map now includes `"issuer_user_id": true`

- [ ] **Step 1: Add the column to the allowlist map**

In `feature/credential/gorm_credential_repository.go`, replace lines 35-44:

```go
// allowedFilterColumns whitelists credential columns clients may filter on.
// holder_user_id and issuer_user_id are intentionally included so the
// user-detail UI can scope credentials to a specific holder or issuer.
var allowedFilterColumns = map[string]bool{
	"name":           true,
	"issued_at":      true,
	"revoked_at":     true,
	"holder_user_id": true,
	"issuer_user_id": true,
	"extract_status": true,
}
```

The only changes from the original are:
- Updated comment on line 35-36 to mention both columns
- Added `"issuer_user_id": true,` before `"extract_status": true,`

- [ ] **Step 2: Verify formatting**

```bash
cd CredChain_Golang && gofmt -l . | grep gorm_credential_repository.go
```

Expected: zero output (no formatting issues).

- [ ] **Step 3: Verify compilation**

```bash
cd CredChain_Golang && go vet ./feature/credential/
```

Expected: zero output.

- [ ] **Step 4: Commit**

```bash
cd CredChain_Golang && git add feature/credential/gorm_credential_repository.go
git commit -m "feat(credential): add issuer_user_id to filter allowlist"
```

---

### Task 2: Add `issuer_user_id` filter test

**Files:**
- Modify: `feature/credential/gorm_credential_repository_test.go` (append new test function)

**Interfaces:**
- Consumes: `allowedFilterColumns` from Task 1 (now includes `issuer_user_id`)
- Produces: `TestGormCredentialGet_FilterByIssuerUserId` — three sub-tests: equal, in, not_in

- [ ] **Step 1: Write the test**

Append the following function at the end of `feature/credential/gorm_credential_repository_test.go`:

```go
func TestGormCredentialGet_FilterByIssuerUserId(t *testing.T) {
	repo := openCredRepo(t)
	ctx := context.Background()

	_, err := repo.Store(ctx,
		domain.Credential{ID: "c1", HolderUserID: "h1", IssuerUserID: "i1", Name: "A", FileHash: "0xa"},
		domain.Credential{ID: "c2", HolderUserID: "h2", IssuerUserID: "i2", Name: "B", FileHash: "0xb"},
		domain.Credential{ID: "c3", HolderUserID: "h3", IssuerUserID: "i3", Name: "C", FileHash: "0xc"},
	)
	require.NoError(t, err)

	t.Run("equal", func(t *testing.T) {
		q := &domainQuery.Query{
			Filters: []domainQuery.Filter{
				{Column: "issuer_user_id", Operator: domainQuery.OperatorEqual, Values: []string{"i2"}},
			},
		}
		results, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Equal(t, "B", results[0].Name)
	})

	t.Run("in", func(t *testing.T) {
		q := &domainQuery.Query{
			Filters: []domainQuery.Filter{
				{Column: "issuer_user_id", Operator: domainQuery.OperatorIn, Values: []string{"i1", "i3"}},
			},
		}
		_, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 2, total)
	})

	t.Run("not_in", func(t *testing.T) {
		q := &domainQuery.Query{
			Filters: []domainQuery.Filter{
				{Column: "issuer_user_id", Operator: domainQuery.OperatorNotIn, Values: []string{"i1", "i3"}},
			},
		}
		results, total, err := repo.Get(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Equal(t, "B", results[0].Name)
	})
}
```

The test mirrors `TestGormCredentialGet_FilterByHolderUserId` (line 617) exactly, substituting `issuer_user_id` for `holder_user_id` and using distinct issuer IDs (`i1`, `i2`, `i3`) separate from holder IDs (`h1`, `h2`, `h3`) to confirm the filter operates on the correct column.

- [ ] **Step 2: Run the new test (expect failure because allowlist not yet applied)**

If Task 1 has not been committed yet, the test will fail silently — the `issuer_user_id` filter is dropped by `ApplyFilters`, returning all 3 credentials instead of the expected 1.

```bash
cd CredChain_Golang && go test ./feature/credential/ -v -run TestGormCredentialGet_FilterByIssuerUserId
```

Expected output (without Task 1):
```
=== RUN   TestGormCredentialGet_FilterByIssuerUserId/equal
    gorm_credential_repository_test.go:NNN:
                Error Trace:    .../gorm_credential_repository_test.go:NNN
                Error:          Not equal: expected: 1, actual: 3
--- FAIL: TestGormCredentialGet_FilterByIssuerUserId/equal
```

- [ ] **Step 3: Run the test again (with Task 1 applied)**

```bash
cd CredChain_Golang && go test ./feature/credential/ -v -run TestGormCredentialGet_FilterByIssuerUserId
```

Expected:
```
=== RUN   TestGormCredentialGet_FilterByIssuerUserId
=== RUN   TestGormCredentialGet_FilterByIssuerUserId/equal
=== RUN   TestGormCredentialGet_FilterByIssuerUserId/in
=== RUN   TestGormCredentialGet_FilterByIssuerUserId/not_in
--- PASS: TestGormCredentialGet_FilterByIssuerUserId
PASS
```

- [ ] **Step 4: Run the existing `holder_user_id` test to confirm no regression**

```bash
cd CredChain_Golang && go test ./feature/credential/ -v -run TestGormCredentialGet_FilterByHolderUserId
```

Expected:
```
=== RUN   TestGormCredentialGet_FilterByHolderUserId
=== RUN   TestGormCredentialGet_FilterByHolderUserId/equal
=== RUN   TestGormCredentialGet_FilterByHolderUserId/in
=== RUN   TestGormCredentialGet_FilterByHolderUserId/not_in
--- PASS: TestGormCredentialGet_FilterByHolderUserId
PASS
```

- [ ] **Step 5: Run full test suite**

```bash
cd CredChain_Golang && go test ./... && go vet ./... && gofmt -l .
```

Expected: all tests pass, `go vet` produces zero output, `gofmt -l .` produces zero output.

- [ ] **Step 6: Commit**

```bash
cd CredChain_Golang && git add feature/credential/gorm_credential_repository_test.go
git commit -m "test(credential): add issuer_user_id filter test"
```
