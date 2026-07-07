# Structured Service-Level Validation Errors + All-or-Nothing Batch Operations

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Convert User Store and Credential Issue service methods to return Ozzo `validation.Errors` for input-driven business-validation failures, remove partial success from Issue (make it all-or-nothing), delete `CodeCredentialIssueFailed` (dead code), and delete `SendPartial` from the responder.

**Architecture:** Service methods return `validation.Errors` (type-asserted by handlers) for per-item field-level validation failures. Server-side failures (wallet gen, encryption, chain sync, policy) stay as `*domain.Error`. The handler uses the existing `responder.SendValidationError` pipeline — zero new types.

**Tech Stack:** Go 1.25.1, ozzo-validation v4, Gin v1.12

## Global Constraints

- Ozzo `validation.Errors` for input-driven failures only; `*domain.Error` for server-side failures
- Reuse existing `CodeSystemValidation` (100040) for all validation error responses
- All batch endpoints all-or-nothing — no partial success
- No new domain types, no changes to `domain/` package
- `gofmt -l .` must produce zero output
- All tests must pass: `go test ./...`

---

## File Structure

| File | Responsibility |
|------|---------------|
| `feature/user/user_service.go` | `Store`: collect `validation.Errors` from `storeValidateEmails`. `storeValidateEmails` returns `validation.Errors` instead of `error`. |
| `feature/user/user_handler.go` | `Store` handler: type-assert `validation.Errors` from service, dispatch to `SendValidationError`. |
| `feature/credential/credential_service.go` | `Issue`: new signature `([]domain.Credential, error)`. New `issueValidate` method returns `validation.Errors`. Remove `fieldErrs` return, per-item loop, `claimedHash`, `fileURIs` tracking. `cleanupOrphanFiles` stays for UoW-failure path. |
| `feature/credential/credential_handler.go` | `Issue` handler: remove `SendPartial`, add `validation.Errors` dispatch. Always `CodeCredentialIssueSuccess` on success. |
| `infrastructure/http/responder/responder.go` | Delete `SendPartial`. Keep `resolveMessage` (used by exported `ResolveMessage`). |
| `domain/codes.go` | Delete `CodeCredentialIssueFailed` (400240). |
| `infrastructure/http/responder/mapper.go` | Delete `CodeCredentialIssueFailed` from `CodeToMessageKey` and `HttpCodes`. |
| `infrastructure/http/responder/mapper_test.go` | Delete `CodeCredentialIssueFailed` from `allDomainCodes`. |
| `locales/en.json` | Delete `error_credential_issue_failed`. Add 4 new Ozzo validation keys. |
| `locales/id.json` | Delete `error_credential_issue_failed`. Add 4 new Ozzo validation keys. |
| `feature/credential/credential_service_test.go` | Update `TestIssue_AllFailed`, `TestIssue_PartialSuccess`, `TestIssue_ChainRollback` for new signature. Add validation error tests. |
| `feature/user/user_service_test.go` | Add tests for `Store` returning `validation.Errors`. |
| `CREDENTIAL.md` | Remove partial success paragraph. Remove `CodeCredentialIssueFailed` from error codes table. |
| `AGENTS.md` | Add service-level `validation.Errors` dispatch pattern. Remove `SendPartial` references. |

---

### Task 1: User Service — Return `validation.Errors` from `storeValidateEmails`

**Files:**
- Modify: `feature/user/user_service.go:70-117`

**Interfaces:**
- Produces: `storeValidateEmails(ctx, users) validation.Errors` (was `error`)
- Produces: `Store` now returns `validation.Errors` for email duplicates

- [ ] **Step 1: Rewrite `storeValidateEmails` to return `validation.Errors`**

Replace lines 83-117 of `feature/user/user_service.go`:

```go
func (s *userService) storeValidateEmails(ctx context.Context, users []domain.User) validation.Errors {
	verrs := validation.Errors{}

	seen := map[string]int{}
	for i, u := range users {
		if prev, ok := seen[u.Email]; ok {
			verrs[fmt.Sprintf("users.%d.email", i)] = validation.NewError(
				"validation_store_email_duplicate_batch",
				map[string]interface{}{"field": u.Email},
			)
			verrs[fmt.Sprintf("users.%d.email", prev)] = validation.NewError(
				"validation_store_email_duplicate_batch",
				map[string]interface{}{"field": u.Email},
			)
		}
		seen[u.Email] = i
	}

	if len(verrs) == 0 {
		emails := make([]string, len(users))
		for i, u := range users {
			emails[i] = u.Email
		}
		existing, _ := s.userRepo.FindByEmails(ctx, emails...)
		existingSet := lo.SliceToMap(existing, func(e domain.User) (string, bool) { return e.Email, true })
		for i, u := range users {
			if existingSet[u.Email] {
				verrs[fmt.Sprintf("users.%d.email", i)] = validation.NewError(
					"validation_store_email_duplicate_db",
					map[string]interface{}{"field": u.Email},
				)
			}
		}
	}

	return verrs
}
```

