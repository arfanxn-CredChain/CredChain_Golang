# Credential Verify — Party-Disabled Verdicts + B5 Delete UoW Unification

Date: 2026-06-10
Status: In design (not yet implemented)

## Context

When a user is soft-deleted, the credentials they hold or issued remain in the
database and on-chain. The `/verify` endpoint should inform verifiers that a
party (holder/issuer) is deactivated, so the credential requires manual review.

Additionally, the `Delete` UoW pattern is being unified (matching the prior
`UpdateRole` unification in `68a27d7`), and B6 is being removed from the plan.

## 1. New Verdict Codes

Three party-disabled verdicts replace the single `PartyDisabled` code from the
initial brainstorm. The verifier gets specific info about which party is gone.

| Code | Constant | Meaning |
|------|----------|---------|
| `400410` | `CodeCredentialVerifyHolderDisabled` | Only the credential **holder** is deleted/missing |
| `400411` | `CodeCredentialVerifyIssuerDisabled` | Only the issuing authority is deleted/missing |
| `400412` | `CodeCredentialVerifyPartyDisabled` | **Both** holder and issuer are deleted/missing |

**Override rule (Option B):** these verdicts only override `CodeCredentialVerifyAuthentic` (400401).
Revoked, Tampered, IntegrityWarning, Suspicious, etc. are stronger signals about
the credential itself and persist unchanged. If the credential is Revoked AND
the holder is deleted, the verifier sees `Revoked` (400402), not a
party-disabled code.

**Response shape:** no change to `response.CredentialVerify`. The matched
credential is served as before with its `holder_user_id` / `issuer_user_id`.
No additional preload needed on the response path beyond what is already done
for the verdict computation (see Section 3).

**HTTP status:** all three verdicts map to **200 OK** (matches `Revoked`,
`Tampered`, etc.). Only `IntegrityWarning` (DB/chain mismatch) gets 409
Conflict; that precedent stands.

## 2. CredentialRepository Interface Change

Reasoning: the verify exact-hash path currently loads credentials via
`FindByFileHashes(ctx, hashes...)` which returns bare entities with no relation
preloads. To check holder/issuer `DeletedAt`, the verify path needs those
relations loaded. The cleanest path is to add optional `*Query` parameters to
all `Find*` methods that lack them, so callers can pass `{Includes: ["holder",
"issuer"]}` at the repository level without a separate re-fetch.

### 2.1 New signatures (domain/credential.go)

```go
// Before
FindByIds(ctx context.Context, ids ...string) ([]Credential, error)
FindByHolderId(ctx context.Context, holderID string) ([]Credential, error)
FindByFileHashes(ctx context.Context, hashes ...string) ([]Credential, error)

// After — query is the LAST parameter, slices are NOT variadic
FindByIds(ctx context.Context, ids []string, query *domainQuery.Query) ([]Credential, error)
FindByHolderId(ctx context.Context, holderID string, query *domainQuery.Query) ([]Credential, error)
FindByFileHashes(ctx context.Context, hashes []string, query *domainQuery.Query) ([]Credential, error)
```

### 2.2 Existing callers (all pass nil)

Callers that don't need includes pass `nil` as the query argument — no
behavioral change:

| Caller | Method | New call |
|--------|--------|----------|
| `credentialService.Issue` (line 175) | `FindByFileHashes` | `FindByFileHashes(ctx, hashes, nil)` |
| `credentialService.Issue` (line 163) | `FindByIds` (via `userRepo`) | Not changed — `UserRepository.FindByIds` stays variadic |
| `credentialService.Revoke` | `FindByIds` | `FindByIds(ctx, ids, nil)` |
| `credentialService.Verify` (exact-hash, line 457) | `FindByFileHashes` | `FindByFileHashes(ctx, []string{uploadedHash}, verifyQuery)` where verifyQuery includes holder+issuer |
| `credentialService.Verify` (fuzzy, line 500) | `Find` | Already accepts `*Query` — switch to verifyQuery |
| `credentialService.Verify` (cache hit, line 448) | `Find` | Already accepts `*Query` — switch to verifyQuery |
| `credentialService.ReExtract` | `FindByIds` | `FindByIds(ctx, ids, nil)` |
| `userService.deleteUserAndSyncBlockchain` | `FindByIds` (via `userRepo`) | Not changed |

