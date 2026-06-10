# Credential Verify Party-Disabled + Delete UoW + Restore Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add party-disabled verdicts to /verify (holder/issuer deleted), add *Query param to 3 CredentialRepository methods, unify Delete into single UoW, add PUT /api/users/batch/restore endpoint, remove B6 from plan doc.

**Architecture:** Verdict codes follow existing 400401-400409 block (new: 400410-400412). Restore codes get dedicated 3009xx block. CredRepo interface change ripples to repo impl + mocks + 6 callers. Party-disabled check uses inline holder/issuer preloads in Verify's 3 resolution branches.

**Tech Stack:** Go 1.25.1, Gin, GORM, testify, Ozzo validation, go-i18n

**Spec:** `docs/superpowers/specs/2026-06-10-credential-verify-party-disabled-design.md`

**Verification:** `go test ./... && go vet ./... && gofmt -l .`

---

### Task 1: Add domain codes (9 new constants)

**Files:**
- Modify: `domain/codes.go`

- [ ] **Step 1: Add 3009xx restore block**

Insert after `CodeUserTransferSuperAdminBlockchainSyncFailed = 300645` (line 78), before the `// ── Credential (40)` comment:

```go
	// ── Restore (09) ─────────────────────────────────────────────────────────
	CodeUserRestoreSuccess                       = 300900
	CodeUserRestoreSignerAdminRequiredForbidden  = 300941
	CodeUserRestoreSelfTargetForbidden           = 300942
	CodeUserRestoreSuperAdminTargetForbidden     = 300943
	CodeUserRestoreNotTrashedForbidden           = 300944
	CodeUserRestoreBlockchainSyncFailed          = 300945
```

- [ ] **Step 2: Add 3 verify verdict codes**

Insert after `CodeCredentialVerifyNoMatch = 400409` (line 120):

```go
	CodeCredentialVerifyHolderDisabled = 400410
	CodeCredentialVerifyIssuerDisabled = 400411
	CodeCredentialVerifyPartyDisabled  = 400412
```

- [ ] **Step 3: Build check**

```bash
cd CredChain_Golang && go build ./...
```

Expected: exit 0.

- [ ] **Step 4: Commit**

```bash
git add domain/codes.go
git commit -m "feat(domain): add restore (3009xx) and party-disabled verify (400410-400412) codes"
```

---

### Task 2: Update CredentialRepository interface

**Files:**
- Modify: `domain/credential.go:72-80`

- [ ] **Step 1: Update 3 method signatures**

Replace lines 72-80 in `domain/credential.go`:

Old:
```go
	// FindByIds retrieves credentials by ID list (batch lookup).
	FindByIds(ctx context.Context, ids ...string) ([]Credential, error)

	// FindByHolderId retrieves all credentials owned by a given holder.
	FindByHolderId(ctx context.Context, holderID string) ([]Credential, error)

	// FindByFileHashes retrieves credentials whose file_hash matches any of
	// the given hashes. Used during issue to detect duplicate uploads.
	FindByFileHashes(ctx context.Context, hashes ...string) ([]Credential, error)
```

New (query is the LAST parameter, slices are NOT variadic):
```go
	// FindByIds retrieves credentials by ID list (batch lookup). When query is
	// non-nil it may carry Includes for preloading holder/issuer/revoker relations.
	FindByIds(ctx context.Context, ids []string, query *domainQuery.Query) ([]Credential, error)

	// FindByHolderId retrieves all credentials owned by a given holder. When
	// query is non-nil it may carry Includes for preloading relations.
	FindByHolderId(ctx context.Context, holderID string, query *domainQuery.Query) ([]Credential, error)

	// FindByFileHashes retrieves credentials whose file_hash matches any of
	// the given hashes. Used during issue to detect duplicate uploads. When
	// query is non-nil it may carry Includes for preloading relations.
	FindByFileHashes(ctx context.Context, hashes []string, query *domainQuery.Query) ([]Credential, error)
```

- [ ] **Step 2: Build check — expect errors from callers not yet updated**