Add `"fmt"` and `validation "github.com/go-ozzo/ozzo-validation/v4"` to imports.

- [ ] **Step 2: Update `Store` orchestrator**

Replace lines 70-81:

```go
func (s *userService) Store(ctx context.Context, users ...domain.User) ([]domain.User, error) {
	if err := s.policy.Store(ctx, users...); err != nil {
		return nil, err
	}

	if verrs := s.storeValidateEmails(ctx, users); len(verrs) > 0 {
		return nil, verrs
	}

	if err := s.storeGenerateWallets(users); err != nil {
		return nil, err
	}

	return s.storeUsersAndSyncBlockchainRoles(ctx, users)
}
```

- [ ] **Step 3: Run tests**

```bash
cd CredChain_Golang && go test ./feature/user/... -v -run TestStore
```

- [ ] **Step 4: Commit**

```bash
git add feature/user/user_service.go
git commit -m "feat(user): return ozzo validation.Errors from storeValidateEmails for per-item email errors"
```

---

### Task 2: User Handler — Add `validation.Errors` dispatch in `Store`

**Files:**
- Modify: `feature/user/user_handler.go:93-116`

**Interfaces:**
- Consumes: `Store` returns `validation.Errors` on email validation failures

- [ ] **Step 1: Replace `Store` handler**

Replace lines 93-116 of `feature/user/user_handler.go`:

```go
func (h *userHandler) Store(c *gin.Context) {
	var req UserStoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	domainUsers := req.ToDomain()
	created, err := h.userSvc.Store(c.Request.Context(), domainUsers...)
	if err != nil {
		c.Error(err)
		if verrs, ok := err.(validation.Errors); ok {
			responder.SendValidationError(c, verrs)
			return
		}
		responder.SendError(c, err)
		return
	}
	users := make([]response.User, len(created))
	for i, u := range created {
		users[i] = response.FromDomainUser(u)
	}
	responder.Send(c, domain.CodeUserStoreSuccess, users)
}
```

Add `validation "github.com/go-ozzo/ozzo-validation/v4"` to imports.

- [ ] **Step 2: Run tests**

```bash
cd CredChain_Golang && go test ./feature/user/... -v
```

- [ ] **Step 3: Verify compilation**

```bash
cd CredChain_Golang && go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add feature/user/user_handler.go
git commit -m "feat(user): dispatch validation.Errors in Store handler"
```

---

### Task 3: Delete `SendPartial` from responder

**Files:**
- Modify: `infrastructure/http/responder/responder.go:238-261`

- [ ] **Step 1: Delete `SendPartial` function, keep `resolveMessage`**

Delete lines 238-250 (the `SendPartial` function). Lines 252-261 (`resolveMessage` and `ResolveMessage`) stay.

Result: file ends at line 236 (`func localize...}`) followed by `resolveMessage` + `ResolveMessage`.

- [ ] **Step 2: Verify compilation**

```bash
cd CredChain_Golang && go build ./...
```

Expected: compilation error because `credential_handler.go` still references `SendPartial`. This is expected — Task 5 will fix it.

- [ ] **Step 3: Commit**

```bash
git add infrastructure/http/responder/responder.go
git commit -m "refactor(responder): delete SendPartial, keep resolveMessage"
```

---

### Task 4: Credential Service — Rewrite `Issue` as all-or-nothing with `validation.Errors`

**Files:**
- Modify: `feature/credential/credential_service.go:51-58` (interface), `feature/credential/credential_service.go:189-382` (Issue method)

**Interfaces:**
- Consumes: `responder.SendPartial` deleted (Task 3)
- Produces: `Issue(ctx, items) ([]domain.Credential, error)` — no `map[string][]string` return
- Produces: new private method `issueValidate(ctx, items) validation.Errors`

The `cleanupOrphanFiles` helper stays — it handles the UoW-failure case where files were already persisted but the transaction rolled back.

- [ ] **Step 1: Update `CredentialService` interface signature**

Replace line 34 in the interface block (around line 44):

