# Verify Cache Revocation Fix + Edge Cases Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the verify endpoint returning "authentic" (400401) for revoked credentials due to stale MongoDB `credential_verifications` cache entries, and harden all related cache-lookup edge cases.

**Architecture:** Defense-in-depth — (A) reactive: re-check `RevokedAt` live in the cache-lookup path, and (B) proactive: delete cache entries post-revoke. Also handles the "credential deleted from DB" cache-hit edge case and the `verifyPickBestMatch` silent-fallback edge case.

**Tech Stack:** Go 1.25, MongoDB driver v2, testify mocks, uber-go/zap logging

## Global Constraints

- **NO-N+1 rule:** All batch repository operations execute a single query/aggregation regardless of batch size.
- **Cache deletion failure is non-fatal:** `s.logger.Warn(...)`, not a returned error.
- **Revocation takes priority over party-disabled:** If `cred.RevokedAt != nil`, verdict is `400402` (revoked) even if holder/issuer are soft-deleted.
- **Follow existing code patterns:** TDD (test first, then implementation), same test helper style, same mock patterns.
- **Tests must be white-box, in-package** (`package credential`).
- **Pre-push verification:** `go test ./... && go vet ./... && gofmt -l .` (last must produce zero output).

---

## Bug Analysis

### Bug 1 (reported): Cache survives revocation

**Root cause:** `credential_service.go:474-492` — the cache lookup path re-checks holder/issuer soft-delete status but does NOT re-check `cred.RevokedAt`. When a credential is revoked, the cached "authentic" verdict persists until TTL expiry (24h default), and indefinitely if re-verified daily (sliding TTL resets `created_at` on every upsert).

**Data flow trace:**
```
Revoke() → Postgres revoked_at set, on-chain token revoked
         → MongoDB credential_verifications cache NOT invalidated
         → Verify() → cache hit → returns cached 400401 (authentic)
```

**Code in question** (`credential_service.go:474-492`):
```go
if cached != nil {
    var cred *domain.Credential
    if cached.MatchedCredentialID != nil {
        cred, _ = s.repo.Find(ctx, *cached.MatchedCredentialID, verifyQuery)
    }
    code := cached.VerdictCode
    if code == domain.CodeCredentialVerifyAuthentic && cred != nil {
        // ONLY checks party-disabled, NOT RevokedAt
        holderGone := cred.Holder == nil || cred.Holder.DeletedAt != nil
        issuerGone := cred.Issuer == nil || cred.Issuer.DeletedAt != nil
        // ...
    }
    return code, cred, cached.SimilarityScore, cached.SimilarityPercent, nil
}
```

### Sliding TTL Explained

The MongoDB `Store` method (`mongo_credential_verification_repository.go:42-52`) uses `$set` (not `$setOnInsert`) for ALL fields including `created_at`:

```go
bson.M{"$set": bson.M{
    "verdict_code":          v.VerdictCode,
    "matched_credential_id": v.MatchedCredentialID,
    "similarity_score":      v.SimilarityScore,
    "similarity_percent":    v.SimilarityPercent,
    "created_at":            v.CreatedAt,    // <-- reset on every upsert
}}
```

The TTL index (`cmd/migrate_mongo.go:68`) deletes documents where `created_at + expireAfterSeconds < now`. Since `$set` overwrites `created_at` with `time.Now()` on every re-verify, the TTL window restarts. A document verified at least once per 24h lives forever. This is by design for active files but becomes dangerous when combined with the no-revocation-check bug — a revoked credential's cache entry becomes immortal if anyone re-verifies that file daily (e.g., an automated checker or attacker).

### All Edge Cases