### 2.3 Repository implementation (gorm_credential_repository.go)

Each updated method gains a `query *domainQuery.Query` param and uses
`preloadByIncludes(db, query)` when the query is non-nil. All existing logic
(allowlist checks, WHERE, batch ORDER BY) is untouched.

```go
func (r *gormCredentialRepository) FindByIds(ctx context.Context, ids []string, query *domainQuery.Query) ([]domain.Credential, error) {
    if len(ids) == 0 { return []domain.Credential{}, nil }
    db := r.db.WithContext(ctx)
    if query != nil { db = preloadByIncludes(db, query) }
    var rows []model.Credential
    if err := db.Where("id IN ?", ids).Find(&rows).Error; err != nil { return nil, err }
    // ...existing conversion...
}
```

Same pattern for `FindByHolderId` and `FindByFileHashes`.

### 2.4 Mock interface (testutil/mocks/)

`MockCredentialRepository` method signatures updated to match. All args use
`mock.Anything` in existing tests — those still match after the change. New
tests use `mock.AnythingOfType("*domainQuery.Query")` or `mock.Anything` for
the nullable query param.

## 3. Verify Flow: Party-Disabled Check

### 3.1 Query construction

All three resolution points use a single `verifyQuery`:

```go
verifyQuery := &domainQuery.Query{Includes: []string{"holder", "issuer"}}
```

Repository `Preload` runs a single batch IN-clause for each include regardless
of result count (already the existing pattern — no N+1).

### 3.2 Inline check (no helper)

After the credential is resolved and the base verdict is computed, check
party-disabled status:

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

**Why `cred.Holder == nil` means disabled:** when the user row is missing
(record-not-found in the Preload batch), GORM sets the relation to nil.
Combined with `DeletedAt != nil` for soft-deleted users, this covers both
hard-delete and soft-delete scenarios.

### 3.3 Injection points

| Branch | File/Line | Change |
|--------|-----------|--------|
| Cache hit | `credential_service.go:448` | Pass `verifyQuery` to `s.repo.Find`; add party-disabled check after `VerdictCode` is read |
| Exact-hash | `credential_service.go:457` | `FindByFileHashes(ctx, []string{uploadedHash}, verifyQuery)`; add check after verdict computed |
| Fuzzy | `credential_service.go:500` | Pass `verifyQuery` to `s.repo.Find`; add check after verdict computed |

### 3.4 Cache behavior

The verify cache (MongoDB `credential_verifications`, 24h TTL) stores only the
**credential-level** verdict snapshot: `verdict_code`, `matched_credential_id`,
`similarity_score`, `similarity_percent`. It does NOT cache holder/issuer
state.

The party-disabled check runs on every cache hit (holder/issuer re-checked
live against the users table). This means a cached `Authentic` verdict will be
overridden to the appropriate disabled verdict when the holder or issuer has
been soft-deleted after the cache was written. No cache schema change, no
cache invalidation in the user-delete flow.

## 4. B5 — Delete UoW Unification

### 4.1 Current (two UoW calls)

```
Delete (user_service.go:415-434)
  ├─ UoW #1 (read):  FindByIds + policy.DeletePostFetch → commit
  └─ UoW #2 (write): Delete (soft-delete) + syncBlockchainRoles (chain RoleNone) → commit
```

### 4.2 Unified (one UoW call, matching `UpdateRole` pattern)

```
Delete
  └─ policy.DeletePreFetch (no DB, before UoW)
  └─ s.uow.Execute:
       ├─ FindByIds (unscoped)
       ├─ policy.DeletePostFetch (targets)
       ├─ Delete (GORM soft-delete, batch)
       └─ syncBlockchainRoles (RoleNone on-chain, live targets only)
```