```bash
cd CredChain_Golang && go build ./... 2>&1 | head -30
```

Expected: compilation errors in `credential_service.go`, `gorm_credential_repository.go`, mocks (signatures changed). This is expected — subsequent tasks fix them.

- [ ] **Step 3: Commit**

```bash
git add domain/credential.go
git commit -m "refactor(domain): add *Query param to CredRepo FindByIds/FindByHolderId/FindByFileHashes"
```

---

### Task 3: Update repo impl to accept *Query

**Files:**
- Modify: `feature/credential/gorm_credential_repository.go` (3 methods: FindByIds at ~240, FindByHolderId at ~256, FindByFileHashes at ~269)

- [ ] **Step 1: Update FindByIds**

Replace the method signature and body to accept `[]string` + `*Query`:

```go
func (r *gormCredentialRepository) FindByIds(ctx context.Context, ids []string, query *domainQuery.Query) ([]domain.Credential, error) {
	if len(ids) == 0 {
		return []domain.Credential{}, nil
	}
	db := r.db.WithContext(ctx)
	if query != nil {
		db = preloadByIncludes(db, query)
	}
	var rows []model.Credential
	if err := db.Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Credential, len(rows))
	for i, c := range rows {
		out[i] = c.ToDomain()
	}
	return out, nil
}
```

- [ ] **Step 2: Update FindByHolderId**

```go
func (r *gormCredentialRepository) FindByHolderId(ctx context.Context, holderID string, query *domainQuery.Query) ([]domain.Credential, error) {
	db := r.db.WithContext(ctx)
	if query != nil {
		db = preloadByIncludes(db, query)
	}
	var rows []model.Credential
	if err := db.Where("holder_user_id = ?", holderID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Credential, len(rows))
	for i, c := range rows {
		out[i] = c.ToDomain()
	}
	return out, nil
}
```

- [ ] **Step 3: Update FindByFileHashes**

```go
func (r *gormCredentialRepository) FindByFileHashes(ctx context.Context, hashes []string, query *domainQuery.Query) ([]domain.Credential, error) {
	if len(hashes) == 0 {
		return []domain.Credential{}, nil
	}
	db := r.db.WithContext(ctx)
	if query != nil {
		db = preloadByIncludes(db, query)
	}
	var rows []model.Credential
	if err := db.Where("file_hash IN ?", hashes).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Credential, len(rows))
	for i, c := range rows {
		out[i] = c.ToDomain()
	}
	return out, nil
}
```

- [ ] **Step 4: Update FindByIds call from ReExtract**

In `credential_service.go`, ReExtract method — find the `s.repo.FindByIds(ctx, ids...)` call and change to `s.repo.FindByIds(ctx, ids, nil)`.

- [ ] **Step 5: Update FindByFileHashes call from Issue**

In `credential_service.go`, Issue method — find `s.repo.FindByFileHashes(ctx, hashes...)` (line ~174) and change to `s.repo.FindByFileHashes(ctx, hashes, nil)`.

- [ ] **Step 6: Update FindByIds call from Revoke**

In `credential_service.go`, Revoke method — find `s.repo.FindByIds(ctx, credIDs...)` and change to `s.repo.FindByIds(ctx, credIDs, nil)`.

- [ ] **Step 7: Build check**

```bash
cd CredChain_Golang && go build ./feature/credential/... 2>&1
```

Expected: credential package builds. Mocks still broken (next task).

- [ ] **Step 8: Commit**

```bash
git add feature/credential/gorm_credential_repository.go feature/credential/credential_service.go
git commit -m "refactor(credential): update repo + callers for *Query param on FindByIds/FindByHolderId/FindByFileHashes"
```

---

### Task 4: Update mock CredentialRepository

**Files:**
- Modify: `infrastructure/testutil/mocks/credential_repository.go`

- [ ] **Step 1: Update 3 mock method signatures**