```go
Issue(ctx context.Context, items []CredentialIssuance) ([]domain.Credential, error)
```

- [ ] **Step 2: Add new `issueValidate` method**

Insert before `Issue` (after line 188). This method collects all input-driven validation errors:

```go
// issueValidate performs batch input-driven validation that belongs at the
// service layer: holder existence, on-chain duplicate file hash, and in-batch
// duplicate hash. Returns validation.Errors keyed by "credentials.N.field".
// Server-side failures (encryption, storage, chain mint) are NOT collected
// here — those remain *domain.Error from the Issue orchestrator.
func (s *credentialService) issueValidate(
	ctx context.Context,
	items []CredentialIssuance,
) validation.Errors {
	verrs := validation.Errors{}

	holderIDs := lo.Map(items, func(it CredentialIssuance, _ int) string { return it.HolderUserID })
	holders, _ := s.userRepo.FindByIds(ctx, holderIDs...)
	holderSet := lo.SliceToMap(holders, func(h domain.User) (string, bool) { return h.Id, true })

	hashes := make([]string, len(items))
	hashBytes := make([][32]byte, len(items))
	for i, it := range items {
		hash := ethCrypto.Keccak256(it.FileBytes)
		hashes[i] = "0x" + hex.EncodeToString(hash)
		copy(hashBytes[i][:], hash)
	}

	statuses, _ := s.registryService.GetCredentialHashStatuses(ctx, hashBytes)
	onChainActive := map[string]bool{}
	for i, st := range statuses {
		if st.Status == 1 {
			onChainActive[hashes[i]] = true
		}
	}

	seenHash := map[string]bool{}
	for i, it := range items {
		prefix := fmt.Sprintf("credentials.%d", i)

		if !holderSet[it.HolderUserID] {
			verrs[prefix+".holder_user_id"] = validation.NewError(
				"validation_issue_holder_not_found",
				map[string]interface{}{"field": it.HolderUserID},
			)
			continue
		}

		if onChainActive[hashes[i]] || seenHash[hashes[i]] {
			verrs[prefix+".file"] = validation.NewError(
				"validation_issue_duplicate_file_hash", nil,
			)
			continue
		}
		seenHash[hashes[i]] = true
	}

	return verrs
}
```

- [ ] **Step 3: Add `issuePrepareCredentials` helper**

Insert after `issueValidate` (before `Issue`). Encrypts files, persists to storage, and builds `[]domain.Credential` entities. Returns `*domain.Error` on encryption or storage failure:

```go
// issuePrepareCredentials encrypts files, persists them to storage, and builds
// domain.Credential entities with extract_status=pending. Returns *domain.Error
// on encryption or storage failure (caller cleans up orphan files).
func (s *credentialService) issuePrepareCredentials(
	ctx context.Context,
	items []CredentialIssuance,
) ([]domain.Credential, error) {
	authUser := httpContext.MustGetUser(ctx)

	creds := make([]domain.Credential, len(items))
	for i, it := range items {
		ext := strings.ToLower(filepath.Ext(it.Filename))
		if ext == "" {
			ext = ".bin"
		}
		encryptedHex, encErr := infraCrypto.Encrypt(it.FileBytes, []byte(*s.cfg.FileEncryptionKey))
		if encErr != nil {
			return nil, domain.NewError(domain.CodeCredentialIssueStorageFailed,
				domain.WithError(encErr))
		}
		filename := ulid.Make().String() + ext
		filePath := filepath.Join(*s.cfg.CredentialFileStoragePath, filename)
		if _, err := s.storage.SaveBytes([]byte(encryptedHex), filePath); err != nil {
			return nil, domain.NewError(domain.CodeCredentialIssueStorageFailed,
				domain.WithError(err))
		}
		hash := "0x" + hex.EncodeToString(ethCrypto.Keccak256(it.FileBytes))
		creds[i] = domain.Credential{
			ID:            ulid.Make().String(),
			HolderUserID:  it.HolderUserID,
			IssuerUserID:  authUser.Id,
			Name:          it.Name,
			Meta:          it.Meta,
			FileHash:      hash,
			FileURI:       &filename,
			ExtractStatus: domain.ExtractStatusPending,
		}
	}
	return creds, nil
}
```

- [ ] **Step 4: Add `issueCleanupOrphanFiles` helper**

Insert after `issuePrepareCredentials`. Builds full paths from credential `FileURI` fields and delegates to `cleanupOrphanFiles`:

```go
func (s *credentialService) issueCleanupOrphanFiles(creds []domain.Credential) {
	paths := make([]string, 0, len(creds))
	for _, c := range creds {
		if c.FileURI != nil {
			paths = append(paths, filepath.Join(*s.cfg.CredentialFileStoragePath, *c.FileURI))
		}
	}
	s.cleanupOrphanFiles(paths)
}
```

- [ ] **Step 5: Add `issueCommit` helper**

Insert after `issueCleanupOrphanFiles`. Runs the UoW: Store → chain mint → update token IDs → enqueue extract jobs:

```go
// issueCommit runs the UoW transaction: Store credentials, mint on-chain,
// update token IDs, and enqueue River extraction jobs. Chain failure or
// enqueue failure rolls back the entire transaction.
func (s *credentialService) issueCommit(
	ctx context.Context,
	authWallet domain.Wallet,
	creds []domain.Credential,
) ([]domain.Credential, error) {
	holderIDs := lo.Map(creds, func(c domain.Credential, _ int) string { return c.HolderUserID })
	holders, err := s.userRepo.FindByIds(ctx, holderIDs...)
	if err != nil {
		return nil, err
	}
	holderByID := lo.SliceToMap(holders, func(h domain.User) (string, domain.User) { return h.Id, h })

	if err := s.policy.IssuePostFetch(ctx, creds, holders); err != nil {
		return nil, err
	}

	var committed []domain.Credential
	err = s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		stored, err := uow.Credential().Store(ctx, creds...)
		if err != nil {
			return err
		}
		for _, c := range stored {
			if c.FileURI == nil {
				return domain.NewError(domain.CodeCredentialIssueStorageFailed,
					domain.WithMetadata("credential_id", c.ID))
			}
		}
		issuances := make([]chain.CredentialIssuance, len(stored))
		for i, c := range stored {
			issuances[i] = chain.CredentialIssuance{
				HolderAddress: holderByID[c.HolderUserID].WalletAddress,
				Hash:          c.FileHash,
				URI:           c.ID,
			}
		}
		tokenIds, err := s.syncBlockchainIssue(ctx, authWallet, issuances)
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
			fileURI := filepath.Join(*s.cfg.CredentialFileStoragePath, *c.FileURI)
			if err := s.issueEnqueueExtractJob(ctx, c.ID, fileURI); err != nil {
				return err
			}
		}
		committed = stored
		return nil
	})
	return committed, err
}
```

- [ ] **Step 6: Rewrite `Issue` orchestrator**

Replace lines 190-368 with the thin orchestrator that delegates to helpers. Mirrors `Store`'s structure:

```go
// Issue performs the synchronous batch credential issuance flow.
//
// Architecture (Option A — sync chain, async embeddings):
//  1. IssuePreFetch policy (signer is Issuer+)
//  2. issueValidate — input-driven checks (holders, duplicate hashes)
//  3. issuePrepareCredentials — encrypt, store, build entities
//  4. issueCommit — UoW: Store → chain mint → update token IDs → enqueue
// All-or-nothing: any validation failure returns validation.Errors;
// any server-side failure rolls back the entire batch.
func (s *credentialService) Issue(ctx context.Context, items []CredentialIssuance) ([]domain.Credential, error) {
	authUser := httpContext.MustGetUser(ctx)

	if err := s.policy.IssuePreFetch(ctx, lo.Map(items, func(it CredentialIssuance, _ int) domain.Credential {
		return domain.Credential{HolderUserID: it.HolderUserID}
	})); err != nil {
		return nil, err
	}

	if verrs := s.issueValidate(ctx, items); len(verrs) > 0 {
		return nil, verrs
	}

	creds, err := s.issuePrepareCredentials(ctx, items)
	if err != nil {
		s.issueCleanupOrphanFiles(creds)
		return nil, err
	}

	authWallet := domain.WalletFromUser(*authUser)
	committed, err := s.issueCommit(ctx, authWallet, creds)
	if err != nil {
		s.issueCleanupOrphanFiles(creds)
		return nil, err
	}

	return committed, nil
}
```

- [ ] **Step 8: Run tests**

```bash
cd CredChain_Golang && go test ./feature/credential/... -v -run TestIssue
```

Expected: compilation errors — tests still reference old signature. Will fix in Task 10.

- [ ] **Step 9: Commit**

```bash
git add feature/credential/credential_service.go
git commit -m "feat(credential): rewrite Issue as all-or-nothing, decompose into focused helpers"
```

---

### Task 5: Credential Handler — Remove `SendPartial`, add `validation.Errors` dispatch