| # | Edge Case | Severity | File:Line | Fix in this plan? |
|---|-----------|----------|-----------|-------------------|
| 1 | Cache hit returns "authentic" for revoked credential — no `RevokedAt` re-check | **High** | `credential_service.go:474-492` | **Yes** — reactive re-check in cache lookup |
| 2 | Cache hit on **deleted credential** (hard-deleted from Postgres) returns cached authentic verdict — `s.repo.Find` error ignored | Medium | `credential_service.go:476-477` | **Yes** — return integrity warning when cred not found |
| 3 | Sliding TTL (`$set` not `$setOnInsert`) means daily re-verification makes cache **never expire** | Medium | `mongo_credential_verification_repository.go:42-44` | Addressed by proactive delete on revoke |
| 4 | Exact-hash path: all nil TokenID → on-chain cross-check skipped | — | `credential_service.go:499-554` | **Not a bug** — lines 551-554 return `integrity_warning` correctly inside `if len(existing) > 0` |
| 5 | `verifyPickBestMatch`: silent fallback to `tied[0]` on `FindByIds` error | Low | `credential_service.go:617-619` | **Yes** — log warning before fallback |
| 6 | Credential re-issued after revoke with same file hash: old cache points to old credential ID | Low | via cache path | Addressed by proactive delete on revoke |
| 7 | No test for "cache hit returns revoked verdict" | Low | `credential_service_test.go` | **Yes** — new tests in Task 5 |

---

## Implementation Tasks

### Task 1: Add `DeleteByUploadedFileHashes` to domain interface

**Files:**
- Modify: `CredChain_Golang/domain/credential_verification.go:23-28`

**Expected compile break:** `mongoCredentialVerificationRepository` and `MockCredentialVerificationRepository` fail compilation (missing method).

```go
type CredentialVerificationRepository interface {
    FindByUploadedFileHash(ctx context.Context, hash string) (*CredentialVerification, error)
    Store(ctx context.Context, verification CredentialVerification) error
    DeleteByUploadedFileHashes(ctx context.Context, hashes []string) error
}
```

---

### Task 2: Implement `DeleteByUploadedFileHashes` in MongoDB repo

**Files:**
- Modify: `CredChain_Golang/feature/credential/mongo_credential_verification_repository.go:38-57` (append after `Store`)

```go
func (r *mongoCredentialVerificationRepository) DeleteByUploadedFileHashes(ctx context.Context, hashes []string) error {
    if len(hashes) == 0 {
        return nil
    }
    _, err := r.coll.DeleteMany(ctx, bson.M{"uploaded_file_hash": bson.M{"$in": hashes}})
    return err
}
```

---

### Task 3: Add `DeleteByUploadedFileHashes` to test mock

**Files:**
- Modify: `CredChain_Golang/infrastructure/testutil/mocks/credential_verification_repository.go:23-27`

```go
func (m *MockCredentialVerificationRepository) DeleteByUploadedFileHashes(ctx context.Context, hashes []string) error {
    return m.Called(ctx, hashes).Error(0)
}
```

---

### Task 4: Fix `verifyPickBestMatch` silent fallback (edge case 5)

**Files:**
- Modify: `CredChain_Golang/feature/credential/credential_service.go:617-619`
- Test: `CredChain_Golang/feature/credential/credential_service_test.go` (new test appended)

**No new logger dependencies needed** — `s.logger` already available on `credentialService`.

#### Test 4.0: TieBreak_FindByIdsErrorFallsBackWithLog (RED)

