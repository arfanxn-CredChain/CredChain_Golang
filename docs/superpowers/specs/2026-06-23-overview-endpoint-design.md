# Overview Endpoint Design

> `GET /api/overview` — role-conditional dashboard endpoint returning aggregated credential counts, user counts, recent activity, and on-chain status. Response shape differs by caller role (Holder vs Issuer+).

## Route

| Method | Path | Auth |
|--------|------|------|
| GET | `/api/overview` | Authenticated (no role gate — service checks role) |

Date filter via `filters` query param using the standard `..` BETWEEN syntax:

```
GET /api/overview?filters=date..2026-01-01,2026-06-30
```

The key `date` is a custom overview filter name (not a real DB column). The handler recognizes it and applies the parsed range to `issued_at`, `revoked_at`, and `created_at` internally. No filter = all-time.

## Response Shape

### Issuer+ (Issuer, Admin, SuperAdmin)

```json
{
  "code": 100100,
  "data": {
    "credential_counts": {
      "total": 500,
      "active": 450,
      "revoked": 40,
      "pending": 10,
      "failed": 3
    },
    "user_counts": {
      "total": 150,
      "holder": 120,
      "issuer": 20,
      "admin": 8,
      "super_admin": 1,
      "active": 145,
      "trashed": 5
    },
    "recents": {
      "active_credentials": [
        {
          "id": "01J...",
          "name": "Bachelor's Degree",
          "holder": {"id": "01J...", "name": "John", "email": "john@example.com"},
          "issuer": {"id": "01J...", "name": "UI", "email": "admin@ui.ac.id"},
          "issued_at": "2026-06-20T10:00:00Z"
        }
      ],
      "revoked_credentials": [
        {
          "id": "01J...",
          "name": "Diploma",
          "holder": {"id": "01J...", "name": "Jane", "email": "jane@example.com"},
          "revoker": {"id": "01J...", "name": "Admin", "email": "admin@example.com"},
          "issued_at": "2026-04-01T00:00:00Z",
          "revoked_at": "2026-06-19T08:00:00Z"
        }
      ],
      "stored_users": [
        {"id": "01J...", "name": "Jane", "email": "jane@example.com", "role": "holder", "created_at": "2026-06-18T00:00:00Z"}
      ]
    },
    "chain_details": {
      "authority_contract": "0x9A...",
      "registry_contract": "0x8B...",
      "last_block": 12345678
    }
  }
}
```

### Holder

Keys `user_counts`, `chain_details`, and `recents.stored_users` are absent via `omitempty` on nil pointers — never serialized as `null`.

```json
{
  "code": 100100,
  "data": {
    "credential_counts": {
      "total": 12,
      "active": 10,
      "revoked": 2,
      "pending": 1,
      "failed": 0
    },
    "recents": {
      "active_credentials": [
        {
          "id": "01J...",
          "name": "Diploma",
          "issuer": {"id": "01J...", "name": "UI", "email": "admin@ui.ac.id"},
          "issued_at": "2026-06-15T00:00:00Z"
        }
      ],
      "revoked_credentials": [
        {
          "id": "01J...",
          "name": "Certificate",
          "revoker": {"id": "01J...", "name": "Admin", "email": "admin@example.com"},
          "issued_at": "2026-01-01T00:00:00Z",
          "revoked_at": "2026-06-14T00:00:00Z"
        }
      ]
    }
  }
}
```

## Credential Count Definitions

| Field | Definition |
|-------|-----------|
| `total` | COUNT(*) — all credentials |
| `active` | `revoked_at IS NULL AND extract_status IN ('pending', 'succeeded')` — not revoked, extraction is pending or succeeded |
| `revoked` | `revoked_at IS NOT NULL AND extract_status IN ('pending', 'succeeded')` — revoked, extraction is pending or succeeded |
| `pending` | `extract_status = 'pending'` — extraction not yet run by River worker |
| `failed` | `extract_status = 'failed'` — extraction failed, retryable via ReExtract |

Credentials with `extract_status = 'failed'` are excluded from `active` and `revoked` — they represent a broken state that needs operator attention. They still appear in `total` and `failed`.

### Count Membership Matrix

Each credential falls into exactly the following counts based on its `revoked_at` and `extract_status`:

| Credential state | `total` | `active` | `revoked` | `pending` | `failed` |
|---|---|---|---|---|---|
| Not revoked, extraction pending | ✓ | ✓ | | ✓ | |
| Not revoked, extraction succeeded | ✓ | ✓ | | | |
| Not revoked, extraction failed | ✓ | | | | ✓ |
| Revoked, extraction pending | ✓ | | ✓ | ✓ | |
| Revoked, extraction succeeded | ✓ | | ✓ | | |
| Revoked, extraction failed | ✓ | | | | ✓ |

Key invariants:
- `active + revoked + failed = total` (failed acts as a third bucket alongside active and revoked)
- `pending` counts active AND revoked credentials with `extract_status = 'pending'` (not filtered by `revoked_at`)
- `failed` counts active AND revoked credentials with `extract_status = 'failed'` (not filtered by `revoked_at`)

## User Count Definitions