Find the `FindByIds`, `FindByHolderId`, `FindByFileHashes` method implementations in the mock and update signatures to match the new interface. Each method signature changes from variadic strings to `ids []string, query *domainQuery.Query` (or equivalent).

- [ ] **Step 2: Update credential service tests that call these mocks**

In `feature/credential/credential_service_test.go`, find any `.On("FindByIds"...`, `.On("FindByHolderId"...`, `.On("FindByFileHashes"...` calls. Add `nil` as the last mock argument (the query param). Also add `mock.Anything` for the query where flexibility is intentionally tested.

- [ ] **Step 3: Build check**

```bash
cd CredChain_Golang && go build ./...
```

Expected: exit 0.

- [ ] **Step 4: Run credential tests**

```bash
cd CredChain_Golang && go test ./feature/credential/...
```

Expected: PASS (existing tests still pass with nil query).

- [ ] **Step 5: Commit**

```bash
git add infrastructure/testutil/mocks/credential_repository.go feature/credential/credential_service_test.go
git commit -m "test(mocks): update MockCredentialRepository for *Query param"
```

---

### Task 5: Add party-disabled check to Verify

**Files:**
- Modify: `feature/credential/credential_service.go:433-503`

- [ ] **Step 1: Build verifyQuery at top of Verify method**

After `uploadedHash := ...` (line 438), add:

```go
	verifyQuery := &domainQuery.Query{Includes: []string{"holder", "issuer"}}
```

- [ ] **Step 2: Cache hit branch — pass verifyQuery to Find, add party check**

Replace lines 445-450:
```go
	if cached != nil {
		var cred *domain.Credential
		if cached.MatchedCredentialID != nil {
			cred, _ = s.repo.Find(ctx, *cached.MatchedCredentialID, verifyQuery)
		}
		code := cached.VerdictCode
		if code == domain.CodeCredentialVerifyAuthentic && cred != nil {
			holderGone := cred.Holder == nil || cred.Holder.DeletedAt != nil
			issuerGone := cred.Issuer == nil || cred.Issuer.DeletedAt != nil
			if holderGone && issuerGone {
				code = domain.CodeCredentialVerifyPartyDisabled
			} else if holderGone {
				code = domain.CodeCredentialVerifyHolderDisabled
			} else if issuerGone {
				code = domain.CodeCredentialVerifyIssuerDisabled
			}
		}
		return code, cred, cached.SimilarityScore, cached.SimilarityPercent, nil
	}
```

- [ ] **Step 3: Exact-hash branch — pass verifyQuery to FindByFileHashes, add party check**

Replace line 454: `existing, err := s.repo.FindByFileHashes(ctx, uploadedHash)` → `existing, err := s.repo.FindByFileHashes(ctx, []string{uploadedHash}, verifyQuery)`

After the existing verdict computation (lines 460-466), before `s.verifyCacheVerdict(...)` on line 467, add:
```go
		if code == domain.CodeCredentialVerifyAuthentic {
			holderGone := cred.Holder == nil || cred.Holder.DeletedAt != nil
			issuerGone := cred.Issuer == nil || cred.Issuer.DeletedAt != nil
			if holderGone && issuerGone {
				code = domain.CodeCredentialVerifyPartyDisabled
			} else if holderGone {
				code = domain.CodeCredentialVerifyHolderDisabled
			} else if issuerGone {
				code = domain.CodeCredentialVerifyIssuerDisabled
			}
		}
```

- [ ] **Step 4: Fuzzy branch — pass verifyQuery to Find, add party check**

Replace `s.repo.Find(ctx, best.CredentialID, nil)` on line 500 → `s.repo.Find(ctx, best.CredentialID, verifyQuery)`

After `code := s.verifyVerdictToCode(result.Verdict)` on line 499, add the same party-disabled check (identical code block to Step 2).

- [ ] **Step 5: Build + run credential tests**

```bash
cd CredChain_Golang && go test ./feature/credential/...
```

Expected: existing tests PASS (party checks are additive, not breaking).

- [ ] **Step 6: Commit**