```go
func TestVerify_Fuzzy_TieBreak_FindByIdsErrorFallsBackWithLog(t *testing.T) {
    user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
    ctx := ctxWithAuth(&user)

    now := time.Now()
    sharedID1 := "id-1"
    sharedID2 := "id-2"
    cred1 := domain.Credential{
        ID: "c1", HolderUserID: "h1", IssuerUserID: user.Id,
        FileHash: "0xabc", TokenID: lo.ToPtr("100"), IssuedAt: now.Add(-2 * time.Hour),
    }
    cred2 := domain.Credential{
        ID: "c2", HolderUserID: "h2", IssuerUserID: user.Id,
        FileHash: "0xdef", TokenID: lo.ToPtr("200"), IssuedAt: now.Add(-1 * time.Hour),
    }

    m := &testCredentialMocks{
        credRepo: &mocks.MockCredentialRepository{},
        verRepo:  &mocks.MockCredentialVerificationRepository{},
        extRepo:  &mocks.MockCredentialExtractionRepository{},
        aiClient: &mocks.MockPythonAIClient{},
        regSvc:   &mocks.MockRegistryService{},
    }
    m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(nil, nil)
    m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
    m.aiClient.On("ExtractIDs", mock.Anything, mock.Anything).Return([]pyai.ExtractedID{
        {Value: sharedID1}, {Value: sharedID2},
    }, nil)

    // Two extractions with same intersection count → tie
    r1 := domain.CredentialExtraction{
        CredentialID: "c1",
        IDs:          []domain.CredentialExtractedID{{Value: sharedID1}, {Value: sharedID2}},
    }
    r2 := domain.CredentialExtraction{
        CredentialID: "c2",
        IDs:          []domain.CredentialExtractedID{{Value: sharedID1}, {Value: sharedID2}},
    }
    m.extRepo.On("FindRankedByIds", mock.Anything, mock.Anything, 10).Return(
        []domain.CredentialExtraction{r1, r2}, nil,
    )
    // FindByIds errors → should fall back to tied[0] with logged warning
    m.credRepo.On("FindByIds", mock.Anything, []string{"c1", "c2"}, mock.Anything).Return(
        nil, assert.AnError,
    )
    // r1's credential is tied[0], so its embedding is used for AI Verify
    m.aiClient.On("Verify", mock.Anything, mock.Anything, r1.Embedding).Return(
        pyai.VerifyResult{Verdict: "authentic", SimilarityScore: 0.99, SimilarityPercent: "99.00"}, nil,
    )
    m.credRepo.On("Find", mock.Anything, "c1", mock.Anything).Return(&cred1, nil)
    m.verRepo.On("Store", mock.Anything, mock.Anything).Return(nil)

    svc := newTestCredentialService(m)
    code, _, _, _, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test")})

    assert.NoError(t, err)
    assert.Equal(t, domain.CodeCredentialVerifyAuthentic, code)
    m.credRepo.AssertCalled(t, "FindByIds", mock.Anything, []string{"c1", "c2"}, mock.Anything)
}
```

**Expected RED phase:** Test FAIL — assertion passes but no log warning emitted.

**Implementation (GREEN phase):** Replace `credential_service.go:617-619`:

```go
    creds, err := s.repo.FindByIds(ctx, ids, nil)
    if err != nil {
        s.logger.Warn("verifyPickBestMatch: FindByIds failed, falling back to first candidate",
            zap.Error(err),
            zap.Int("candidate_count", len(tied)),
        )
        return tied[0]
    }
```

---

### Task 5: Add revocation re-check in cache lookup path (reactive fix for edge cases 1, 2, 6)

**Files:**
- Modify: `CredChain_Golang/feature/credential/credential_service.go:474-492`
- Test: `CredChain_Golang/feature/credential/credential_service_test.go` (new tests appended)

**Tests to write (RED phase first):**

#### Test 5.1: CacheHit_RevokedCredential (edge case 1)

```go
func TestVerify_CacheHit_RevokedCredential(t *testing.T) {
    user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
    ctx := ctxWithAuth(&user)

    now := time.Now()
    credID := "01J0000000000000000000000A"
    cached := &domain.CredentialVerification{
        VerdictCode:         domain.CodeCredentialVerifyAuthentic,
        MatchedCredentialID: &credID,
    }
    cred := &domain.Credential{
        ID:        credID,
        Holder:    &domain.User{},
        Issuer:    &domain.User{},
        RevokedAt: &now,
    }

    m := &testCredentialMocks{
        credRepo: &mocks.MockCredentialRepository{},
        verRepo:  &mocks.MockCredentialVerificationRepository{},
        extRepo:  &mocks.MockCredentialExtractionRepository{},
        aiClient: &mocks.MockPythonAIClient{},
        regSvc:   &mocks.MockRegistryService{},
    }
    m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(cached, nil)
    m.credRepo.On("Find", mock.Anything, credID, mock.Anything).Return(cred, nil)

    svc := newTestCredentialService(m)
    code, _, _, _, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test-file")})

    assert.NoError(t, err)
    assert.Equal(t, domain.CodeCredentialVerifyRevoked, code)
    m.aiClient.AssertNotCalled(t, "ExtractIDs", mock.Anything, mock.Anything)
    m.aiClient.AssertNotCalled(t, "Verify", mock.Anything, mock.Anything, mock.Anything)
}
```