**Files:**
- Modify: `feature/credential/credential_handler.go:192-266`

**Interfaces:**
- Consumes: `Issue` returns `([]domain.Credential, error)` (Task 4)
- Consumes: `SendPartial` deleted (Task 3)

- [ ] **Step 1: Replace `Issue` handler**

Replace lines 192-266 of `feature/credential/credential_handler.go`:

```go
func (h *credentialHandler) Issue(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}

	items, err := buildIssueItems(form)
	if err != nil {
		c.Error(err)
		responder.SendValidationError(c, err)
		return
	}

	req := CredentialIssueRequest{Credentials: items}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}

	serviceItems := make([]CredentialIssuance, len(items))
	for i, it := range items {
		fileBytes, mime, filename, err := readUploadedFile(it.File)
		if err != nil {
			c.Error(err)
			responder.SendError(c, err)
			return
		}
		if !allowedMIMETypes[mime] {
			verrs := validation.Errors{
				fmt.Sprintf("credentials.%d.file", i): validation.NewError("validation_file_type_invalid", ""),
			}
			responder.SendValidationError(c, verrs)
			return
		}
		if int64(len(fileBytes)) > maxFileBytes {
			verrs := validation.Errors{
				fmt.Sprintf("credentials.%d.file", i): validation.NewError("validation_file_max_size", ""),
			}
			responder.SendValidationError(c, verrs)
			return
		}
		serviceItems[i] = CredentialIssuance{
			HolderUserID: it.HolderUserID,
			Name:         it.Name,
			Meta:         it.Meta,
			Filename:     filename,
			MIMEType:     mime,
			FileBytes:    fileBytes,
		}
	}

	created, err := h.credSvc.Issue(c.Request.Context(), serviceItems)
	if err != nil {
		c.Error(err)
		if verrs, ok := err.(validation.Errors); ok {
			responder.SendValidationError(c, verrs)
			return
		}
		responder.SendError(c, err)
		return
	}
	out := mapCredentialsToResponse(created)
	responder.Send(c, domain.CodeCredentialIssueSuccess, out)
}
```

- [ ] **Step 2: Remove unused imports**

Remove `"encoding/json"`, `"fmt"`, `"io"`, `"mime"`, `"mime/multipart"`, `"path/filepath"`, `"strconv"` from imports — verify which are still needed by other handler methods (they all are). Actually, keep all imports since other handler methods use them too.

- [ ] **Step 3: Run tests**

```bash
cd CredChain_Golang && go test ./feature/credential/... -v -run TestIssue
```

Expected: handler tests may still reference old `SendPartial` path. Will fix in Task 11.

- [ ] **Step 4: Verify compilation**

```bash
cd CredChain_Golang && go build ./...
```

- [ ] **Step 5: Commit**

```bash
git add feature/credential/credential_handler.go
git commit -m "feat(credential): remove SendPartial, add validation.Errors dispatch in Issue handler"
```

---

### Task 6: Delete `CodeCredentialIssueFailed` (400240) from domain codes

**Files:**
- Modify: `domain/codes.go:103`
- Modify: `infrastructure/http/responder/mapper.go:99,226`
- Modify: `infrastructure/http/responder/mapper_test.go:96`
- Modify: `locales/en.json:102`
- Modify: `locales/id.json:102`

- [ ] **Step 1: Delete from `domain/codes.go`**

Delete line 103: `CodeCredentialIssueFailed               = 400240`

- [ ] **Step 2: Delete from `mapper.go` CodeToMessageKey**

Delete line 99: `domain.CodeCredentialIssueFailed:                "error_credential_issue_failed",`

- [ ] **Step 3: Delete from `mapper.go` HttpCodes**

Delete line 226: `domain.CodeCredentialIssueFailed:                http.StatusInternalServerError,`

- [ ] **Step 4: Delete from `mapper_test.go` allDomainCodes**

Delete line 96: `domain.CodeCredentialIssueFailed,`

- [ ] **Step 5: Delete from `locales/en.json`**

Delete line 102: `"error_credential_issue_failed": "Failed to issue credential.",`

- [ ] **Step 6: Delete from `locales/id.json`**

Delete line 102: `"error_credential_issue_failed": "Gagal menerbitkan kredensial.",`

- [ ] **Step 7: Run tests**

```bash
cd CredChain_Golang && go test ./domain/... ./infrastructure/http/responder/... -v
```

Expected: mapper_test.go `TestRegistry_EveryCodeHasMessageKey` and `TestRegistry_EveryCodeHasHttpStatus` pass, locale_keys_test passes.

