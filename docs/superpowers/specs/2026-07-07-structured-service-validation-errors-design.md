# Structured Service-Level Validation Errors + All-or-Nothing Batch Operations

Date: 2026-07-07
Status: In design (not yet implemented)

## Context

Batch operations (`POST /api/users/batch`, `POST /api/credentials/batch/issue`) have two
categories of error:

1. **Ozzo validation** — structural/format errors caught at the handler by the request DTO's
   `Validate()` method. These already produce rich `{"errors": {"users.0.email": ["..."]}}`
   responses via `responder.SendValidationError`.

2. **Service-layer business logic** — duplicate emails, policy violations, wallet generation
   failures, chain sync failures. These currently return a flat domain error code + message
   with no per-item granularity. For a batch of 50 users, the caller learns WHAT went wrong
   but not WHERE.

Additionally, `Credential Issue` currently supports partial success (some items committed while
others return field errors), which creates an inconsistent contract across batch endpoints.
The goal is **all endpoints all-or-nothing** with **rich per-item errors** on failure.

## Design Decision

**Service methods return Ozzo's `validation.Errors` for business-validation failures.**

This approach:

- Reuses the existing `SendValidationError` pipeline — zero new responder or domain types
- Produces identical response shapes to Ozzo structural validation (one contract for all validation)
- Keeps domain layer pure — no HTTP-concern types leak into `domain/`
- Simplifies `Credential Issue` by removing the `map[string][]string` return and `SendPartial`

## Response Shape (identical to current Ozzo validation errors)

```json
{
  "code": 100040,
  "message": "Validation failed",
  "errors": {
    "users.0.email": ["Email already exists in this batch"],
    "users.2.email": ["Email already registered by another user"],
    "credentials.1.holder_user_id": ["Holder user not found"],
    "credentials.3.file": ["Duplicate file hash"]
  }
}
```

## 1. Concept: Ozzo `validation.Errors` at the Service Level

Ozzo's `validation.Errors` is `map[string]error` where keys are field paths and values are
`validation.Error` instances (with `Code()` and `Params()`). The handler type-asserts the
error returned by the service:

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

This dispatch pattern is the only handler-side change. `SendValidationError` already knows
how to render `validation.Errors` → `{"code":100040, "errors":{...}}`.

## 2. User Store — Structured Per-Item Errors

### 2.1 Service signature (unchanged)

`Store(ctx, users ...domain.User) ([]domain.User, error)`

The `error` return is now either:
- `validation.Errors` — business validation failures (structured, per-item)
- `*domain.Error` — domain-level failures (chain sync, unexpected DB errors)

### 2.2 Which errors become `validation.Errors` vs `*domain.Error`

Only **input-driven** business rules produce `validation.Errors` — errors the client can fix
by changing their input. Server-side operations (wallet generation, encryption, storage, chain
sync, policy violations) stay as `*domain.Error`. This preserves a clear semantic boundary:
`validation.Errors` = "your input is wrong", `*domain.Error` = "something broke on our side".

**`storeValidateEmails`** — duplicate email detection (input-driven → `validation.Errors`):

```go
func (s *userService) storeValidateEmails(ctx context.Context, users []domain.User) validation.Errors {
    verrs := validation.Errors{}

    // In-batch duplicates
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

    // DB duplicates
    if len(verrs) == 0 {
        emails := lo.Map(users, func(u domain.User, _ int) string { return u.Email })
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

### 2.3 Store orchestrates — collects input errors, returns atomically

```go
func (s *userService) Store(ctx context.Context, users ...domain.User) ([]domain.User, error) {
    if err := s.policy.Store(ctx, users...); err != nil {
        return nil, err
    }

    verrs := s.storeValidateEmails(ctx, users)
    if len(verrs) > 0 {
        return nil, verrs
    }

    // Wallet generation failures stay *domain.Error (server-side, not input validation).
    if err := s.storeGenerateWallets(users); err != nil {
        return nil, err
    }

    // Chain sync failure returns *domain.Error as before (not validation.Errors)
    return s.storeUsersAndSyncBlockchainRoles(ctx, users)
}
```

### 2.4 Handler dispatch

```go
func (h *userHandler) Store(c *gin.Context) {
    var req UserStoreRequest
    if err := c.ShouldBindJSON(&req); err != nil { ... }
    if err := req.Validate(); err != nil {
        responder.SendValidationError(c, err)
        return
    }
    domainUsers := req.ToDomain()
    created, err := h.userSvc.Store(c.Request.Context(), domainUsers...)
    if err != nil {
        if verrs, ok := err.(validation.Errors); ok {
            responder.SendValidationError(c, verrs)
            return
        }
        c.Error(err)
        responder.SendError(c, err)
        return
    }
    // ... success response ...
}
```

### 2.5 Policy errors stay as `*domain.Error`

Policy violations (`CodeUserStoreSuperAdminForbidden`, `CodeUserStoreAdminCreateAdminForbidden`)
are not per-item validation — they're auth/rule violations that invalidate the entire request.
These remain `*domain.Error` and go through `responder.SendError`.

## 3. Credential Issue — Remove Partial Success, Use `validation.Errors`

### 3.1 Service signature change

```go
// Before
Issue(ctx, items []CredentialIssuance) ([]domain.Credential, map[string][]string, error)