#### Test 5.2: RevokedOverridesPartyDisabled (edge case 1 + party-disabled interaction)

```go
func TestVerify_CacheHit_RevokedOverridesPartyDisabled(t *testing.T) {
    user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
    ctx := ctxWithAuth(&user)

    now := time.Now()
    delTime := time.Now().Add(-1 * time.Hour)
    holder := fixtures.NewDomainUser(fixtures.WithID("holder-1"), fixtures.WithRole(domain.RoleHolder))
    holder.DeletedAt = &delTime
    issuer := fixtures.NewDomainUser(fixtures.WithID("issuer-1"), fixtures.WithRole(domain.RoleIssuer))

    credID := "c1"
    cached := &domain.CredentialVerification{
        VerdictCode:         domain.CodeCredentialVerifyAuthentic,
        MatchedCredentialID: &credID,
    }
    cred := &domain.Credential{
        ID:        credID,
        Holder:    &holder,
        Issuer:    &issuer,
        RevokedAt: &now,
    }

    m := &testCredentialMocks{
        credRepo: &mocks.MockCredentialRepository{},
        verRepo:  &mocks.MockCredentialVerificationRepository{},
        extRepo:  &mocks.MockCredentialExtractionRepository{},
        aiClient: &mocks.MockPythonAIClient{},
        regSvc:   &mocks.MockRegistryService{},
    }
    m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(cached, nil)
    m.credRepo.On("Find", mock.Anything, credID, mock.Anything).Return(cred, nil)

    svc := newTestCredentialService(m)
    code, _, _, _, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test")})

    assert.NoError(t, err)
    assert.Equal(t, domain.CodeCredentialVerifyRevoked, code)
}
```

#### Test 5.3: PreservesNonAuthenticVerdict (non-authentic verdicts unchanged)

```go
func TestVerify_CacheHit_PreservesNonAuthenticVerdict(t *testing.T) {
    user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
    ctx := ctxWithAuth(&user)

    credID := "c1"
    cached := &domain.CredentialVerification{
        VerdictCode:         domain.CodeCredentialVerifyTampered,
        MatchedCredentialID: &credID,
    }
    cred := &domain.Credential{
        ID:     credID,
        Holder: &domain.User{},
        Issuer: &domain.User{},
    }

    m := &testCredentialMocks{
        credRepo: &mocks.MockCredentialRepository{},
        verRepo:  &mocks.MockCredentialVerificationRepository{},
        extRepo:  &mocks.MockCredentialExtractionRepository{},
        aiClient: &mocks.MockPythonAIClient{},
        regSvc:   &mocks.MockRegistryService{},
    }
    m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(cached, nil)
    m.credRepo.On("Find", mock.Anything, credID, mock.Anything).Return(cred, nil)

    svc := newTestCredentialService(m)
    code, _, _, _, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test")})

    assert.NoError(t, err)
    assert.Equal(t, domain.CodeCredentialVerifyTampered, code)
}
```

#### Test 5.4: CredentialNotFound (edge case 2)

```go
func TestVerify_CacheHit_CredentialNotFound(t *testing.T) {
    user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
    ctx := ctxWithAuth(&user)

    credID := "01J0000000000000000000000A"
    cached := &domain.CredentialVerification{
        VerdictCode:         domain.CodeCredentialVerifyAuthentic,
        MatchedCredentialID: &credID,
    }

    m := &testCredentialMocks{
        credRepo: &mocks.MockCredentialRepository{},
        verRepo:  &mocks.MockCredentialVerificationRepository{},
        extRepo:  &mocks.MockCredentialExtractionRepository{},
        aiClient: &mocks.MockPythonAIClient{},
        regSvc:   &mocks.MockRegistryService{},
    }
    m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(cached, nil)
    m.credRepo.On("Find", mock.Anything, credID, mock.Anything).Return(nil, gorm.ErrRecordNotFound)

    svc := newTestCredentialService(m)
    code, cred, _, _, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test-file")})

    assert.NoError(t, err)
    assert.Equal(t, domain.CodeCredentialVerifyIntegrityWarning, code)
    assert.Nil(t, cred)
    m.aiClient.AssertNotCalled(t, "ExtractIDs", mock.Anything, mock.Anything)
}
```