```bash
git add feature/credential/credential_service.go
git commit -m "feat(credential): add party-disabled verdicts to Verify (holder/issuer deleted check)"
```

---

### Task 6: Verify party-disabled service tests

**Files:**
- Modify: `feature/credential/credential_service_test.go`

- [ ] **Step 1: Add test — HolderDisabled overrides Authentic**

```go
func TestVerify_HolderDisabled_OverridesAuthentic(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	delTime := time.Now().Add(-1 * time.Hour)
	holder := fixtures.NewDomainUser(
		fixtures.WithID("holder-1"),
		fixtures.WithRole(domain.RoleHolder),
	)
	holder.DeletedAt = &delTime
	issuer := fixtures.NewDomainUser(
		fixtures.WithID("issuer-1"),
		fixtures.WithRole(domain.RoleIssuer),
	)

	cred := domain.Credential{
		ID:           "c1",
		HolderUserID: "holder-1",
		IssuerUserID: "issuer-1",
		Holder:       &holder,
		Issuer:       &issuer,
	}
	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}
	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return((*domain.CredentialVerification)(nil), nil)
	m.regSvc.On("FindCredentialByHash", mock.Anything, mock.Anything).Return((*domain.Credential)(nil), false, nil)
	m.credRepo.On("FindByFileHashes", mock.Anything, []string{mock.Anything}, mock.AnythingOfType("*domainQuery.Query")).
		Return([]domain.Credential{cred}, nil)

	svc := newTestCredentialService(m)
	code, _, _, _, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test")})
	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyHolderDisabled, code)
}
```

- [ ] **Step 2: Add test — IssuerDisabled overrides Authentic**

Similar to Step 1 but issuer.DeletedAt set, holder live. Expect `CodeCredentialVerifyIssuerDisabled`.

- [ ] **Step 3: Add test — PartyDisabled (both deleted)**

Both holder and issuer have DeletedAt set. Expect `CodeCredentialVerifyPartyDisabled`.

- [ ] **Step 4: Add test — Does NOT override Revoked**

Credential has RevokedAt != nil AND holder is deleted. Expect `CodeCredentialVerifyRevoked` (not a party-disabled code).

- [ ] **Step 5: Add test — Missing holder (nil Holder) treated as disabled**

Credential returned by repo has no `Holder` preloaded (query with no includes — but our test uses verifyQuery, which does). For this test, mock `FindByFileHashes` to return a cred with `Holder: nil`. Expect `CodeCredentialVerifyHolderDisabled`.

- [ ] **Step 6: Add test — Does NOT override Tampered**

AI Verify returns `Tampered`, issuer deleted. Expect `CodeCredentialVerifyTampered` (the fuzzy verdict persists).

- [ ] **Step 7: Run credential tests**

```bash
cd CredChain_Golang && go test ./feature/credential/... -v -run "TestVerify_Party|TestVerify_Holder|TestVerify_Issuer"
```

Expected: 6 new tests PASS.

- [ ] **Step 8: Commit**

```bash
git add feature/credential/credential_service_test.go
git commit -m "test(credential): party-disabled verdict tests for Verify"
```

---

### Task 7: Delete UoW unification

**Files:**
- Modify: `feature/user/user_service.go` (Delete method ~376-434)

- [ ] **Step 1: Unify Delete into single s.uow.Execute**

Replace the `deleteUserAndSyncBlockchain` helper function AND the `Delete` method with:

```go
func (s *userService) Delete(ctx context.Context, ids ...string) (int64, error) {
	if err := s.policy.DeletePreFetch(ctx, ids...); err != nil {
		return 0, err
	}
	var rowsAffected int64
	err := s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		targetUsers, err := uow.User().FindByIds(ctx, ids...)
		if err != nil {
			return err
		}
		if err := s.policy.DeletePostFetch(ctx, targetUsers); err != nil {
			return err
		}
		if len(targetUsers) == 0 {
			return nil
		}
		var err2 error
		rowsAffected, err2 = uow.User().Delete(ctx, ids...)
		if err2 != nil {
			return err2
		}
		liveTargets := make([]domain.User, 0, len(targetUsers))
		for _, t := range targetUsers {
			if t.DeletedAt == nil {
				liveTargets = append(liveTargets, t)
			}
		}
		if len(liveTargets) == 0 {
			return nil
		}
		revocationUsers := make([]domain.User, len(liveTargets))
		for i, t := range liveTargets {
			revocationUsers[i] = domain.User{
				WalletAddress:             t.WalletAddress,
				EncryptedWalletPrivateKey: t.EncryptedWalletPrivateKey,
				Role:                      domain.RoleNone,
			}
		}
		return s.syncBlockchainRoles(ctx, revocationUsers, domain.CodeUserDeleteBlockchainSyncFailed)
	})
	return rowsAffected, err
}
```

Remove the `deleteUserAndSyncBlockchain` helper entirely (lines ~376-413).

- [ ] **Step 2: Run user tests**

```bash
cd CredChain_Golang && go test ./feature/user/... -v -run "TestUserService_Delete"
```

Expected: existing Delete tests PASS (PropagatingUnitOfWork wraps the single TX correctly).

- [ ] **Step 3: Commit**

```bash
git add feature/user/user_service.go
git commit -m "refactor(user): unify Delete into single UoW (fetch + delete + chain sync in one TX)"
```

---

### Task 8: Restore policy

**Files:**
- Modify: `feature/user/user_policy.go` (interface + impl)

- [ ] **Step 1: Add RestorePreFetch + RestorePostFetch to UserPolicy interface**

After `DeletePostFetch` line (~19), add:

```go
	RestorePreFetch(ctx context.Context, ids ...string) error
	RestorePostFetch(ctx context.Context, targets []domain.User) error
```

- [ ] **Step 2: Implement RestorePreFetch**

After `DeletePostFetch` implementation, add:

```go
func (p *userPolicy) RestorePreFetch(ctx context.Context, ids ...string) error {
	authUser := httpContext.MustGetUser(ctx)
	if authUser.Role.Rank() < domain.RoleAdmin.Rank() {
		return domain.NewError(domain.CodeUserRestoreSignerAdminRequiredForbidden)
	}
	for _, id := range ids {
		if id == authUser.Id {
			return domain.NewError(domain.CodeUserRestoreSelfTargetForbidden,
				domain.WithMetadata("user_id", authUser.Id))
		}
	}
	return nil
}
```

- [ ] **Step 3: Implement RestorePostFetch**

```go
func (p *userPolicy) RestorePostFetch(ctx context.Context, targets []domain.User) error {
	for _, t := range targets {
		if t.Role == domain.RoleSuperAdmin {
			return domain.NewError(domain.CodeUserRestoreSuperAdminTargetForbidden,
				domain.WithMetadata("user_id", t.Id))
		}
		if t.DeletedAt == nil {
			return domain.NewError(domain.CodeUserRestoreNotTrashedForbidden,
				domain.WithMetadata("user_id", t.Id))
		}
	}
	return nil
}
```

- [ ] **Step 4: Build check**

```bash
cd CredChain_Golang && go build ./feature/user/...
```

Expected: build fails — MockUserPolicy doesn't implement RestorePreFetch/RestorePostFetch yet (next task).

- [ ] **Step 5: Update MockUserPolicy**

In `infrastructure/testutil/mocks/user_policy.go`, add mock implementations for `RestorePreFetch` and `RestorePostFetch` with the standard testify mock pattern.

- [ ] **Step 6: Build check**

```bash
cd CredChain_Golang && go build ./...
```

Expected: exit 0.

- [ ] **Step 7: Commit**

```bash
git add feature/user/user_policy.go infrastructure/testutil/mocks/user_policy.go
git commit -m "feat(user): add RestorePreFetch/RestorePostFetch policy + mocks"
```

---

### Task 9: Restore request DTO

**Files:**
- Modify: `feature/user/user_request.go` (append)
- Create: (none — appended to existing file)

