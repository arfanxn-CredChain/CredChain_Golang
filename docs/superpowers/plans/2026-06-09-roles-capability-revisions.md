# Roles Capability Revisions

Date: 2026-06-09
Status: Complete

## Context

Revisions to the CredChain role/authorization system following review of
`ROLES.md`. Splits into two parts:

- **Part A — execute now:** SuperAdmin self-edit carve-out + email guard,
  capability tweaks (Issuer+ reads users, public verify), and doc sync.
- **Part B — deferred TODOs:** holder self-credentials endpoints, credential
  revoke ownership rule, on-chain credential revocation on user delete, and a
  Unit-of-Work consistency cleanup.

> **Reverted 2026-06-09:** Part A originally included a role-storage change
> (soft-delete sets `role = NULL`, `model.User.Role` as `*string`, persisted
> `'none'` enum). This was fully reverted — the `role` column stays
> `NOT NULL` with values `super_admin/admin/issuer/holder` only, `Role` stays
> `string`, and `Delete` is a pure soft-delete. The on-chain `RoleNone`
> revocation in `userService.Delete` is unchanged; `AuthMiddleware` rejecting
> trashed users at 401 remains the guard against a preserved DB role.

Verification command (run after Part A):

```bash
cd CredChain_Golang
go test ./... && go vet ./... && gofmt -l .   # gofmt must print nothing
```

---

## Part A — Execute Now

### A1. New domain code `CodeUserUpdateSelfEmailForbidden = 300847`

Slot `300847` is the next free code in the `30084x` user-update block
(`300845` is `CodeUserUpdateBlockchainSyncFailed`, `300846` is
`CodeUserUpdateTrashedForbidden`).

**Files:**

1. `domain/codes.go` — add `CodeUserUpdateSelfEmailForbidden = 300847`
   (in the `30084x` user-update block, after `CodeUserUpdateTrashedForbidden`).

2. `infrastructure/http/responder/mapper.go`
   - `CodeToMessageKey`: add `CodeUserUpdateSelfEmailForbidden ->
     "error_users_update_self_email_forbidden"`.
   - `HttpCodes`: add `CodeUserUpdateSelfEmailForbidden -> 403`.

3. `infrastructure/http/responder/mapper_test.go`
   - Add `CodeUserUpdateSelfEmailForbidden` to the hardcoded `allDomainCodes`
     list.

4. `locales/en.json` + `locales/id.json` — add the message key:
   - en: `You cannot change your own email via batch update. Use the self email endpoint instead.`
   - id: `Anda tidak dapat mengubah email sendiri melalui pembaruan batch. Gunakan endpoint email mandiri.`

**Verification:** `locale_keys_test.go` (key exists in both locales) and
`mapper_test.go` (code registered) pass.

### A2. SuperAdmin self-edit carve-out + email guard

**File:** `feature/user/user_policy.go`

1. `UpdatePreFetch`: skip the `CodeUserUpdateSelfForbidden` self-target block
   when the signer is SuperAdmin. Admin and below remain blocked from
   self-targeting via batch.

   ```go
   for _, u := range users {
       if u.Id == authUser.Id && authUser.Role != domain.RoleSuperAdmin {
           return domain.NewError(domain.CodeUserUpdateSelfForbidden,
               domain.WithMetadata("user_id", authUser.Id))
       }
   }
   ```

2. `UpdatePostFetch`: the "Cannot update SuperAdmin" block exempts self-target
   (`t.Id != authUser.Id`) so SuperAdmin can self-edit; add a guard — if any
   update targets the signer's own ID AND carries a non-empty `Email`, return
   `CodeUserUpdateSelfEmailForbidden`. Role-agnostic.

**Verification:** policy unit tests —
- SuperAdmin self-update (no email) -> allowed
- SuperAdmin self-update (with email) -> `CodeUserUpdateSelfEmailForbidden`
- Admin self-update -> `CodeUserUpdateSelfForbidden` (unchanged)

### A3. Router capability tweaks

**File:** `infrastructure/http/router.go`

1. `GET /api/users` and `GET /api/users/:id`: swap `AdminRoleMiddleware` ->
   `IssuerRoleMiddleware` (Issuer+ may read/list users, read-only).
2. `POST /api/credentials/verify`: move out of the `secure` group (drop
   `AuthMiddleware`) and drop `IssuerRoleMiddleware` — becomes a fully public
   endpoint for external verifiers (HR, employers).
   - Place under the public `/api` group (no auth) alongside health.
   - Confirm the verify handler does NOT call `httpContext.MustGetUser`
     directly (it only reaches user via the policy, which A4 makes a no-op).

**Verification:** manual route inspection; existing handler tests still pass.

### A4. VerifyPreFetch -> no-op

**File:** `feature/credential/credential_policy.go`

Change `VerifyPreFetch` to return `nil` unconditionally (public verifier
endpoint). Keep the method on the interface so the service call site is
unchanged.

```go
func (p *credentialPolicy) VerifyPreFetch(ctx context.Context) error {
    return nil // public endpoint — external verifiers need no auth
}
```

**Verification:** credential policy tests updated; verify still returns a
verdict without an authenticated user in context.

### A5. Sync `AGENTS.md`