**Expected RED phase:** All 4 new tests fail — `RevokedCredential` returns 400401 instead of 400402; `CredentialNotFound` returns 400401 instead of 400403.

**Implementation (GREEN phase):** Replace `credential_service.go:474-492`:

```go
    if cached != nil {
        var cred *domain.Credential
        if cached.MatchedCredentialID != nil {
            cred, _ = s.repo.Find(ctx, *cached.MatchedCredentialID, verifyQuery)
        }
        code := cached.VerdictCode
        if cred != nil {
            if cred.RevokedAt != nil {
                code = domain.CodeCredentialVerifyRevoked
            } else if code == domain.CodeCredentialVerifyAuthentic {
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
        } else if cached.MatchedCredentialID != nil {
            code = domain.CodeCredentialVerifyIntegrityWarning
        }
        return code, cred, cached.SimilarityScore, cached.SimilarityPercent, nil
    }
```

**Key logic change:**
1. If `cred` exists AND `RevokedAt != nil` → override to `400402` (revoked) regardless of cached verdict
2. Else if `cred` exists AND cached verdict is `400401` (authentic) → check party-disabled as before
3. Else if `cred` is nil but cache had a `MatchedCredentialID` → `400403` (integrity warning), credential no longer exists
4. Otherwise → return cached verdict as-is

---

### Task 6: Proactive cache deletion on credential revoke (edge cases 3, 6)

**Files:**
- Modify: `CredChain_Golang/feature/credential/credential_service.go:386-454` (Revoke method)
- Test: `CredChain_Golang/feature/credential/credential_service_test.go` (new tests appended)

**Tests to write (RED phase first):**

#### Test 6.1: DeletesVerificationCache

```go
func TestRevoke_DeletesVerificationCache(t *testing.T) {
    issuer := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
    ctx := ctxWithAuth(&issuer)

    now := time.Now()
    target := domain.Credential{
        ID:           "c1",
        HolderUserID: "h1",
        IssuerUserID: issuer.Id,
        FileHash:     "0xabc123def",
        TokenID:      lo.ToPtr("12345"),
        IssuedAt:     now.Add(-1 * time.Hour),
    }

    m := &testCredentialMocks{
        credRepo: &mocks.MockCredentialRepository{},
        uow: func() *mocks.MockUnitOfWork {
            uow := &mocks.MockUnitOfWork{}
            uowCredRepo := &mocks.MockCredentialRepository{}
            uow.On("Credential").Return(uowCredRepo)
            uowCredRepo.On("FindByIds", mock.Anything, []string{"c1"}, mock.Anything).Return([]domain.Credential{target}, nil)
            uowCredRepo.On("Update", mock.Anything, mock.Anything).Return([]domain.Credential{target}, nil)
            return uow
        }(),
        verRepo: &mocks.MockCredentialVerificationRepository{},
        regSvc:  &mocks.MockRegistryService{},
    }
    m.uow.On("Execute", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
        fn := args.Get(1).(func(domain.UnitOfWork) error)
        _ = fn(m.uow)
    })
    m.regSvc.On("RevokeCredentials", mock.Anything, mock.Anything, mock.Anything).Return(nil)
    m.verRepo.On("DeleteByUploadedFileHashes", mock.Anything, []string{"0xabc123def"}).Return(nil)

    svc := newTestCredentialService(m)
    _, err := svc.Revoke(ctx, "c1")
    assert.NoError(t, err)
    m.verRepo.AssertCalled(t, "DeleteByUploadedFileHashes", mock.Anything, []string{"0xabc123def"})
}
```

#### Test 6.2: FailureNotFatal