- [ ] **Step 1: Add UserRestoreRequest DTO**

Append to `feature/user/user_request.go`:

```go
type UserRestoreRequest struct {
	IDs []string `json:"ids"`
}

func (r UserRestoreRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.IDs, validation.Required, validation.Length(1, 0)),
	)
}
```

- [ ] **Step 2: Add request test**

Append to `feature/user/user_request_test.go`:

```go
func TestUserRestoreRequest_Validate_EmptyIDs(t *testing.T) {
	req := UserRestoreRequest{IDs: []string{}}
	err := req.Validate()
	assert.Error(t, err)
}

func TestUserRestoreRequest_Validate_Valid(t *testing.T) {
	req := UserRestoreRequest{IDs: []string{"01J123456789012345678901"}}
	err := req.Validate()
	assert.NoError(t, err)
}
```

- [ ] **Step 3: Run request tests**

```bash
cd CredChain_Golang && go test ./feature/user/... -run "TestUserRestoreRequest"
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add feature/user/user_request.go feature/user/user_request_test.go
git commit -m "feat(user): add UserRestoreRequest DTO + tests"
```

---

### Task 10: Restore user repository method + service

**Files:**
- Modify: `domain/user.go` (UserRepository interface — add Restore)
- Modify: `feature/user/gorm_user_repository.go` (impl)
- Modify: `infrastructure/testutil/mocks/user_repository.go` (mock)
- Modify: `feature/user/user_service.go` (interface + impl)

- [ ] **Step 1: Add Restore to UserRepository interface** (`domain/user.go`)

Insert after the `Delete` signature in the `UserRepository` interface:

```go
	// Restore unsets deleted_at for trashed users (batch operation).
	Restore(ctx context.Context, ids ...string) (int64, error)
```

- [ ] **Step 2: Implement Restore in gormUserRepository** (`feature/user/gorm_user_repository.go`)

Append after the `Delete` method:

```go
// Restore unsets deleted_at for trashed users (batch operation).
// Pure soft-delete reversal: clears the deleted_at timestamp.
// Returns: (int64, error) — rows affected count, error
func (r *gormUserRepository) Restore(ctx context.Context, ids ...string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id IN ?", ids).
		Update("deleted_at", nil)
	return result.RowsAffected, result.Error
}
```

- [ ] **Step 3: Update MockUserRepository** (`infrastructure/testutil/mocks/user_repository.go`)

Add mock implementation for `Restore` matching the interface.

- [ ] **Step 4: Add Restore to UserService interface** (`feature/user/user_service.go`)

After `TransferSuperAdmin` in the interface:

```go
	Restore(ctx context.Context, ids []string) ([]domain.User, int64, error)
```

- [ ] **Step 5: Implement Restore service** (`feature/user/user_service.go`)

```go
func (s *userService) Restore(ctx context.Context, ids []string) ([]domain.User, int64, error) {
	if err := s.policy.RestorePreFetch(ctx, ids...); err != nil {
		return nil, 0, err
	}
	var restored []domain.User
	var count int64
	err := s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		targets, err := uow.User().FindByIds(ctx, ids...)
		if err != nil {
			return err
		}
		if err := s.policy.RestorePostFetch(ctx, targets); err != nil {
			return err
		}
		count, err = uow.User().Restore(ctx, ids...)
		if err != nil {
			return err
		}
		restored, err = uow.User().FindByIds(ctx, ids...)
		if err != nil {
			return err
		}
		return s.syncBlockchainRoles(ctx, restored, domain.CodeUserRestoreBlockchainSyncFailed)
	})
	return restored, count, err
}
```

- [ ] **Step 6: Build check**

```bash
cd CredChain_Golang && go build ./...
```

Expected: exit 0.

- [ ] **Step 7: Commit**

```bash
git add domain/user.go feature/user/gorm_user_repository.go infrastructure/testutil/mocks/user_repository.go feature/user/user_service.go
git commit -m "feat(user): add Restore repo method + service (batch un-soft-delete + chain sync)"
```

---