Chain failure rolls back the DB soft-delete (single TX). The `deleteUserAndSyncBlockchain`
helper is removed. The `syncBlockchainRoles` call and live-targets filter are
inlined into the unified UoW closure.

### 4.3 No behavioral change

- `FindByIds` is unscoped — still sees soft-deleted rows inside the TX
- Live-targets filter checks `DeletedAt == nil` on the same structs — works
  identically
- Idempotency preserved: GORM soft-delete is a no-op on already-trashed rows
- Batch delete under single `db.Delete(...)` call — respects NO-N+1 rule

## 5. B6 Removal

Remove the B6 (verify audit-log) section from
`docs/superpowers/plans/2026-06-09-roles-capability-revisions.md`. The verify
endpoint stays public and anonymous. No other file references B6.

## 6. Code Registration

| What | File |
|------|------|
| `CodeCredentialVerifyHolderDisabled = 400410` | `domain/codes.go` |
| `CodeCredentialVerifyIssuerDisabled = 400411` | `domain/codes.go` |
| `CodeCredentialVerifyPartyDisabled = 400412` | `domain/codes.go` |
| Key → msg mapping (3 entries) | `infrastructure/http/responder/mapper.go` — `CodeToMessageKey` |
| HTTP 200 (3 entries) | `mapper.go` — `HttpCodes` |
| Add to `allDomainCodes` (3 entries) | `mapper_test.go` |
| Locale — en (3 entries) | `locales/en.json` |
| Locale — id (3 entries) | `locales/id.json` |

### 6.1 Locale message keys

| Key | en |
|-----|----|
| `success_credential_verify_holder_disabled` | `The credential holder has been deactivated. Credential validity requires manual review.` |
| `success_credential_verify_issuer_disabled` | `The issuing authority has been deactivated. Credential validity requires manual review.` |
| `success_credential_verify_party_disabled` | `Both the credential holder and issuing authority have been deactivated. Credential validity requires manual review.` |

### 6.2 Locale message keys (id)

| Key | id |
|-----|----|
| `success_credential_verify_holder_disabled` | `Pemegang kredensial telah dinonaktifkan. Validitas kredensial memerlukan tinjauan manual.` |
| `success_credential_verify_issuer_disabled` | `Otoritas penerbit telah dinonaktifkan. Validitas kredensial memerlukan tinjauan manual.` |
| `success_credential_verify_party_disabled` | `Pemegang dan otoritas penerbit kredensial telah dinonaktifkan. Validitas kredensial memerlukan tinjauan manual.` |

## 7. Doc Sync

| File | Edit |
|------|------|
| `AGENTS.md` | Add cache-design paragraph (verify cache stores credential-level verdict only; holder/issuer state re-checked live). Add party-disabled verdict codes to verify section. Add UoW consistency note for Delete. |
| `ROLES.md` | Verify row: note the three disabled verdict codes as possible responses |
| `docs/superpowers/plans/2026-06-09-roles-capability-revisions.md` | B4/B5 done; update B5 detail to reflect approach; remove B6 |

## 8. Files Touched (implementation-order view)

| # | File | Change |
|---|------|--------|
| 1 | `domain/credential.go` | +3 interface signature changes (`*Query` param) |
| 2 | `domain/codes.go` | +3 verdict codes (400410, 400411, 400412) |
| 3 | `feature/credential/gorm_credential_repository.go` | Update 3 repo methods to accept + use `*Query` |
| 4 | `feature/credential/credential_service.go` | Update callers of changed repo methods; add party-disabled check at 3 Verify branches; inline `verifyQuery` |
| 5 | `feature/user/user_service.go` | Unify `Delete` into single UoW; remove `deleteUserAndSyncBlockchain` |
| 6 | `infrastructure/testutil/mocks/` | Update `MockCredentialRepository` method signatures |
| 7 | `infrastructure/http/responder/mapper.go` | +6 entries (3 msg keys + 3 http codes) |
| 8 | `infrastructure/http/responder/mapper_test.go` | +3 `allDomainCodes` |
| 9 | `locales/en.json` | +3 entries |
| 10 | `locales/id.json` | +3 entries |
| 11 | `AGENTS.md` | Cache design note, verify routes update, Delete UoW note |
| 12 | `ROLES.md` | Verify row update |
| 13 | `docs/superpowers/plans/2026-06-09-roles-capability-revisions.md` | B4 done, B5 done, B6 removed |

