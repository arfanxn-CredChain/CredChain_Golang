# Credential Search & Filter Backend Changes — Design Spec

**Date:** 2026-06-21
**Scope:** `feature/credential/gorm_credential_repository.go`
**Status:** Awaiting implementation

---

## 1. Goal

Extend credential search to cover credential identity fields and all related user fields (holder, issuer, revoker). Add `extract_status` to the filter allowlist so the frontend can filter by extraction state.

---

## 2. Change 1: Extend Search Columns

**File:** `feature/credential/gorm_credential_repository.go`
**Lines:** 127-138 (the `HasSearch()` block)

### Current search fields (6):

| Column | Join |
|---|---|
| `credentials.name` | none |
| `credentials.meta` (CAST AS TEXT) | none |
| `holder.name` | `LEFT JOIN users AS holder` |
| `holder.email` | `LEFT JOIN users AS holder` |
| `holder.number` | `LEFT JOIN users AS holder` |
| `holder.phone_number` | `LEFT JOIN users AS holder` |

### Target search fields (14):

| Column | Join needed |
|---|---|
| `credentials.name` | none |
| `credentials.meta` (CAST AS TEXT) | none |
| `credentials.id` | none |
| `credentials.token_id` | none |
| `credentials.file_hash` | none |
| `holder.name` | `LEFT JOIN users AS holder` |
| `holder.email` | `LEFT JOIN users AS holder` |
| `holder.number` | `LEFT JOIN users AS holder` |
| `issuer.name` | `LEFT JOIN users AS issuer` |
| `issuer.email` | `LEFT JOIN users AS issuer` |
| `issuer.number` | `LEFT JOIN users AS issuer` |
| `revoker.name` | `LEFT JOIN users AS revoker` |
| `revoker.email` | `LEFT JOIN users AS revoker` |
| `revoker.number` | `LEFT JOIN users AS revoker` |

### Implementation

The existing `needsHolderJoin` helper (line ~83-89) determines when to add the holder LEFT JOIN. Extend it (or add a separate `needsIssuerJoin`/`needsRevokerJoin`) so that when `HasSearch()` is true, all three user joins are added:

```go
// Pseudocode for the revised search block:
if query.HasSearch() {
    needle := "%" + query.Search + "%"
    db = db.Where(
        "LOWER(credentials.name) LIKE LOWER(?) OR "+
            "LOWER(CAST(credentials.meta AS TEXT)) LIKE LOWER(?) OR "+
            "LOWER(credentials.id) LIKE LOWER(?) OR "+
            "LOWER(credentials.token_id) LIKE LOWER(?) OR "+
            "LOWER(credentials.file_hash) LIKE LOWER(?) OR "+
            "LOWER(holder.name) LIKE LOWER(?) OR "+
            "LOWER(holder.email) LIKE LOWER(?) OR "+
            "LOWER(holder.number) LIKE LOWER(?) OR "+
            "LOWER(issuer.name) LIKE LOWER(?) OR "+
            "LOWER(issuer.email) LIKE LOWER(?) OR "+
            "LOWER(issuer.number) LIKE LOWER(?) OR "+
            "LOWER(revoker.name) LIKE LOWER(?) OR "+
            "LOWER(revoker.email) LIKE LOWER(?) OR "+
            "LOWER(revoker.number) LIKE LOWER(?)",
        needle, needle, needle, needle, needle, // 5 for credential columns
        needle, needle, needle,                   // 3 for holder
        needle, needle, needle,                   // 3 for issuer
        needle, needle, needle,                   // 3 for revoker
    )
}
```

**Total `needle` placeholders:** 14

### Join logic

The `needsHolderJoin`, `needsIssuerJoin`, and `needsRevokerJoin` helpers should each return true when:
- `query.HasSearch()` is true (all three joins needed for search)
- OR sorts reference columns from that user table (existing behavior)

The joins themselves already exist in the codebase:

```go
// holder join (existing)
db = db.Joins("LEFT JOIN users AS holder ON holder.id = credentials.holder_user_id")

// issuer join (new — add when needsIssuerJoin)
db = db.Joins("LEFT JOIN users AS issuer ON issuer.id = credentials.issuer_user_id")

// revoker join (new — add when needsRevokerJoin)
db = db.Joins("LEFT JOIN users AS revoker ON revoker.id = credentials.revoker_user_id")
```

---

## 3. Change 2: Add `extract_status` to Filter Allowlist

**File:** `feature/credential/gorm_credential_repository.go`
**Lines:** 35-40 (the column allowlist)

### Current allowlist:

```
name, issued_at, revoked_at, holder_user_id
```

### Target allowlist:

```
name, issued_at, revoked_at, holder_user_id, extract_status
```

`extract_status` is a string column on the `credentials` table with values `"pending"`, `"succeeded"`, `"failed"`. No join needed. The existing `ApplyFilters` helper in `infrastructure/database/gorm/helpers.go` already handles string equality (`=`) and other operators — no new operator logic needed.

### Frontend filter usage

The frontend will send these filters:

| Filter label | Backend filters sent |
|---|---|
| All | (none) |
| Active | `revoked_at_`, `extract_status!=failed` |
| Revoked | `revoked_at!_`, `extract_status!=failed` |
| Pending | `extract_status=pending` |
| Failed | `extract_status=failed` |

---

## 4. Out of Scope

- No new sort columns needed (existing 7 cover all requirements)
- No new batch operation limits (already capped at 100)
- No changes to `credential_service.go` or `credential_handler.go`
- No changes to response shapes

---

## 5. Verification

After changes, run from the Go repo:

```bash
go test ./... && go vet ./... && gofmt -l .
```

Expected: all pass, last command produces zero output.

---

## 6. Self-Review

- Placeholder scan: none
- Internal consistency: filter allowlist change enables all 5 frontend filter options. Search join addition covers all 14 requested columns.
- Scope: single file (`gorm_credential_repository.go`), two isolated changes.
- Ambiguity: join order (LEFT JOIN) is safe — rows without issuer/revoker still return with NULLs.