**File:** `CredChain_Golang/AGENTS.md`

- API Routes table: `GET /api/users` and `/api/users/:id` -> Issuer+ (read-only).
- API Routes table: `POST /api/credentials/verify` -> None (public).
- Self-profile lockdown note: SuperAdmin may self-update profile via
  `PUT /api/users/batch` (except email, which requires `/users/self/email`).

### A6. Verify

```bash
cd CredChain_Golang
go test ./... && go vet ./... && gofmt -l .
```

All tests green; `go vet` clean; `gofmt -l .` prints nothing.

---

## Part B — Deferred TODOs

B1 and B2 (holder self-credentials endpoints) were **implemented 2026-06-09**
(handler `SelfPaginate`/`SelfFind` + service + routes + tests). B3–B5 were
implemented 2026-06-10 (credential revocation on delete, UoW unification for
Delete, Restore endpoint). B6 (verify audit-log) was never needed and has been
removed from this plan.

### B1. `GET /api/users/self/credentials` — Holder lists own credentials *(DONE)*

**Objective:** Allow any authenticated user (regardless of role) to list their
own credentials. Frontend uses this on the Holder dashboard / profile page.

**Requirements:**
- Route: `GET /api/users/self/credentials` (Authenticated, any role).
- Scoped: returns ONLY credentials where `holder_user_id == auth_user.id`.
- Supports the same pagination + query DSL as `GET /api/credentials` (page,
  per_page, filters on credential-level fields like `name`, `issued_at`,
  `revoked_at`).
- Response DTO: uses existing `response.Credential` or a slim variant.
- Policy: no additional role check (authenticated is sufficient; you always
  own your own credentials).

**Implementation sketch:**
1. `credential_handler.go` — new method `SelfPaginate` similar to `Paginate`
   but injects `holder_user_id = authUser.Id` filter.
2. `credential_service.go` — new method `FindSelf(ctx, authUser, query)` that
   wraps `credRepo.Find` with the holder filter.
3. `credential_policy.go` — optional: add `SelfPaginatePreFetch(ctx)` no-op
   (keep interface symmetry).
4. `infrastructure/http/router.go` — register `GET /users/self/credentials`.
5. Tests: happy path (auth user sees own creds), empty (no creds), error
   (repo failure).

**Dependencies:** None — uses existing `CredentialRepository.Find`.

### B2. `GET /api/users/self/credentials/:id` — Holder fetches own credential *(DONE)*

**Objective:** Fetch a single credential, scoped to the auth user.

**Requirements:**
- Route: `GET /api/users/self/credentials/:id` (Authenticated, any role).
- 404 if credential does not exist OR `holder_user_id != auth_user.id`.
- Response DTO: `response.Credential`.

**Implementation sketch:**
1. `credential_handler.go` — new method `SelfFind` similar to `Find` but
   checks ownership after fetch, returns 404 (not 403) to avoid leaking which
   IDs exist.
2. `credential_service.go` — new method `FindSelfOne(ctx, authUser, id)` that
   calls `credRepo.Find(id)` (or `Get(id)`) and asserts ownership.
3. `infrastructure/http/router.go` — register `GET /users/self/credentials/:id`.
4. Tests: exists+owned (200), exists+not-owned (404), not-exists (404).

**Dependencies:** B1 (uses same concepts).

### B4. User credential revocation on delete *(DONE 2026-06-10)*

**Objective:** When a user is soft-deleted (via `DELETE /api/users/batch`), all
their credentials should also be revoked on-chain via CredentialRegistry and
marked revoked in the DB.

**Implemented:** `feature/user/user_service.go:Delete` now calls `credSvc.RevokeByHolder`
in the same UoW transaction, batch-revoking all credentials for the deleted
holder on-chain via CredentialRegistry and marking them revoked in the DB.
The multi-line TODO has been removed. Party-disabled verify verdicts
(400410–400412) are re-checked live against the users table on every verify
call to detect trashed holders/issuers.

### B5. UoW pattern unification *(DONE 2026-06-10)*

**Objective:** `UpdateRole` and `Delete` originally used two `uow.Execute()`
calls (read phase then write phase) while `Update` uses a single call. Unify
to a single UoW to eliminate the small race window between read and write.

**Status:**
- `UpdateRole` — **DONE 2026-06-09**: collapsed into one `s.uow.Execute` call.
  Fetch + post-fetch policy + DB update + chain sync now run inside a single
  transaction. Chain-sync failure rolls back the DB write atomically.
  `updateRoleAndSyncBlockchainRoles` helper removed; `updateRoleValidateAndPrepare`
  retained (still called from inside the unified UoW).
- `Delete` — **DONE 2026-06-10**: collapsed into a single `s.uow.Execute` call
  (fetch + validate + soft-delete DB + credential revocation + chain role
  revocation in one transaction).
- **Restore** — **DONE 2026-06-10**: `PUT /api/users/batch/restore` clears
  `deleted_at` and re-syncs the preserved DB role to chain via
  `syncBlockchainRoles`. Policy: `RestorePreFetch` blocks below-Admin and
  self-targeting; `RestorePostFetch` blocks SuperAdmin targets and non-trashed
  targets.