## 9. Verification

```bash
cd CredChain_Golang
go test ./... && go vet ./... && gofmt -l .
```

`locale_keys_test.go` and `mapper_test.go` auto-catch missing registrations.

## 10. Test Plan

### New service tests (`credential_service_test.go`)

| Test | Scenario | Expected |
|------|----------|----------|
| `TestVerify_HolderDisabled_OverridesAuthentic` | Credential is Authentic, holder has `DeletedAt` set | Returns 400410, credential in payload |
| `TestVerify_IssuerDisabled_OverridesAuthentic` | Credential is Authentic, issuer has `DeletedAt` set | Returns 400411, credential in payload |
| `TestVerify_PartyDisabled_OverridesAuthentic` | Both holder and issuer soft-deleted | Returns 400412, credential in payload |
| `TestVerify_HolderDisabled_DoesNotOverrideRevoked` | Credential is Revoked AND holder is deleted | Returns 400402 (Revoked), not a party-disabled code |
| `TestVerify_PartyDisabled_MissingHolder` | Holder row missing entirely (record-not-found) | Returns 400410 (holderGone via `cred.Holder == nil`) |
| `TestVerify_PartyDisabled_NoOverrideForTampered` | Credential is Tampered AND issuer deleted | Returns 400404 (Tampered) |

### Existing repo tests — updated

| Test | Change |
|------|--------|
| `FindByIds`, `FindByHolderId`, `FindByFileHashes` calls | Add `nil` query argument to match new signatures |

### Existing service tests — updated

| Test | Change |
|------|--------|
| `TestUserService_Delete_BlockchainRevokeFailure_RollsBack` | Still passes (uses `PropagatingUnitOfWork`, single TX) |
| `TestUserService_Delete_Success_CallsAuthorityService` | Still passes |
| Verify tests using `MockCredentialRepository.FindByFileHashes` | Add `nil` query argument |

### Policy tests — unchanged

`CredentialPolicy` is untouched in this spec.

### Locale/mapper tests — auto-caught

`locale_keys_test.go` enforces key existence. `mapper_test.go` enforces code registration.
Adding three codes to `allDomainCodes` and three keys to locale files is sufficient.

---

## 11. Restore Endpoint (`PUT /api/users/batch/restore`)

### 11.1 Route + Auth

| Method | Route | Auth | Min Role |
|--------|-------|------|----------|
| PUT | `/api/users/batch/restore` | Authenticated | Admin+ |

Mirrors the existing `PUT /api/users/batch/role` sub-action pattern. Owned by the
user feature — handler, service, policy, and request DTO all live under
`feature/user/`.

### 11.2 Request DTO

```json
{"ids": ["01J...", "01K..."]}
```

No role parameter — restore re-uses the preserved DB role (untouched on delete).
Request DTO: `UserRestoreRequest` with Ozzo validation (non-empty `ids` slice).

### 11.3 Dedicated Domain Codes (3009xx Block)

Following the existing pattern where every operation owns a code block:
3002xx (Store), 3005xx (UpdateRole), 3006xx (TransferSuperAdmin),
3007xx (Delete), 3008xx (Update). Restore gets 3009xx. No reuse of existing codes.