```go
func TestRevoke_DeleteVerificationCache_FailureNotFatal(t *testing.T) {
    issuer := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
    ctx := ctxWithAuth(&issuer)

    now := time.Now()
    target := domain.Credential{
        ID:           "c1",
        HolderUserID: "h1",
        IssuerUserID: issuer.Id,
        FileHash:     "0xabc123def",
        TokenID:      lo.ToPtr("12345"),
        IssuedAt:     now.Add(-1 * time.Hour),
    }

    m := &testCredentialMocks{
        credRepo: &mocks.MockCredentialRepository{},
        uow: func() *mocks.MockUnitOfWork {
            uow := &mocks.MockUnitOfWork{}
            uowCredRepo := &mocks.MockCredentialRepository{}
            uow.On("Credential").Return(uowCredRepo)
            uowCredRepo.On("FindByIds", mock.Anything, []string{"c1"}, mock.Anything).Return([]domain.Credential{target}, nil)
            uowCredRepo.On("Update", mock.Anything, mock.Anything).Return([]domain.Credential{target}, nil)
            return uow
        }(),
        verRepo: &mocks.MockCredentialVerificationRepository{},
        regSvc:  &mocks.MockRegistryService{},
    }
    m.uow.On("Execute", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
        fn := args.Get(1).(func(domain.UnitOfWork) error)
        _ = fn(m.uow)
    })
    m.regSvc.On("RevokeCredentials", mock.Anything, mock.Anything, mock.Anything).Return(nil)
    m.verRepo.On("DeleteByUploadedFileHashes", mock.Anything, mock.Anything).Return(assert.AnError)

    svc := newTestCredentialService(m)
    revokedCredentials, err := svc.Revoke(ctx, "c1")
    assert.NoError(t, err)
    assert.Len(t, revokedCredentials, 1)
    m.verRepo.AssertCalled(t, "DeleteByUploadedFileHashes", mock.Anything, mock.Anything)
}
```

**Expected RED phase:** `TestRevoke_DeletesVerificationCache` FAIL — `DeleteByUploadedFileHashes` never called.

**Implementation (GREEN phase):** Restructure Revoke return (`credential_service.go:389-454`):

Change:
```go
    var revoked []domain.Credential
    err := s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
        // ... existing UoW code unchanged ...
        revoked, err = uow.Credential().Update(ctx, updates...)
        // ...
    })
    return revoked, err
```

To:
```go
    var revokedCredentials []domain.Credential
    err := s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
        // ... existing UoW code unchanged ...
        revokedCredentials, err = uow.Credential().Update(ctx, updates...)
        // ...
    })
    if err != nil {
        return nil, err
    }
    fileHashes := lo.Map(revokedCredentials, func(c domain.Credential, _ int) string { return c.FileHash })
    if len(fileHashes) > 0 {
        if delErr := s.verificationRepo.DeleteByUploadedFileHashes(ctx, fileHashes); delErr != nil {
            s.logger.Warn("failed to delete verification cache after revoke",
                zap.Error(delErr),
                zap.Int("credential_count", len(revokedCredentials)),
            )
        }
    }
    return revokedCredentials, nil
```

**Variable rename:** `revoked` → `revokedCredentials` for clarity (the local variable in the UoW closure and the outer variable).

---

### Task 7: Final verification

- [ ] `go test ./...` — all tests pass (including 7 new tests)
- [ ] `go vet ./...` — zero warnings
- [ ] `gofmt -l .` — zero output
- [ ] `go test -race ./feature/credential/...` — no races

---

## Summary: Edge Cases Addressed

| # | Edge Case | Fix | Task |
|---|-----------|-----|------|
| 1 | Cache hit returns authentic for revoked | Re-check `RevokedAt` in cache lookup | Task 5 |
| 2 | Cache hit on deleted credential | Return `integrity_warning` when `Find` fails | Task 5 |
| 3 | Sliding TTL perpetuates stale cache | Proactive cache deletion on revoke | Task 6 |
| 4 | All nil TokenID → fuzzy fallthrough | **Not a bug** — lines 551-554 correctly return `integrity_warning` | — |
| 5 | `verifyPickBestMatch` silent fallback | Log warning before fallback | Task 4 |
| 6 | Re-issue with same file hash | Addressed by proactive delete on revoke | Task 6 |
| 7 | No cache-hit revocation test | 5 new tests added (Tasks 4 + 5 + 6) | Tasks 4-6 |

## No New Codes or Locale Updates

All codes used (`400401` authentic, `400402` revoked, `400403` integrity_warning) already exist in:
- `domain/codes.go`
- `mapper.go` (CodeToMessageKey + HttpCodes)
- `locales/en.json` and `locales/id.json`

No new constants, no locale additions needed.
