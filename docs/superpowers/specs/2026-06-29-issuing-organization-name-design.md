# Issuing Organization Name & Meta Endpoint — Design Spec

## Overview

Credentials display "Issued by `<institution>`" instead of just the issuer's employee
name. CredChain deployments are single-tenant (one organization per instance), so the
organization name comes from an environment variable: `ISSUING_ORGANIZATION_NAME`.

A public `/api/meta` endpoint exposes deployment-wide metadata (org name, contract
addresses, chain info) so the frontend can fetch it once per session.

## Config

**New field** in `config.Config`:

```go
IssuingOrganizationName *string
```

Mapped from env var `ISSUING_ORGANIZATION_NAME`. Nil if not set.

**Validation** (inside `NewConfig`): fatal if nil or empty string. Same fatal-pattern
as `JWTSecret`, `WalletEncryptionKey`, and `FileEncryptionKey`.

## Chain Client

**New method** on `*chain.Client`:

```go
func (c *Client) ChainID(ctx context.Context) (uint64, error)
```

Wraps `c.EthClient.ChainID(ctx)`. Not cached — calls RPC on every invocation.
Follows the existing `BlockNumber(ctx context.Context) (uint64, error)` pattern
(no caching there either).

## Response Codes

| Constant | Value | HTTP |
|---|---|---|
| `CodeMetaSuccess` | `100200` | 200 |
| `CodeMetaInternal` | `100250` | 500 |

Category `10` (system), sub-category `02` (meta). Success CC = 00, error CC = 50.

## Response DTO

**File:** `infrastructure/http/response/meta.go`

```go
type MetaResponse struct {
    IssuingOrganizationName string `json:"issuing_organization_name"`
    AuthorityContract       string `json:"authority_contract"`
    RegistryContract        string `json:"registry_contract"`
    ChainID                 uint64 `json:"chain_id"`
    LastBlock               uint64 `json:"last_block"`
}
```

No `FromDomainXxx` factory — there is no domain entity backing this DTO.
Fields are populated directly from `*config.Config` and `*chain.Client`.

## Service

**File:** `feature/meta/meta_service.go`

```go
type MetaService interface {
    Get(ctx context.Context) (*response.MetaResponse, error)
}
```

Dependencies:
- `*config.Config` — `IssuingOrganizationName`, `AuthorityContract`, `RegistryContract`
- `*chain.Client`   — `BlockNumber(ctx)`, `ChainID(ctx)`

No database, no repository, no UnitOfWork, no policy. Read-only, thin service.

## Handler

**File:** `feature/meta/meta_handler.go`

```go
type MetaHandler interface {
    GetMeta(c *gin.Context)
}
```

Route: `GET /api/meta` — public, no auth middleware.

Handler flow:
1. Call `service.Get(c.Request.Context())`
2. On success: `responder.Send(c, domain.CodeMetaSuccess, meta)`
3. On error: `responder.SendError(c, err)` — service wraps errors via
   `domain.NewError(domain.CodeMetaInternal, ...)`

No request DTO needed (no body or query parameters).

## Router Registration

Public route on the `/api` group (alongside `/health`, `/auth/google`, etc.):

```go
api.GET("/meta", p.MetaHandler.GetMeta)
```

## DI Wiring

Add to `fx.Provide` in `cmd/server.go`:

```go
meta.NewMetaService,
meta.NewMetaHandler,
```

Add `MetaHandler` to `RouteParams` in `infrastructure/http/router.go`.

## Locale Messages

`locales/en.json`:
```json
{
    "success_meta": "Meta information retrieved successfully.",
    "error_meta_internal": "Failed to retrieve meta information."
}
```

`locales/id.json`:
```json
{
    "success_meta": "Informasi meta berhasil diambil.",
    "error_meta_internal": "Gagal mengambil informasi meta."
}
```

## Mapper Entries

`infrastructure/http/responder/mapper.go`:

```go
CodeToMessageKey: {
    domain.CodeMetaSuccess:  "success_meta",
    domain.CodeMetaInternal: "error_meta_internal",
}
HttpCodes: {
    domain.CodeMetaSuccess:  200,
    domain.CodeMetaInternal: 500,
}
```

## File Manifest

| Action | File |
|--------|------|
| Create | `feature/meta/meta_handler.go` |
| Create | `feature/meta/meta_handler_test.go` |
| Create | `feature/meta/meta_service.go` |
| Create | `feature/meta/meta_service_test.go` |
| Create | `infrastructure/http/response/meta.go` |
| Create | `infrastructure/http/response/meta_test.go` |
| Modify | `config/config.go` |
| Modify | `config/config_test.go` |
| Modify | `domain/codes.go` |
| Modify | `infrastructure/http/router.go` |
| Modify | `infrastructure/http/responder/mapper.go` |
| Modify | `cmd/server.go` |
| Modify | `locales/en.json` |
| Modify | `locales/id.json` |
| Modify | `.env.example` |
| Modify | `infrastructure/chain/client.go` |

## Non-Goals

- **Not** modifying `FromDomainCredential` factory
- **No** caching on the backend (`/api/meta` is cheap — 2 RPC calls)
- **No** database changes, repositories, or DB schema
- Frontend caching is out of scope (in `CredChain_React` repo)