- [ ] **Step 8: Commit**

```bash
git add domain/codes.go infrastructure/http/responder/mapper.go infrastructure/http/responder/mapper_test.go locales/en.json locales/id.json
git commit -m "refactor: delete CodeCredentialIssueFailed (400240) — dead code"
```

---

### Task 7: Add 4 new i18n keys to locale files

**Files:**
- Modify: `locales/en.json`
- Modify: `locales/id.json`

- [ ] **Step 1: Add to `locales/en.json`**

Add after the existing `validation_*` keys (around line 32):

```json
"validation_store_email_duplicate_batch": "Email \"{{.field}}\" appears multiple times in this batch.",
"validation_store_email_duplicate_db": "Email \"{{.field}}\" is already registered.",
"validation_issue_holder_not_found": "Holder user \"{{.field}}\" was not found.",
"validation_issue_duplicate_file_hash": "A credential with this file hash already exists.",
```

- [ ] **Step 2: Add to `locales/id.json`**

Add after the existing `validation_*` keys (around line 32):

```json
"validation_store_email_duplicate_batch": "Email \"{{.field}}\" muncul beberapa kali dalam batch ini.",
"validation_store_email_duplicate_db": "Email \"{{.field}}\" sudah terdaftar.",
"validation_issue_holder_not_found": "Pemegang \"{{.field}}\" tidak ditemukan.",
"validation_issue_duplicate_file_hash": "Kredensial dengan hash file ini sudah ada.",
```

- [ ] **Step 3: Run locale keys test**

```bash
cd CredChain_Golang && go test ./infrastructure/http/responder/... -v -run Locale
```