### Task 11: Restore handler + route

**Files:**
- Modify: `feature/user/user_handler.go` (interface + impl)
- Modify: `infrastructure/http/router.go` (add route)

- [ ] **Step 1: Add Restore to UserHandler interface + impl**

Add `Restore(c *gin.Context)` to the `UserHandler` interface. Implementation:

```go
func (h *userHandler) Restore(c *gin.Context) {
	var req UserRestoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	users, count, err := h.userSvc.Restore(c.Request.Context(), req.IDs)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responseUsers := make([]response.User, len(users))
	for i, u := range users {
		responseUsers[i] = response.FromDomainUser(u)
	}
	responder.Send(c, domain.CodeUserRestoreSuccess, gin.H{
		"users":          responseUsers,
		"restored_count": count,
	})
}
```

- [ ] **Step 2: Register route**

In `infrastructure/http/router.go`, after the `DELETE /batch` route, add:

```go
				users.PUT("/batch/restore", gin.HandlerFunc(p.AdminRoleMiddleware), p.UserHandler.Restore)
```

- [ ] **Step 3: Build check**

```bash
cd CredChain_Golang && go build ./...
```

Expected: exit 0.

- [ ] **Step 4: Commit**

```bash
git add feature/user/user_handler.go infrastructure/http/router.go
git commit -m "feat(user): add Restore handler + PUT /api/users/batch/restore route"
```

---

### Task 12: Register codes in mapper + locales

**Files:**
- Modify: `infrastructure/http/responder/mapper.go`
- Modify: `infrastructure/http/responder/mapper_test.go`
- Modify: `locales/en.json`
- Modify: `locales/id.json`

- [ ] **Step 1: Add 3 verify verdict keys to CodeToMessageKey (mapper.go)**

Insert after `CodeCredentialVerifyNoMatch` entry:

```go
	domain.CodeCredentialVerifyHolderDisabled: "success_credential_verify_holder_disabled",
	domain.CodeCredentialVerifyIssuerDisabled: "success_credential_verify_issuer_disabled",
	domain.CodeCredentialVerifyPartyDisabled:  "success_credential_verify_party_disabled",
```

- [ ] **Step 2: Add 6 restore keys to CodeToMessageKey**

```go
	domain.CodeUserRestoreSuccess:                      "success_users_restore",
	domain.CodeUserRestoreSignerAdminRequiredForbidden: "error_users_restore_signer_admin_required",
	domain.CodeUserRestoreSelfTargetForbidden:          "error_users_restore_self_target_forbidden",
	domain.CodeUserRestoreSuperAdminTargetForbidden:    "error_users_restore_super_admin_target_forbidden",
	domain.CodeUserRestoreNotTrashedForbidden:          "error_users_restore_not_trashed_forbidden",
	domain.CodeUserRestoreBlockchainSyncFailed:         "error_users_restore_blockchain_sync_failed",
```

- [ ] **Step 3: Add 9 HTTP codes to HttpCodes (mapper.go)**

3 verify codes → 200, 6 restore codes → 5 entries at 403 + 300900 at 200 + 300945 at 500.

- [ ] **Step 4: Add 9 codes to allDomainCodes (mapper_test.go)**

- [ ] **Step 5: Add 9 entries to locales/en.json + locales/id.json**

en messages (see spec Section 6.1 and 11.8 for exact text).

- [ ] **Step 6: Run locale + mapper tests**

```bash
cd CredChain_Golang && go test ./infrastructure/http/responder/...
```

Expected: PASS (locale_keys_test + mapper_test enforce all registrations).

- [ ] **Step 7: Commit**

```bash
git add infrastructure/http/responder/mapper.go infrastructure/http/responder/mapper_test.go locales/en.json locales/id.json
git commit -m "feat(i18n): register 9 new codes (3 verify party-disabled + 6 restore) in mapper + locales"
```

---

### Task 13: Restore service tests

**Files:**
- Modify: `feature/user/user_service_test.go`

- [ ] **Step 1: Add Restore success test**

