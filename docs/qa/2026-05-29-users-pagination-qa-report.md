# QA Report: Users Pagination, Filtering, and Sorting

**Date**: 2026-05-29  
**Component**: `GET /api/users` endpoint (user listing)  
**Test Method**: HTTP requests via Postman/Newman against running CredChain-Golang server  

## Executive Summary

All pagination features work correctly. Filtering and sorting were partially implemented (present in query parser/validator but missing repository logic). As of this test run, filtering and sorting support has been added to the `gorm_user_repository.Get` method via code changes made during this session.

**Result**: Pagination ✅, Filtering ✅ (now implemented), Sorting ✅ (now implemented with expanded column support)

## Detailed Test Results

### Pagination

Pagination implementation in the repository was already complete and correct.

- **Tested**: A1–A10 (10 test cases)
- **Result**: PASS

All pagination controls behave as specified:
- Default page=1, limit=10
- Custom page and limit values respected
- Page navigation (first, prev, next, last) URLs built correctly
- Edge cases (empty results, past-last page, invalid values) handled with appropriate validation errors
- `total` count reflects filtered/search results when applicable

No issues found in pagination logic.

### Filtering

Prior to changes, filter parameters were accepted by the HTTP query validator but silently ignored by the repository (comment in code: "filters, includes, groups skipped for now").

**Implementation Detail**: Added `applyUserFilters` helper that maps `domainQuery.Filter` operators to GORM Where clauses with column allowlisting for security (prevents filtering on sensitive columns like `wallet_address`, `encrypted_wallet_private_key`, `deleted_at`, `meta`). Pattern matching uses dialect-agnostic `LOWER(col) LIKE LOWER(?)` for case-insensitivity.

- **Tested**: B1–B12 (12 test cases)
- **Result**: PASS (after implementation)

All filter types now work correctly:
- Equality (`=`), inequality (`!=`), comparison (`>`, `<`, `>=`, `<=`)
- Pattern matching (`~`, `~*`, `!~`, `!~*`) with case-insensitive semantics
- Set membership (`$`, `!$`) and range (`..`, `!..`)
- Null checks (`_`, `!_`)
- Combined with search and pagination
- Invalid syntax rejected by validator (422)
- Disallowed columns silently dropped (security)

No issues found after implementation.

### Sorting

Prior to changes, only `created_at` and `name` columns were supported for sorting; other columns were silently ignored.

**Implementation Detail**: Expanded `allowedSortColumns` whitelist to include `id`, `name`, `email`, `role`, `created_at`, `updated_at`. Disallowed columns are now silently dropped (consistent with filtering approach).

- **Tested**: C1–C12 (12 test cases)
- **Result**: PASS (after implementation)

All sort combinations now work correctly:
- Single-column sorts (ASC/DESC)
- Multi-column sorts (e.g., `-created_at,name`)
- Sort + search combinations
- Sort + pagination
- Disallowed columns silently ignored (no crash, no sort applied)
- Invalid syntax rejected by validator (422)

No issues found after implementation.

## Issues Encountered & Fixes

### Filtering
- **Issue**: Filter parameters parsed and validated but not applied in repository (code comment: "filters, includes, groups skipped for now")
- **Fix**: Added `applyUserFilters` helper and integrated into `Get` method
- **Files Changed**: `feature/user/gorm_user_repository.go`

### Sorting
- **Issue**: Only `created_at` and `name` columns supported; others silently ignored without indication
- **Fix**: Expanded whitelist to include all safe, indexable user columns (`id`, `name`, `email`, `role`, `created_at`, `updated_at`)
- **Files Changed**: `feature/user/gorm_user_repository.go`

### Test Coverage
- Added 15 new unit tests covering filter and sort functionality in `gorm_user_repository_test.go`

## Verification Commands Run

```bash
# Build verification
go build ./...

# Test suite
go test ./feature/user/... -v

# Linting (project has no formal lint but these are clean)
go vet ./...
gofmt -l .  # no output = clean

# Specific filter/sort test run
go test ./feature/user -v -run "TestGormUserRepository_Get_Filter|TestGormUserRepository_Get_Sort"
```

All commands passed with no errors or formatting issues.

## Postman Collection Updates

Response examples for `GET /api/users` in `CredChain_postman_collection.json` have been updated to reflect:
- Successful paginated response with filter/query parameters
- Validation error responses for invalid filter/sort syntax
- Empty results case
- Updated `total`, `from`, `to`, `last_page` values based on applied filters

## Conclusion

All user-facing pagination, filtering, and sorting features on the `GET /api/users` endpoint are now fully functional and tested. The implementation is secure (column allowlisting), dialect-agnostic (works with Postgres + SQLite), and maintains backward compatibility.

No further action required.

---

## Follow-up: trashed users pagination (added 2026-05-29)

After the initial pagination/filter/sort work, the user requested support for paginating trashed (soft-deleted) users via the same endpoint. The chosen approach is **3a (literal): all `Find*` repository methods are now unscoped**, with explicit guards in the auth middleware, policy layer, service idempotency, and `init-super-admin` to keep mutation paths and recovery flows correct.

### Scope summary