// After
Issue(ctx, items []CredentialIssuance) ([]domain.Credential, error)
```

Removes the `fieldErrs map[string][]string` return. All errors are collected into
`validation.Errors` and returned as the `error` value.

### 3.2 Issue orchestrates — all-or-nothing

```go
func (s *credentialService) Issue(ctx context.Context, items []CredentialIssuance) ([]domain.Credential, error) {
    authUser := httpContext.MustGetUser(ctx)

    if err := s.policy.IssuePreFetch(ctx, ...); err != nil {
        return nil, err
    }

    verrs := s.issueValidate(ctx, items)
    if len(verrs) > 0 {
        return nil, verrs
    }

    // ... UoW: Store → chain sync → token update → extract enqueue ...
}
```

### 3.3 `issueValidate` collects all per-item errors

```go
func (s *credentialService) issueValidate(
    ctx context.Context,
    items []CredentialIssuance,
) validation.Errors {
    verrs := validation.Errors{}

    // Holder existence check (single batch IN query)
    holderIDs := lo.Uniq(lo.Map(items, func(it CredentialIssuance, _ int) string { return it.HolderUserID }))
    holders, _ := s.userRepo.FindByIds(ctx, holderIDs...)
    holderSet := lo.SliceToMap(holders, func(h domain.User) (string, bool) { return h.Id, true })

    // Hash computation + on-chain status check
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

    // In-batch duplicate detection
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

### 3.4 Handler dispatch

```go
func (h *credentialHandler) Issue(c *gin.Context) {
    // ... parse multipart, build items ...

    created, err := h.credSvc.Issue(c.Request.Context(), serviceItems)
    if err != nil {
        if verrs, ok := err.(validation.Errors); ok {
            responder.SendValidationError(c, verrs)
            return
        }
        c.Error(err)
        responder.SendError(c, err)
        return
    }
    // ... success: always CodeCredentialIssueSuccess ...
    out := mapCredentialsToResponse(created)
    responder.Send(c, domain.CodeCredentialIssueSuccess, out)
}
```

## 4. Remove `SendPartial`

### 4.1 Files

- `infrastructure/http/responder/responder.go` — delete `SendPartial`. Keep `resolveMessage` (used by exported `ResolveMessage`).

### 4.2 Callers

- `feature/credential/credential_handler.go:265` — the only caller. Replaced by the
  `validation.Errors` dispatch + `Send` pattern from Section 3.4.

## 5. Other Batch Endpoints

| Endpoint | All-or-nothing? | Change needed? |
|----------|-----------------|----------------|
| `PUT /api/users/batch/role` | Already all-or-nothing | Add `validation.Errors` dispatch in handler |
| `DELETE /api/users/batch` | Already all-or-nothing | Add `validation.Errors` dispatch if per-item errors added |
| `PUT /api/users/batch/restore` | Already all-or-nothing | Add `validation.Errors` dispatch if per-item errors added |
| `POST /api/credentials/batch/revoke` | Already all-or-nothing | Add `validation.Errors` dispatch if per-item errors added |
| `POST /api/credentials/batch/reextract` | Already all-or-nothing | Add `validation.Errors` dispatch if per-item errors added |

These endpoints currently return flat `*domain.Error` for all failure paths. Adding per-item
granularity to them is a follow-on task — this spec focuses on `Store` and `Issue` as the
two endpoints with the most complex validation logic. The handler dispatch pattern
(asserting `validation.Errors`) is applied uniformly so future enhancements are drop-in.

## 6. New i18n Keys

### 6.1 Ozzo validation codes → locale keys

Service-layer errors use Ozzo's `validation.Error.Code()` as the i18n message key.
New keys to add:

| Ozzo Code | en |
|-----------|----|
| `validation_store_email_duplicate_batch` | `Email "{{.field}}" appears multiple times in this batch.` |
| `validation_store_email_duplicate_db` | `Email "{{.field}}" is already registered.` |
| `validation_issue_holder_not_found` | `Holder user "{{.field}}" was not found.` |
| `validation_issue_duplicate_file_hash` | `A credential with this file hash already exists.` |

### 6.2 Ozzo Codes Without `{{.field}}` Auto-Injection

`SendValidationError` only auto-injects `field` for Ozzo-internal codes (e.g.,
`validation_required`). Service-created `validation.Error` instances must include
`"field": label` in their params explicitly. The methods in Sections 2.2 and 3.3 do this.

For auto-injected keys (`field`, `min`, `max`, `values`), the `locale_keys_test.go` parser
traces `WithMetadata` calls in Go source. For service-layer validation, the params are
set inline via `validation.NewError(code, params)` — not through `domain.WithMetadata`.

**Two options for the locale key checker:**

1. **Extend AST scanner** — teach `locale_keys_test.go` to also scan for `validation.NewError`
   calls (detect `map[string]interface{}` literals as sources of template data keys).
2. **Manual verification** — add the 4 keys to locale files. The existing check that every
   `CodeToMessageKey` value exists in both locale files still runs. The extra `{{.field}}`
   placeholder is manually verified. Option B is simpler and sufficient since `field` is
   already an auto-injected key in the existing `SendValidationError` logic (line 125).

Recommendation: Option B (manual). The `SendValidationError` handler already auto-injects
`"field"` at line 125 for all `validation.Error` instances regardless of Ozzo-internal
vs custom codes.

## 7. Code Registration

| What | File |
|------|------|
| No new domain codes | — reuses `CodeSystemValidation` (100040) |
| 4 locale entries (en) | `locales/en.json` |
| 4 locale entries (id) | `locales/id.json` |

## 8. Files Touched (implementation-order view)

| # | File | Change |
|---|------|--------|
| 1 | `feature/user/user_service.go` | `Store`: collect per-item `validation.Errors` from `storeValidateEmails`. `storeValidateEmails` returns `validation.Errors`. `storeGenerateWallets` unchanged (server-side → `*domain.Error`). |
| 2 | `feature/user/user_handler.go` | `Store` handler: type-assert `validation.Errors` dispatch |
| 3 | `feature/credential/credential_service.go` | `Issue`: remove `fieldErrs` return, accumulate `validation.Errors` in new `issueValidate` helper. Remove per-item file URI tracking and `claimedHash` map. `cleanupOrphanFiles` stays for UoW-failure path (chain rollback → orphan files on disk). |
| 4 | `feature/credential/credential_handler.go` | `Issue` handler: remove `SendPartial`, add `validation.Errors` dispatch. Always `CodeCredentialIssueSuccess` on success. |
| 5 | `infrastructure/http/responder/responder.go` | Delete `SendPartial`. Keep `resolveMessage` (used by exported `ResolveMessage`). |
| 6 | `locales/en.json` | +4 entries |
| 7 | `locales/id.json` | +4 entries |
| 8 | `feature/credential/credential_service_test.go` | Update `Issue` tests to match new signature. Remove partial-success test cases. Add validation error test cases. |
| 9 | `feature/user/user_service_test.go` | Update `Store` tests to exercise new `validation.Errors` paths. |
| 10 | `feature/credential/credential_handler_test.go` | Update `Issue` handler tests. |
| 11 | `AGENTS.md` | Update response envelope section to note `validation.Errors` at service level. Remove `SendPartial` references. |
| 12 | `CREDENTIAL.md` | Remove "Partial success" paragraph from Issue section. Remove `CodeCredentialIssueFailed` (400240) — Issue is always all-or-nothing. |

## 9. Remove `CodeCredentialIssueFailed` (400240)

With all-or-nothing Issue, `CodeCredentialIssueFailed` is dead. It was only used when
`successCount == 0 || len(fieldErrs) > 0`. Both conditions are impossible in the new model:

- If validation errors exist → `validation.Errors` returned → `100040`
- If all items valid → `CodeCredentialIssueSuccess` (400200)
- If chain/DB failure → `*domain.Error` with the relevant code (400244, 400245, etc.)

Remove `400240` from:

| File | Change |
|------|--------|
| `domain/codes.go` | Delete `CodeCredentialIssueFailed` |
| `mapper.go` CodeToMessageKey | Delete entry |
| `mapper.go` HttpCodes | Delete entry |
| `mapper_test.go` allDomainCodes | Delete entry |
| `locales/en.json` | Delete `error_credential_issue_failed` |
| `locales/id.json` | Delete entry |

## 10. Verification

```bash
cd CredChain_Golang
go test ./... && go vet ./... && gofmt -l .
```

`locale_keys_test.go` catches missing i18n keys. `mapper_test.go` catches missing code
registrations. `gofmt -l .` must produce zero output.

## 11. Doc Sync

| File | Edit |
|------|------|
| `AGENTS.md` | Remove `SendPartial` references. Add service-level `validation.Errors` dispatch pattern. Remove partial-success mention. |
| `CREDENTIAL.md` | Remove partial-success paragraph (§Batch Flows → Issue). Remove `CodeCredentialIssueFailed` from error codes table. Update Issue flow to reflect all-or-nothing. |