Expected: all locale tests pass (the new keys are Ozzo codes, not domain codes — they don't need entries in `CodeToMessageKey`).

- [ ] **Step 4: Commit**

```bash
git add locales/en.json locales/id.json
git commit -m "feat(i18n): add 4 new ozzo validation keys for service-layer errors"
```

---

### Task 8: Update `CREDENTIAL.md` — remove partial success docs

**Files:**
- Modify: `CREDENTIAL.md:244,389-391`

- [ ] **Step 1: Remove partial success paragraph**

Replace lines 243-244 in `CREDENTIAL.md` (the "Partial success" paragraph):

```
**Partial success:** Returns both `results` (committed credentials) and `fieldErrs` (per-item errors keyed `"credentials.N"`). Handler maps to `responder.SendPartial(c, code, out, fieldErrs)`.
```

with:

```
All-or-nothing: any input-driven validation failure returns `validation.Errors` with per-item field paths; any server-side failure rolls back the entire batch. Service returns `([]domain.Credential, error)`.
```

- [ ] **Step 2: Update `CodeCredentialIssueFailed` entry in error codes table**

Replace line 390:

```markdown
| 400240 | `CodeCredentialIssueFailed` | 200 | All credentials failed (partial success returns 200 with field errors) |
```

with (remove the row entirely):

```markdown
```

- [ ] **Step 3: Commit**

```bash
git add CREDENTIAL.md
git commit -m "docs: update CREDENTIAL.md for all-or-nothing Issue, remove CodeCredentialIssueFailed"
```

---

### Task 9: Update `AGENTS.md` — add `validation.Errors` dispatch pattern

**Files:**
- Modify: `AGENTS.md` (in `CredChain_Golang/`)

- [ ] **Step 1: Add service-level `validation.Errors` pattern near the Response Envelope section**

Find the section that describes `responder.SendValidationError`. After "Ozzo validation errors" line, add:

```
### Service-Level Validation Errors

Service methods may return `validation.Errors` (Ozzo's `map[string]error`) for input-driven
business-validation failures (duplicate emails, holder not found, duplicate file hash).
Handlers type-assert the error and dispatch to `responder.SendValidationError`:

```go
created, err := h.userSvc.Store(ctx, users...)
if err != nil {
    if verrs, ok := err.(validation.Errors); ok {
        responder.SendValidationError(c, verrs)
        return
    }
    c.Error(err)
    responder.SendError(c, err)
    return
}
```

Server-side failures (wallet generation, encryption, chain sync, policy violations)
stay as `*domain.Error` and go through `responder.SendError`.

`SendPartial` has been removed. All batch endpoints are all-or-nothing.
```

- [ ] **Step 2: Verify no `SendPartial` references remain in AGENTS.md**

```bash
grep -n "SendPartial\|partial.success" CredChain_Golang/AGENTS.md
```

Expected: zero output.

- [ ] **Step 3: Commit**

```bash
git add AGENTS.md
git commit -m "docs: add service-level validation.Errors dispatch pattern to AGENTS.md"
```

---

### Task 10: Update credential service tests

**Files:**
- Modify: `feature/credential/credential_service_test.go:686-829`

**Interfaces:**
- Consumes: `Issue` returns `([]domain.Credential, error)` (Task 4)

- [ ] **Step 1: Rewrite `TestIssue_AllFailed`**

Replace lines 686-725:

```go
func TestIssue_ValidationErrors(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)
	userRepo := &mocks.MockUserRepository{}

	userRepo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{}, nil)

	regSvc := &mocks.MockRegistryService{}
	regSvc.On("GetCredentialHashStatuses", mock.Anything, mock.Anything).Return(
		[]contracts.CredentialRegistryCredentialHashStatus{}, nil,
	)

	svc := &credentialService{
		cfg:             testConfig(),
		registryService: regSvc,
		policy:          &credentialPolicy{},
		userRepo:        userRepo,
		logger:          zap.NewNop(),
	}
	items := []CredentialIssuance{
		{HolderUserID: "holder-1", Name: "a", Filename: "x.pdf", FileBytes: []byte("x")},
		{HolderUserID: "holder-2", Name: "b", Filename: "x.pdf", FileBytes: []byte("y")},
	}
	results, err := svc.Issue(ctx, items)
	assert.Nil(t, results)
	assert.Error(t, err)
	verrs, ok := err.(validation.Errors)
	assert.True(t, ok, "expected validation.Errors, got %T", err)
	assert.Contains(t, verrs, "credentials.0.holder_user_id")
	assert.Contains(t, verrs, "credentials.1.holder_user_id")
}
```

- [ ] **Step 2: Rewrite `TestIssue_ChainRollback`**

Replace lines 727-770:

```go
func TestIssue_ChainRollback(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)
	enq := &localMockEnqueuer{}
	userRepo := &mocks.MockUserRepository{}
	stor := &storage.Storage{Config: &config.Config{StoragePath: lo.ToPtr(t.TempDir())}}

	userRepo.On("FindByIds", mock.Anything, mock.Anything).Return(
		[]domain.User{{Id: "holder-valid"}}, nil)

	regSvc := &mocks.MockRegistryService{}
	regSvc.On("GetCredentialHashStatuses", mock.Anything, mock.Anything).Return(
		[]contracts.CredentialRegistryCredentialHashStatus{}, nil,
	)
	regSvc.On("IssueCredentials", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, assert.AnError)

	uow := mocks.NewPropagatingUnitOfWork()
	innerCredRepo := &mocks.MockCredentialRepository{}
	innerCredRepo.On("Store", mock.Anything, mock.Anything).Return(
		[]domain.Credential{{ID: "stored-1", FileURI: lo.ToPtr("up/test.pdf")}}, nil)
	uow.On("Credential").Return(innerCredRepo)

	svc := &credentialService{
		uow:             uow,
		cfg:             testConfig(),
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
	_, err := svc.Issue(ctx, items)
	assert.Error(t, err)
	_, ok := err.(validation.Errors)
	assert.False(t, ok, "expected *domain.Error for chain rollback, not validation.Errors")
}
```

- [ ] **Step 3: Delete `TestIssue_PartialSuccess`**

Delete lines 772-829 entirely. Partial success no longer exists.

- [ ] **Step 4: Add test for validation error on duplicate hash**

Add after `TestIssue_ChainRollback`:

```go
func TestIssue_DuplicateFileHash(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)
	userRepo := &mocks.MockUserRepository{}

	userRepo.On("FindByIds", mock.Anything, mock.Anything).Return(
		[]domain.User{{Id: "holder-1"}, {Id: "holder-2"}}, nil)

	regSvc := &mocks.MockRegistryService{}
	regSvc.On("GetCredentialHashStatuses", mock.Anything, mock.Anything).Return(
		[]contracts.CredentialRegistryCredentialHashStatus{
			{Status: 1}, // Issued
			{Status: 0},
		}, nil,
	)

	svc := &credentialService{
		cfg:             testConfig(),
		registryService: regSvc,
		policy:          &credentialPolicy{},
		userRepo:        userRepo,
		logger:          zap.NewNop(),
	}
	items := []CredentialIssuance{
		{HolderUserID: "holder-1", Name: "a", Filename: "x.pdf", FileBytes: []byte("dup")},
		{HolderUserID: "holder-2", Name: "b", Filename: "x.pdf", FileBytes: []byte("unique")},
	}
	results, err := svc.Issue(ctx, items)
	assert.Nil(t, results)
	assert.Error(t, err)
	verrs, ok := err.(validation.Errors)
	assert.True(t, ok, "expected validation.Errors for duplicate hash, got %T", err)
	assert.Contains(t, verrs, "credentials.0.file")
	assert.NotContains(t, verrs, "credentials.1.file")
}
```

- [ ] **Step 5: Run tests**

```bash
cd CredChain_Golang && go test ./feature/credential/... -v -run TestIssue
```

- [ ] **Step 6: Commit**

```bash
git add feature/credential/credential_service_test.go
git commit -m "test(credential): update Issue tests for all-or-nothing + validation.Errors"
```

---

### Task 11: Update credential handler tests

**Files:**
- Modify: `feature/credential/credential_handler_test.go`

- [ ] **Step 1: Find any Issue handler tests referencing `SendPartial` or `CodeCredentialIssueFailed`**

```bash
grep -n "SendPartial\|CodeCredentialIssueFailed\|fieldErrs" CredChain_Golang/feature/credential/credential_handler_test.go
```

- [ ] **Step 2: Update tests**

If tests reference the old `SendPartial` pattern or `fieldErrs`, rewrite them to expect `SendValidationError` for validation errors and `Send` with `CodeCredentialIssueSuccess` for success.

- [ ] **Step 3: Run tests**

```bash
cd CredChain_Golang && go test ./feature/credential/... -v
```

- [ ] **Step 4: Commit**

```bash
git add feature/credential/credential_handler_test.go
git commit -m "test(credential): update handler tests for all-or-nothing Issue"
```

---

### Task 12: Update user service tests

**Files:**
- Modify: `feature/user/user_service_test.go`

- [ ] **Step 1: Find existing `Store` tests**

```bash
grep -n "TestStore\|testStore\|func Test.*Store" CredChain_Golang/feature/user/user_service_test.go
```

- [ ] **Step 2: Add tests for `validation.Errors` path**

Add a test that verifies duplicate emails in batch return `validation.Errors`:

```go
func TestStore_DuplicateEmailInBatchReturnsValidationErrors(t *testing.T) {
	userRepo := &mocks.MockUserRepository{}
	userRepo.On("FindByEmails", mock.Anything, mock.Anything).Return([]domain.User{}, nil)

	svc := &userService{
		userRepo: userRepo,
		policy:   &userPolicy{},
		logger:   zap.NewNop(),
	}

	users := []domain.User{
		{Email: "dup@test.com", Role: domain.RoleHolder},
		{Email: "dup@test.com", Role: domain.RoleHolder},
	}

	_, err := svc.Store(context.Background(), users...)
	assert.Error(t, err)
	verrs, ok := err.(validation.Errors)
	assert.True(t, ok, "expected validation.Errors, got %T", err)
	assert.Contains(t, verrs, "users.0.email")
	assert.Contains(t, verrs, "users.1.email")
}
```

- [ ] **Step 3: Run tests**

```bash
cd CredChain_Golang && go test ./feature/user/... -v -run TestStore
```

- [ ] **Step 4: Commit**

```bash
git add feature/user/user_service_test.go
git commit -m "test(user): add validation.Errors test for Store duplicate emails"
```

---

### Task 13: Full verification

- [ ] **Step 1: Run all tests**

```bash
cd CredChain_Golang && go test ./... -v
```

- [ ] **Step 2: Run vet**

```bash
cd CredChain_Golang && go vet ./...
```

- [ ] **Step 3: Run format check**

```bash
cd CredChain_Golang && gofmt -l .
```

Expected: zero output.

- [ ] **Step 4: Commit if any fixes were needed**

```bash
git add -A && git commit -m "chore: fix tests and vet after validation.Errors refactor"
```

---

### Task 14: Run mock service (if applicable)

- [ ] **Step 1: Check if mock_credential_service_test.go needs update**

```bash
grep -n "Issue.*map\[string\]\[\]string" CredChain_Golang/feature/credential/mock_credential_service_test.go
```

If the mock's `Issue` signature changed, update it to `(context.Context, []CredentialIssuance) ([]domain.Credential, error)`.

- [ ] **Step 2: Run tests again**

```bash
cd CredChain_Golang && go test ./... -v
```

- [ ] **Step 3: Final commit**

```bash
git add -A && git commit -m "chore: update mock service for Issue signature change"
```