| Value | Constant | Policy location | Purpose | HTTP |
|-------|----------|-----------------|---------|------|
| `300900` | `CodeUserRestoreSuccess` | — | Restore completed | 200 |
| `300941` | `CodeUserRestoreSignerAdminRequiredForbidden` | `RestorePreFetch` | Signer below Admin | 403 |
| `300942` | `CodeUserRestoreSelfTargetForbidden` | `RestorePreFetch` | Cannot restore self | 403 |
| `300943` | `CodeUserRestoreSuperAdminTargetForbidden` | `RestorePostFetch` | Target role is SuperAdmin | 403 |
| `300944` | `CodeUserRestoreNotTrashedForbidden` | `RestorePostFetch` | Target not trashed (strict validation, Option B) | 403 |
| `300945` | `CodeUserRestoreBlockchainSyncFailed` | Service | Chain restore failed, DB rolled back | 500 |

### 11.4 Policy — Two-Method Split

Following the existing pre-fetch / post-fetch pattern established by
`DeletePreFetch` / `DeletePostFetch` and `UpdateRolePreFetch` /
`UpdateRolePostFetch`. Added to the `UserPolicy` interface.

**RestorePreFetch** (no DB access):

| Rule | Condition | Code |
|------|-----------|------|
| Signer must be Admin+ | `authUser.Role.Rank() < RoleAdmin.Rank()` | `CodeUserRestoreSignerAdminRequiredForbidden` (300941) |
| Cannot restore self | Signer ID in target list | `CodeUserRestoreSelfTargetForbidden` (300942) |

**RestorePostFetch** (targets fetched — DB access):

| Rule | Condition | Code |
|------|-----------|------|
| Cannot restore SuperAdmin target | Any target DB role is `RoleSuperAdmin` | `CodeUserRestoreSuperAdminTargetForbidden` (300943) |
| Cannot restore live users | Any target `DeletedAt == nil` | `CodeUserRestoreNotTrashedForbidden` (300944) |

Admin **can** restore trashed Admin peers — restore is an undo, not an
escalation. SuperAdmin restore is unconditionally blocked (one-SuperAdmin
invariant).

### 11.5 Service Flow

All inside a single `s.uow.Execute` — mirrors the `UpdateRole` and `Update`
patterns (fetch + policy + DB + chain in one TX):

```
Restore(ctx, ids)
  ├─ policy.RestorePreFetch(ctx, ids)
  └─ s.uow.Execute:
       ├─ FindByIds (unscoped — sees trashed users)
       ├─ policy.RestorePostFetch(ctx, targets)
       ├─ DB: UPDATE users SET deleted_at = NULL WHERE id IN (?)  ← single batch UPDATE
       └─ chain: syncBlockchainRoles(targets, CodeUserRestoreBlockchainSyncFailed)
            └─ Re-syncs the preserved DB role to chain (undoes RoleNone revocation)
```

If chain sync fails, the DB `deleted_at` nulling rolls back (single TX).

Added to `UserService` interface as `Restore(ctx, ids []string) ([]domain.User, int64, error)`.

### 11.6 Handler + Route

`UserHandler` interface gets `Restore(c *gin.Context)`. Handler parses
`UserRestoreRequest`, validates via Ozzo, calls `userSvc.Restore(ctx, ids)`.

Route registration (inserted after `DELETE /batch` in router.go):
```go
users.PUT("/batch/restore", gin.HandlerFunc(p.AdminRoleMiddleware), p.UserHandler.Restore)
```

Response: `responder.Send(c, CodeUserRestoreSuccess, responseUsers)` with
restored user list + count.

### 11.7 Registration

6 new domain codes → 6 entries each in:

| File | Entries |
|------|---------|
| `domain/codes.go` | 6 constants (300900, 300941–300945) |
| `mapper.go` CodeToMessageKey | 6 keys |
| `mapper.go` HttpCodes | 6 entries (200 + 5×403 + 500) |
| `mapper_test.go` allDomainCodes | 6 entries |
| `locales/en.json` | 6 entries |
| `locales/id.json` | 6 entries |

### 11.8 Locale Messages