Mock FindByIds returns one trashed Holder user with `DeletedAt` set. Mock Restore policy checks pass. Mock AuthorityService UpdateUserRole succeeds. Expect 200, restored user has `DeletedAt == nil`.

- [ ] **Step 2: Add Restore Admin restores Admin peer test**

Trashed Admin target, Admin signer. Expect 200 — restore is an undo, not escalation.

- [ ] **Step 3: Add Restore SuperAdmin target forbidden test**

Trashed SuperAdmin target. Expect `CodeUserRestoreSuperAdminTargetForbidden` (300943).

- [ ] **Step 4: Add Restore live target forbidden test**

Target has `DeletedAt == nil`. Expect `CodeUserRestoreNotTrashedForbidden` (300944).

- [ ] **Step 5: Add Restore self-target forbidden test**

Signer ID in ids. Expect `CodeUserRestoreSelfTargetForbidden` (300942).

- [ ] **Step 6: Add Restore below-Admin forbidden test**

Holder signer. Expect `CodeUserRestoreSignerAdminRequiredForbidden` (300941).

- [ ] **Step 7: Add Restore blockchain failure rollback test**

Uses `PropagatingUnitOfWork`. AuthorityService.UpdateUserRole returns error. Expect `CodeUserRestoreBlockchainSyncFailed` (300945), DB rollback.

- [ ] **Step 8: Run all user tests**

```bash
cd CredChain_Golang && go test ./feature/user/...
```

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add feature/user/user_service_test.go
git commit -m "test(user): Restore endpoint service tests (7 scenarios)"
```

---

### Task 14: Doc sync + B6 removal

**Files:**
- Modify: `CredChain_Golang/AGENTS.md`
- Modify: `CredChain_Golang/ROLES.md`
- Modify: `docs/superpowers/plans/2026-06-09-roles-capability-revisions.md`

- [ ] **Step 1: Update AGENTS.md**

Targeted edits:
- Routes table: add `PUT /api/users/batch/restore` row
- Routes table: update `/api/credentials/verify` notes to mention party-disabled verdicts (400410-400412)
- Two-Method Policy Splits: add RestorePreFetch/RestorePostFetch bullet
- Chain Infrastructure → Verify verdicts: add 400410-400412 to the list
- Soft Delete section: note `Restore` unsets `deleted_at` + re-syncs DB role to chain
- Add cache-design paragraph (verify cache stores credential-level verdict only; holder/issuer re-checked live)

- [ ] **Step 2: Update ROLES.md**

Targeted edits:
- Route table: new `PUT /api/users/batch/restore` row (Admin+, handler Restore)
- Capability matrix: `Restore users` row (Admin ✓, SuperAdmin ✓)
- Policy rules: RestorePreFetch + RestorePostFetch tables
- Denied ops: "Restore SuperAdmin", "Restore live user", "Restore self" entries
- Verify row: note three party-disabled verdict codes (400410-400412) as possible responses

- [ ] **Step 3: Update plan doc**

Targeted edits:
- B4: mark as DONE (party-disabled verdicts implemented)
- B5: mark as fully DONE (Delete UoW unified + Restore implemented)
- B6: remove the entire section
- Update status note at top of Part B

- [ ] **Step 4: Commit**

```bash
git add AGENTS.md ROLES.md docs/superpowers/plans/2026-06-09-roles-capability-revisions.md
git commit -m "docs: sync AGENTS.md, ROLES.md, plan doc for B4/B5/B6"
```

---

### Task 15: Full verification

- [ ] **Step 1: Run all tests**

```bash
cd CredChain_Golang && go test ./...
```

Expected: all PASS.

- [ ] **Step 2: Run go vet**

```bash
cd CredChain_Golang && go vet ./...
```

Expected: exit 0.

- [ ] **Step 3: Check formatting**

```bash
cd CredChain_Golang && gofmt -l .
```

Expected: empty output.

- [ ] **Step 4: Push**

```bash
cd CredChain_Golang && git push credchain-go master
```