| Field | Definition |
|-------|-----------|
| `total` | COUNT(*) — all users |
| `holder` / `issuer` / `admin` / `super_admin` | COUNT(*) grouped by `role` |
| `active` | `deleted_at IS NULL` — live users |
| `trashed` | `deleted_at IS NOT NULL` — soft-deleted users |

Note: `holder + issuer + admin + super_admin = total`, and `active + trashed = total` — two different breakdowns of the same set.

## Recents (5 items each, sorted DESC)

| Field | Issuer+ | Holder |
|-------|---------|--------|
| `active_credentials` | System-wide, `revoked_at IS NULL`, ordered by `issued_at DESC` | Own only, scoped to `holder_user_id` |
| `revoked_credentials` | System-wide, `revoked_at IS NOT NULL`, ordered by `revoked_at DESC` | Own only, scoped to `holder_user_id` |
| `stored_users` | Most recent `created_at DESC` | absent |

All credential recents are fetched via GORM with `Preload("Holder").Preload("Issuer")` (and `Preload("Revoker")` for revocations). No raw JOINs — relations are loaded through GORM's standard Preload mechanism and mapped via existing `response.Credential` / `response.User` DTOs.

Holder recents exclude `stored_users`. `recents.active_credentials` excludes `holder` from the Preload for holders (the holder is the authenticated user — redundant).

## Response Codes

| Code | Constant | HTTP | Meaning |
|------|----------|------|---------|
| 100100 | `CodeOverviewSuccess` | 200 | Overview fetched successfully |
| 100150 | `CodeOverviewInternal` | 500 | Internal error (DB error) |

Category `10` (system), feature BB `01` (overview), CC `00` (success) / `50` (internal error).

## Data Queries (2 round-trips, no N+1)

### Query 1: Aggregate counts (single SQL with scalar subqueries)

Issuer+:
```sql
SELECT
  (SELECT COUNT(*) FROM credentials WHERE issued_at BETWEEN $1 AND $2) AS total,
  (SELECT COUNT(*) FROM credentials
   WHERE revoked_at IS NULL AND extract_status IN ('pending', 'succeeded')
   AND issued_at BETWEEN $1 AND $2) AS active,
  (SELECT COUNT(*) FROM credentials
   WHERE revoked_at IS NOT NULL AND extract_status IN ('pending', 'succeeded')
   AND issued_at BETWEEN $1 AND $2) AS revoked,
  (SELECT COUNT(*) FROM credentials
   WHERE extract_status = 'pending' AND issued_at BETWEEN $1 AND $2) AS pending,
  (SELECT COUNT(*) FROM credentials
   WHERE extract_status = 'failed' AND issued_at BETWEEN $1 AND $2) AS failed,
  (SELECT COUNT(*) FROM users WHERE created_at BETWEEN $1 AND $2) AS user_total,
  (SELECT COUNT(*) FROM users WHERE role = 'holder' AND created_at BETWEEN $1 AND $2) AS user_holder,
  (SELECT COUNT(*) FROM users WHERE role = 'issuer' AND created_at BETWEEN $1 AND $2) AS user_issuer,
  (SELECT COUNT(*) FROM users WHERE role = 'admin' AND created_at BETWEEN $1 AND $2) AS user_admin,
  (SELECT COUNT(*) FROM users WHERE role = 'super_admin' AND created_at BETWEEN $1 AND $2) AS user_super_admin,
  (SELECT COUNT(*) FROM users WHERE deleted_at IS NULL AND created_at BETWEEN $1 AND $2) AS user_active,
  (SELECT COUNT(*) FROM users WHERE deleted_at IS NOT NULL AND created_at BETWEEN $1 AND $2) AS user_trashed;
```

Holder variant: same credential subqueries but scoped to `holder_user_id = $3`. User subqueries replaced with constant zeroes — user counts not needed for holders.

### Query 2: Recents (GORM Preload, 3 separate DB calls)

Issuer+ active_credentials:
```go
db.Where("revoked_at IS NULL").
   Where("issued_at BETWEEN ? AND ?", dateFrom, dateTo).
   Preload("Holder").Preload("Issuer").
   Order("issued_at DESC").Limit(5).Find(&creds)
```

Issuer+ revoked_credentials:
```go
db.Where("revoked_at IS NOT NULL").
   Where("revoked_at BETWEEN ? AND ?", dateFrom, dateTo).
   Preload("Holder").Preload("Revoker").
   Order("revoked_at DESC").Limit(5).Find(&creds)
```

Issuer+ stored_users:
```go
db.Where("created_at BETWEEN ? AND ?", dateFrom, dateTo).
   Order("created_at DESC").Limit(5).Find(&users)
```

Holder: same credential queries with additional `Where("holder_user_id = ?", authUserID)`. Holder's `active_credentials` skips `Preload("Holder")` since the holder is the authenticated user. `stored_users` not queried.

### On-Chain Info

- `authority_contract` / `registry_contract`: read from `*config.Config` (in memory, no RPC)
- `last_block`: single RPC call via `chain.Client.BlockNumber(ctx)` — Issuer+ only, called once. On RPC failure, set `last_block = 0` and return 200 (non-fatal).