| Key | en |
|-----|----|
| `success_users_restore` | `Users restored successfully.` |
| `error_users_restore_signer_admin_required` | `Only administrators can restore users.` |
| `error_users_restore_self_target_forbidden` | `You cannot restore your own account.` |
| `error_users_restore_super_admin_target_forbidden` | `SuperAdmin users cannot be restored via this endpoint.` |
| `error_users_restore_not_trashed_forbidden` | `All target users must be previously deleted.` |
| `error_users_restore_blockchain_sync_failed` | `Failed to sync user restore to blockchain.` |

### 11.9 Tests

| Test | Expect |
|------|--------|
| `TestUserService_Restore_Success` | Trashed Holder restored; `deleted_at` nulled; chain role restored |
| `TestUserService_Restore_AdminRestoresAdminPeer` | 200 — Admin can restore trashed Admin peer |
| `TestUserService_Restore_SuperAdminTarget` | 403 — `CodeUserRestoreSuperAdminTargetForbidden` |
| `TestUserService_Restore_LiveTarget` | 403 — `CodeUserRestoreNotTrashedForbidden` |
| `TestUserService_Restore_Self` | 403 — `CodeUserRestoreSelfTargetForbidden` |
| `TestUserService_Restore_BelowAdmin` | 403 — `CodeUserRestoreSignerAdminRequiredForbidden` |
| `TestUserService_Restore_BlockchainFailure` | 500 — `CodeUserRestoreBlockchainSyncFailed`; DB rolls back |
| `TestUserRestoreRequest_Validate_EmptyIDs` | Ozzo validation fails on empty `ids` slice |
| `TestUserRestoreRequest_Validate_Valid` | Ozzo validation passes |

### 11.10 Doc Sync

| File | Edit |
|------|------|
| `ROLES.md` | Route table: new `PUT /api/users/batch/restore` row. Capability matrix: restore row (Admin+). Policy rules: RestorePreFetch + RestorePostFetch tables. Denied ops: restore SuperAdmin, restore live, restore self rows |
| `AGENTS.md` | Routes table: new restore row. Two-Method Policy Splits: add RestorePreFetch/RestorePostFetch. Restore success/error codes in domain code listing. Soft Delete section: note restore unsets `deleted_at` |
| Plan doc | B5 fully done (Delete UoW unified + Restore implemented). Section updated |

### 11.11 Updated Files Summary (combined with Sections 1–10)

| # | File | Change |
|---|------|--------|
| 1 | `domain/credential.go` | +3 interface signature changes (`*Query` param) |
| 2 | `domain/codes.go` | +3 verify verdict codes (400410–400412) + 6 restore codes (300900, 300941–300945) |
| 3 | `feature/credential/gorm_credential_repository.go` | Update 3 repo methods to accept + use `*Query` |
| 4 | `feature/credential/credential_service.go` | Update callers; add party-disabled check at 3 Verify branches |
| 5 | `feature/user/user_service.go` | Unify `Delete` into single UoW; add `Restore` method |
| 6 | `feature/user/user_policy.go` | + `RestorePreFetch` + `RestorePostFetch` |
| 7 | `feature/user/user_handler.go` | + `Restore` handler + request DTO |
| 8 | `feature/user/user_request.go` | + `UserRestoreRequest` DTO |
| 9 | `infrastructure/http/router.go` | + `PUT /batch/restore` route |
| 10 | `infrastructure/testutil/mocks/` | Update `MockCredentialRepository`; `MockUserPolicy` |
| 11 | `infrastructure/http/responder/mapper.go` | +18 entries (9 msg keys + 9 http codes) |
| 12 | `infrastructure/http/responder/mapper_test.go` | +9 `allDomainCodes` |
| 13 | `locales/en.json` | +9 entries |
| 14 | `locales/id.json` | +9 entries |
| 15 | `AGENTS.md` | Cache design, verify codes, restore endpoint, routes table |
| 16 | `ROLES.md` | Verify row update, restore route + policy rules + matrix |
| 17 | `docs/superpowers/plans/2026-06-09-roles-capability-revisions.md` | B4 done, B5 done, B6 removed