- **Repository (`gormUserRepository`)**:
  - `Get` auto-unscopes when its query references `deleted_at` in any filter or sort (via `referencesDeletedAt`). Otherwise default scope applies.
  - `Find`, `FindByIds`, `FindByEmails`, `FindByRole` are unconditionally `Unscoped()` — they return trashed users so admins can inspect, list, and recover them.
  - `deleted_at` added to `allowedFilterColumns` and `allowedSortColumns`.

- **Filter operator vocabulary for `deleted_at`**:
  - `deleted_at!_` → only trashed users (IS NOT NULL)
  - `deleted_at_` → only live users (IS NULL, explicit)
  - `deleted_at..2026-01-01 and 2026-12-31` → trashed within date range
  - `-deleted_at` → sort desc; auto-unscopes; mixes live (NULL) and trashed
  - `deleted_at>=2026-05-01` → trashed since date

- **Mutation-path guards** (preventing accidental writes against trashed rows):
  - `AuthMiddleware` rejects trashed users with `CodeAuthUnauthorized` (401) — even with a valid JWT.
  - `userPolicy.UpdatePostFetch` returns `CodeUserUpdateTrashedForbidden` (300846, 403) when any target carries `DeletedAt != nil`.
  - `userPolicy.UpdateRolePostFetch` returns `CodeUserRoleTrashedForbidden` (300547, 403) under the same condition.
  - `userService.Delete` is idempotent on already-trashed targets: filters them out of the on-chain `RoleNone` revocation set; `deleted_count` reflects only freshly-deleted rows.

- **Recovery path**:
  - `cmd/init_super_admin` filters out trashed entries from the `FindByRole(SuperAdmin)` existence check, so a system whose previous SuperAdmin was soft-deleted can re-initialize.

### New domain codes

| Code | Constant | HTTP | Locale key |
|---|---|---|---|
| 300547 | `CodeUserRoleTrashedForbidden` | 403 | `error_role_trashed_forbidden` |
| 300846 | `CodeUserUpdateTrashedForbidden` | 403 | `error_users_update_trashed_forbidden` |

Locale messages include a `{{.user_id}}` placeholder so admins can see which trashed user blocked a batch operation.

### Test coverage added

- `gorm_user_repository_test.go`: 7 new tests for trashed-user pagination (filter `deleted_at!_`, `deleted_at_`, BETWEEN range, sort `-deleted_at`, AND-combined with `role`, pagination count consistency, default-scope regression guard). 4 existing soft-delete tests renamed from `HidesFrom*` to `*ReturnsTrashed` and inverted to assert that trashed users are now returned with `DeletedAt` populated.
- `user_policy_test.go`: 2 new tests for `UpdatePostFetch` and `UpdateRolePostFetch` rejecting trashed targets.
- `user_service_test.go`: 2 new tests — `Delete_AlreadyTrashed_SkipsChainSync` and `Delete_MixedLiveAndTrashed_OnlyLiveSyncsToChain` — using `MockAuthorityService` to assert the chain sync is invoked only for live targets.
- `infrastructure/http/middleware/auth_test.go`: `TestAuthMiddleware_TrashedUser_401` confirms a trashed user with a valid JWT receives 401.

### Postman updates

- `List Users`: `deleted_at` added to filter/sort allowed-columns docs; 2 new disabled query examples (`deleted_at!_` and `deleted_at..a and b`); 1 new response example "300100 OK — only trashed".
- `Find User by ID`: description notes trashed-user visibility; 1 new response example "300100 OK — trashed user" with non-null `deleted_at`.
- `Batch Update Users`: 1 new "300846 Trashed Forbidden" 403 example.
- `Batch Update Roles`: 1 new "300547 Trashed Forbidden" 403 example.
- `Batch Delete Users`: description updated to note idempotency on already-trashed rows.

### Files changed (this follow-up)

```
domain/codes.go                                   ← +CodeUserRoleTrashedForbidden, +CodeUserUpdateTrashedForbidden
infrastructure/http/responder/mapper.go           ← +CodeToMessageKey + HttpCodes entries
infrastructure/http/responder/mapper_test.go      ← +allDomainCodes entries
locales/en.json + locales/id.json                 ← +2 messages each (with {{.user_id}})
feature/user/gorm_user_repository.go              ← +deleted_at allowlist, +referencesDeletedAt, Find* Unscoped
feature/user/gorm_user_repository_test.go         ← rename 4 + 7 new + 1 fix (TestGormUserRepository_Delete)
feature/user/user_policy.go                       ← +trashed guards in UpdatePostFetch, UpdateRolePostFetch
feature/user/user_policy_test.go                  ← +2 trashed-guard tests
feature/user/user_service.go                      ← Delete idempotency on trashed (skip chain sync)
feature/user/user_service_test.go                 ← +2 service tests
infrastructure/http/middleware/auth.go            ← +trashed-user 401 reject
infrastructure/http/middleware/auth_test.go       ← +TestAuthMiddleware_TrashedUser_401
cmd/init_super_admin.go                           ← filter trashed from FindByRole result
CredChain_postman_collection.json                 ← deleted_at examples + trashed responses + 403 errors
AGENTS.md                                         ← refined Soft delete bullet + trashed-pagination bullet
```

All tests pass; `go vet` and `gofmt -l .` clean; Postman JSON revalidated.