## Response DTO Design

Recents use existing response package DTOs (`response.Credential` and `response.User`). Only the overview envelope and count types are new.

```go
type Overview struct {
    CredentialCounts *OverviewCredentialCounts `json:"credential_counts,omitempty"`
    UserCounts       *OverviewUserCounts       `json:"user_counts,omitempty"`
    Recents          *OverviewRecents          `json:"recents,omitempty"`
    ChainDetails     *OverviewChainDetails     `json:"chain_details,omitempty"`
}

type OverviewCredentialCounts struct {
    Total   int `json:"total"`
    Active  int `json:"active"`
    Revoked int `json:"revoked"`
    Pending int `json:"pending"`
    Failed  int `json:"failed"`
}

type OverviewUserCounts struct {
    Total      int `json:"total"`
    Holder     int `json:"holder"`
    Issuer     int `json:"issuer"`
    Admin      int `json:"admin"`
    SuperAdmin int `json:"super_admin"`
    Active     int `json:"active"`
    Trashed    int `json:"trashed"`
}

type OverviewRecents struct {
    ActiveCredentials  []response.Credential `json:"active_credentials"`
    RevokedCredentials []response.Credential `json:"revoked_credentials"`
    StoredUsers        []response.User       `json:"stored_users"`
}

type OverviewChainDetails struct {
    AuthorityContract string `json:"authority_contract"`
    RegistryContract  string `json:"registry_contract"`
    LastBlock         uint64 `json:"last_block"`
}
```

`response.Credential` already includes nested `holder`, `issuer`, `revoker` (each `*response.User` with `omitempty`). `response.User` already matches the user shape used across the API.

## File Structure

```
feature/overview/
  overview_handler.go              → HTTP handler, parses auth user + query filters, delegates to service
  overview_service.go              → Business logic: role detection, query orchestration, assembles Overview
  overview_response.go             → Overview, OverviewCredentialCounts, OverviewUserCounts, OverviewRecents, OverviewChainDetails
  gorm_overview_repository.go      → OverviewRepository interface + GORM implementation (aggregate counts + recents with Preload)
  overview_handler_test.go         → Handler tests (mock service)
  overview_service_test.go         → Service tests (mock repo)
  gorm_overview_repository_test.go → Repository tests (in-memory SQLite)
```

Plus changes to shared files:
- `domain/codes.go` — add `CodeOverviewSuccess = 100100`, `CodeOverviewInternal = 100150`
- `infrastructure/http/responder/mapper.go` — register codes in `CodeToMessageKey`, `HttpCodes`, and `allDomainCodes`
- `infrastructure/http/router.go` — register `GET /api/overview` route with `AuthMiddleware`
- `locales/en.json`, `locales/id.json` — add message keys for `CodeOverviewSuccess` and `CodeOverviewInternal`
- `cmd/server.go` — register FX providers for `OverviewHandler`, `OverviewService`, and `gormOverviewRepository`
- `AGENTS.md` — add `GET /api/overview` row to the API Routes table (Authenticated, no role gate, role-conditional response)
- `ROLES.md` — add `GET /api/overview` to the API Route Authorization table (Authenticated, any role); add overview capabilities to the Per-Role Capability Matrix
- `CREDENTIAL.md` — add overview endpoint to the API Routes section (overview credential_counts + recents reference)
- `CredChain_postman_collection.json` — add overview endpoint with Issuer+ and Holder response examples

## Role Detection Logic

In `overviewService.GetOverview(ctx, query)`:

```go
if authUser.Role.Rank() >= domain.RoleIssuer.Rank():
    return issuerOverview  // all fields populated
else:
    return holderOverview  // user_counts, chain_details, recents.stored_users = nil
```

## Date Filter Behavior

The user provides one date range via the `filters` query param using the standard `..` BETWEEN syntax with the `date` key:

```
?filters=date..2026-01-01,2026-06-30
```

`date` is a custom filter name recognized only by the overview endpoint — not a real DB column. The handler parses it via the existing `QueryRequest` parser (which applies the BETWEEN operator to extract `dateFrom` and `dateTo` values). The service then applies that **same range** uniformly to all time-based fields in all queries:

| Query | Columns filtered |
|-------|-----------------|
| Credential counts (`total`, `active`, `revoked`) | `issued_at BETWEEN` |
| Credential counts (`pending`, `failed`) | `issued_at BETWEEN` |
| User counts (all) | `created_at BETWEEN` |
| Recents `active_credentials` | `issued_at BETWEEN` |
| Recents `revoked_credentials` | `revoked_at BETWEEN` |
| Recents `stored_users` | `created_at BETWEEN` |

One input, applied to `issued_at`, `revoked_at`, and `created_at` where relevant. When no filter is provided, defaults to `0001-01-01` to `9999-12-31` (all-time).

## Fallback Behavior

- **Chain RPC fails for `last_block`:** set `last_block = 0`, return 200 success — don't fail the whole overview over a flaky RPC
- **DB fails on any query:** return `CodeOverviewInternal` (500)
- **Empty state (no credentials, no users):** return 200 with all counts = 0, empty recents arrays — valid state for fresh deployment
