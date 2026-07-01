# Issuing Organization Name & Meta Endpoint — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `ISSUING_ORGANIZATION_NAME` config, `ChainID` method on chain client, and a public `GET /api/meta` endpoint returning deployment metadata.

**Architecture:** Config gets a new `*string` field validated at startup. `*chain.Client` gains `ChainID(ctx)`. A new thin `feature/meta` package with a service (reads config + chain) and handler (Gin endpoint). No DB, no repository, no policy.

**Tech Stack:** Go 1.25.1, gin-gonic/gin, uber/fx, go-ethereum, stretchr/testify

## Global Constraints

- All Config struct fields are pointers (`*string`); `nil` = not provided; validated in `NewConfig`
- Response codes use 6-digit `AABBCC` format in `domain/codes.go`; must be added to `CodeToMessageKey`, `HttpCodes`, and `allDomainCodes`
- Every new `CodeToMessageKey` value must exist in both `locales/en.json` and `locales/id.json`
- Feature packages use exported interface + unexported struct + `fx.In` params + exported factory pattern
- Handler methods: bind → validate → call service → `responder.Send`/`SendError`
- Service methods return domain errors via `domain.NewError(code, ...)`
- FX providers listed linearly in `cmd/server.go`; route handler types added to `RouteParams` in `router.go`
- Module path is `CredChain_Golang` (with underscore)

---

### Task 1: Response DTO

**Files:**
- Create: `infrastructure/http/response/meta.go`
- Create: `infrastructure/http/response/meta_test.go`

**Interfaces:**
- Produces: `type Meta struct { IssuingOrganizationName string; AuthorityContract string; RegistryContract string; ChainID uint64; LastBlock uint64 }` (consumed by Task 3, 4, 5)

- [ ] **Step 1: Write the DTO**

```go
// infrastructure/http/response/meta.go
package response

type Meta struct {
	IssuingOrganizationName string `json:"issuing_organization_name"`
	AuthorityContract       string `json:"authority_contract"`
	RegistryContract        string `json:"registry_contract"`
	ChainID                 uint64 `json:"chain_id"`
	LastBlock               uint64 `json:"last_block"`
}
```

- [ ] **Step 2: Write the DTO test**

```go
// infrastructure/http/response/meta_test.go
package response

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMeta_JSONMarshal(t *testing.T) {
	m := Meta{
		IssuingOrganizationName: "University of Indonesia",
		AuthorityContract:       "0xAAA",
		RegistryContract:        "0xBBB",
		ChainID:                 137,
		LastBlock:               42000000,
	}
	b, err := json.Marshal(m)
	require.NoError(t, err)

	var out map[string]interface{}
	require.NoError(t, json.Unmarshal(b, &out))
	assert.Equal(t, "University of Indonesia", out["issuing_organization_name"])
	assert.Equal(t, "0xAAA", out["authority_contract"])
	assert.Equal(t, "0xBBB", out["registry_contract"])
	assert.Equal(t, float64(137), out["chain_id"])
	assert.Equal(t, float64(42000000), out["last_block"])
}
```

- [ ] **Step 3: Run DTO test**

Run: `go test ./infrastructure/http/response/... -v -run TestMeta`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add infrastructure/http/response/meta.go infrastructure/http/response/meta_test.go
git commit -m "feat: add Meta DTO for /api/meta endpoint"
```

---

### Task 2: Domain Codes

**Files:**
- Modify: `domain/codes.go` (add constants)
- Modify: `infrastructure/http/responder/mapper_test.go` (add to allDomainCodes)

**Interfaces:**
- Produces: `CodeMetaSuccess = 100200`, `CodeMetaInternal = 100250` (consumed by Task 5, 6)

- [ ] **Step 1: Add code constants**

Add after the `CodeOverviewInternal` line in `domain/codes.go`:

```go
	// ── Meta (10) ─────────────────────────────────────────────────────────────
	CodeMetaSuccess  = 100200
	CodeMetaInternal = 100250
```

- [ ] **Step 2: Add to allDomainCodes in mapper_test.go**

Add after `domain.CodeOverviewInternal,` in the `allDomainCodes` slice:

```go
	// Meta
	domain.CodeMetaSuccess,
	domain.CodeMetaInternal,
```

- [ ] **Step 3: Run tests to verify codes compile**

Run: `go test ./domain/... ./infrastructure/http/responder/... -v`
Expected: PASS (mapper tests will fail on missing message keys — expected, fixed in Task 6)

- [ ] **Step 4: Commit**

```bash
git add domain/codes.go infrastructure/http/responder/mapper_test.go
git commit -m "feat: add CodeMetaSuccess (100200) and CodeMetaInternal (100250)"
```

---

### Task 3: Config

**Files:**
- Modify: `config/config.go` (add field + validation)
- Modify: `config/config_test.go` (add test)
- Modify: `.env.example` (add env var comment)

**Interfaces:**
- Produces: `Config.IssuingOrganizationName *string` (consumed by Task 5)

- [ ] **Step 1: Add field to Config struct**

Add after the `RegistryContract` line in `config/config.go`:

```go
	IssuingOrganizationName           *string
```

- [ ] **Step 2: Load from env in NewConfig**

Add after `RegistryContract: getEnv("REGISTRY_CONTRACT", nil),`:

```go
		IssuingOrganizationName:           getEnv("ISSUING_ORGANIZATION_NAME", nil),
```

- [ ] **Step 3: Add fatal validation**

Add after the `FileEncryptionKey` validation block (before the CookieSameSite check):

```go
	if cfg.IssuingOrganizationName == nil || *cfg.IssuingOrganizationName == "" {
		return nil, fmt.Errorf("issuing_organization_name is required")
	}
```

- [ ] **Step 4: Write config test**

Add to `config/config_test.go`:

```go
func TestNewConfig_MissingIssuingOrganizationName_ReturnsError(t *testing.T) {
	t.Setenv("JWT_SECRET", "my-secret")
	t.Setenv("WALLET_ENCRYPTION_KEY", "my-wallet-key-exactly-32-chars-x")
	t.Setenv("FILE_ENCRYPTION_KEY", "my-file-key-exactly-32-chars-xxx")
	t.Setenv("ISSUING_ORGANIZATION_NAME", "")
	_, err := NewConfig(".env.nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "issuing_organization_name is required")
}

func TestNewConfig_IssuingOrganizationName_Set(t *testing.T) {
	t.Setenv("JWT_SECRET", "my-secret")
	t.Setenv("WALLET_ENCRYPTION_KEY", "my-wallet-key-exactly-32-chars-x")
	t.Setenv("FILE_ENCRYPTION_KEY", "my-file-key-exactly-32-chars-xxx")
	t.Setenv("ISSUING_ORGANIZATION_NAME", "University of Indonesia")
	cfg, err := NewConfig(".env.nonexistent")
	assert.NoError(t, err)
	assert.Equal(t, "University of Indonesia", *cfg.IssuingOrganizationName)
}
```

- [ ] **Step 5: Run config tests**

Run: `go test ./config/... -v`
Expected: PASS (2 new tests pass)

- [ ] **Step 6: Add to .env.example**

Add after `REGISTRY_CONTRACT=0x...` line:

```
# Organization identity (displayed on credentials as "Issued by ...")
ISSUING_ORGANIZATION_NAME=
```

- [ ] **Step 7: Commit**

```bash
git add config/config.go config/config_test.go .env.example
git commit -m "feat: add ISSUING_ORGANIZATION_NAME config with fatal validation"
```

---

### Task 4: Chain Client ChainID Method

**Files:**
- Modify: `infrastructure/chain/client.go` (add ChainID method)

**Interfaces:**
- Produces: `func (c *Client) ChainID(ctx context.Context) (uint64, error)` (consumed by Task 5)

- [ ] **Step 1: Add ChainID method**

Add after the `BlockNumber` method in `infrastructure/chain/client.go`:

```go
func (c *Client) ChainID(ctx context.Context) (uint64, error) {
	chainID, err := c.EthClient.ChainID(ctx)
	if err != nil {
		return 0, err
	}
	return chainID.Uint64(), nil
}
```

Requires adding `"math/big"` to imports. But `ChainID` returns `*big.Int` — check if `math/big` is already imported (it's transitively available via go-ethereum types). Actually need to explicitly import it.

Add `"math/big"` to the import block.

- [ ] **Step 2: Verify compilation**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add infrastructure/chain/client.go
git commit -m "feat: add ChainID method to chain.Client"
```

---

### Task 5: Meta Service

**Files:**
- Create: `feature/meta/meta_service.go`
- Create: `feature/meta/meta_service_test.go`

**Interfaces:**
- Consumes: `*config.Config` (from Task 3), `*chain.Client` (from Task 4), `*response.Meta` (from Task 1), `domain.CodeMetaSuccess`, `domain.CodeMetaInternal` (from Task 2)
- Produces: `MetaService interface { Get(ctx context.Context) (*response.Meta, error) }` (consumed by Task 6)

- [ ] **Step 1: Write the failing service test**

```go
// feature/meta/meta_service_test.go
package meta

import (
	"context"
	"errors"
	"testing"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/http/response"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockChainClient struct {
	mock.Mock
}

func (m *mockChainClient) BlockNumber(ctx context.Context) (uint64, error) {
	args := m.Called(ctx)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *mockChainClient) ChainID(ctx context.Context) (uint64, error) {
	args := m.Called(ctx)
	return args.Get(0).(uint64), args.Error(1)
}

func TestMetaService_Get_Success(t *testing.T) {
	chainMock := new(mockChainClient)
	cfg := &config.Config{
		IssuingOrganizationName: ptrStr("University of Indonesia"),
		AuthorityContract:       ptrStr("0xAAA"),
		RegistryContract:        ptrStr("0xBBB"),
	}
	svc := newMetaService(cfg, chainMock)

	chainMock.On("BlockNumber", mock.Anything).Return(uint64(42000000), nil)
	chainMock.On("ChainID", mock.Anything).Return(uint64(137), nil)

	result, err := svc.Get(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "University of Indonesia", result.IssuingOrganizationName)
	assert.Equal(t, "0xAAA", result.AuthorityContract)
	assert.Equal(t, "0xBBB", result.RegistryContract)
	assert.Equal(t, uint64(137), result.ChainID)
	assert.Equal(t, uint64(42000000), result.LastBlock)
	chainMock.AssertExpectations(t)
}

func TestMetaService_Get_BlockNumberError(t *testing.T) {
	chainMock := new(mockChainClient)
	cfg := &config.Config{
		IssuingOrganizationName: ptrStr("University of Indonesia"),
		AuthorityContract:       ptrStr("0xAAA"),
		RegistryContract:        ptrStr("0xBBB"),
	}
	svc := newMetaService(cfg, chainMock)

	chainMock.On("BlockNumber", mock.Anything).Return(uint64(0), errors.New("rpc down"))

	_, err := svc.Get(context.Background())
	require.Error(t, err)

	var dErr *domain.Error
	assert.True(t, errors.As(err, &dErr))
	assert.Equal(t, domain.CodeMetaInternal, dErr.Code)
}

func TestMetaService_Get_ChainIDError(t *testing.T) {
	chainMock := new(mockChainClient)
	cfg := &config.Config{
		IssuingOrganizationName: ptrStr("University of Indonesia"),
		AuthorityContract:       ptrStr("0xAAA"),
		RegistryContract:        ptrStr("0xBBB"),
	}
	svc := newMetaService(cfg, chainMock)

	chainMock.On("BlockNumber", mock.Anything).Return(uint64(42000000), nil)
	chainMock.On("ChainID", mock.Anything).Return(uint64(0), errors.New("rpc down"))

	_, err := svc.Get(context.Background())
	require.Error(t, err)

	var dErr *domain.Error
	assert.True(t, errors.As(err, &dErr))
	assert.Equal(t, domain.CodeMetaInternal, dErr.Code)
}

func ptrStr(s string) *string { return &s }
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./feature/meta/... -v`
Expected: compilation error — `newMetaService` not defined

- [ ] **Step 3: Write the service implementation**

```go
// feature/meta/meta_service.go
package meta

import (
	"context"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/chain"
	"CredChain_Golang/infrastructure/http/response"

	"go.uber.org/fx"
)

type chainClient interface {
	BlockNumber(ctx context.Context) (uint64, error)
	ChainID(ctx context.Context) (uint64, error)
}

type MetaService interface {
	Get(ctx context.Context) (*response.Meta, error)
}

type metaService struct {
	cfg     *config.Config
	chain   chainClient
}

type MetaServiceParams struct {
	fx.In
	Config      *config.Config
	ChainClient *chain.Client
}

func NewMetaService(p MetaServiceParams) MetaService {
	return newMetaService(p.Config, p.ChainClient)
}

func newMetaService(cfg *config.Config, chain chainClient) *metaService {
	return &metaService{cfg: cfg, chain: chain}
}

func (s *metaService) Get(ctx context.Context) (*response.Meta, error) {
	lastBlock, err := s.chain.BlockNumber(ctx)
	if err != nil {
		return nil, domain.NewError(domain.CodeMetaInternal, domain.WithError(err))
	}

	chainID, err := s.chain.ChainID(ctx)
	if err != nil {
		return nil, domain.NewError(domain.CodeMetaInternal, domain.WithError(err))
	}

	return &response.Meta{
		IssuingOrganizationName: *s.cfg.IssuingOrganizationName,
		AuthorityContract:       *s.cfg.AuthorityContract,
		RegistryContract:        *s.cfg.RegistryContract,
		ChainID:                 chainID,
		LastBlock:               lastBlock,
	}, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./feature/meta/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add feature/meta/meta_service.go feature/meta/meta_service_test.go
git commit -m "feat: add MetaService with Get method"
```

---

### Task 6: Meta Handler

**Files:**
- Create: `feature/meta/meta_handler.go`
- Create: `feature/meta/meta_handler_test.go`

**Interfaces:**
- Consumes: `MetaService` (from Task 5), `domain.CodeMetaSuccess` (from Task 2)
- Produces: `MetaHandler interface { Get(c *gin.Context) }` (consumed by Task 7)

- [ ] **Step 1: Write the failing handler test**

```go
// feature/meta/meta_handler_test.go
package meta

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/http/response"
	"CredChain_Golang/infrastructure/testutil/gintest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockService struct {
	mock.Mock
}

func (m *mockService) Get(ctx context.Context) (*response.Meta, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*response.Meta), args.Error(1)
}

func TestHandlerGet_Success(t *testing.T) {
	svc := new(mockService)
	handler := NewMetaHandler(MetaHandlerParams{Svc: svc})

	ginCtx, rr := gintest.NewContext(t, gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)))

	svc.On("Get", mock.Anything).Return(&response.Meta{
		IssuingOrganizationName: "University of Indonesia",
		AuthorityContract:       "0xAAA",
		RegistryContract:        "0xBBB",
		ChainID:                 137,
		LastBlock:               42000000,
	}, nil)

	handler.Get(ginCtx)

	assert.Equal(t, http.StatusOK, rr.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	assert.Equal(t, float64(domain.CodeMetaSuccess), body["code"])

	data := body["data"].(map[string]interface{})
	assert.Equal(t, "University of Indonesia", data["issuing_organization_name"])
	assert.Equal(t, "0xAAA", data["authority_contract"])
	assert.Equal(t, "0xBBB", data["registry_contract"])
	assert.Equal(t, float64(137), data["chain_id"])
	assert.Equal(t, float64(42000000), data["last_block"])
}

func TestHandlerGet_ServiceError(t *testing.T) {
	svc := new(mockService)
	handler := NewMetaHandler(MetaHandlerParams{Svc: svc})

	ginCtx, rr := gintest.NewContext(t, gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)))

	svc.On("Get", mock.Anything).Return(nil, domain.NewError(domain.CodeMetaInternal))

	handler.Get(ginCtx)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	assert.Equal(t, float64(domain.CodeMetaInternal), body["code"])
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./feature/meta/... -v`
Expected: compilation error — `MetaHandler`, `NewMetaHandler`, `MetaHandlerParams`, `Get` not defined

- [ ] **Step 3: Write handler implementation**

```go
// feature/meta/meta_handler.go
package meta

import (
	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/http/responder"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type MetaHandler interface {
	Get(c *gin.Context)
}

type metaHandler struct {
	svc MetaService
}

type MetaHandlerParams struct {
	fx.In
	Svc MetaService
}

func NewMetaHandler(p MetaHandlerParams) MetaHandler {
	return &metaHandler{svc: p.Svc}
}

func (h *metaHandler) Get(c *gin.Context) {
	meta, err := h.svc.Get(c.Request.Context())
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}

	responder.Send(c, domain.CodeMetaSuccess, meta)
}
```

- [ ] **Step 4: Run handler tests**

Run: `go test ./feature/meta/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add feature/meta/meta_handler.go feature/meta/meta_handler_test.go
git commit -m "feat: add MetaHandler with GET /api/meta endpoint"
```

---

### Task 7: Wiring — Mapper, Locales, Router

**Files:**
- Modify: `infrastructure/http/responder/mapper.go` (add to CodeToMessageKey and HttpCodes)
- Modify: `locales/en.json` (add message keys)
- Modify: `locales/id.json` (add message keys)
- Modify: `infrastructure/http/router.go` (add MetaHandler to RouteParams, register route)

**Interfaces:**
- Consumes: `domain.CodeMetaSuccess`, `domain.CodeMetaInternal` (from Task 2), `meta.MetaHandler` (from Task 6)

- [ ] **Step 1: Add mapper entries**

Add to `CodeToMessageKey` after the overview entries:

```go
	// Meta codes
	domain.CodeMetaSuccess:  "success_meta",
	domain.CodeMetaInternal: "error_meta_internal",
```

Add to `HttpCodes` after the overview entries:

```go
	domain.CodeMetaSuccess:  http.StatusOK,
	domain.CodeMetaInternal: http.StatusInternalServerError,
```

- [ ] **Step 2: Add locale entries**

Add to `locales/en.json` before the closing `}`:

```json
    "success_meta": "Meta information retrieved successfully.",
    "error_meta_internal": "Failed to retrieve meta information."
```

Add to `locales/id.json` before the closing `}`:

```json
    "success_meta": "Informasi meta berhasil diambil.",
    "error_meta_internal": "Gagal mengambil informasi meta."
```

Note: Add a comma after `"error_overview_internal": "Gagal mengambil data overview."` in id.json, then add the two new entries followed by the closing `}` without trailing comma.

- [ ] **Step 3: Add MetaHandler to RouteParams and register route**

In `infrastructure/http/router.go`:

Add import: `"CredChain_Golang/feature/meta"` to the import block.

Add to `RouteParams` struct (after `OverviewHandler` line):

```go
	MetaHandler                meta.MetaHandler
```

In `RegisterRoutes`, add on the public `api` group (after the health endpoint):

```go
		api.GET("/meta", p.MetaHandler.Get)
```

- [ ] **Step 4: Run all tests to verify locale and mapper integrity**

Run: `go test ./... -v`
Expected: PASS (locale_keys_test.go verifies all message keys exist in both locales; mapper_test.go verifies all codes have entries)

- [ ] **Step 5: Commit**

```bash
git add infrastructure/http/responder/mapper.go locales/en.json locales/id.json infrastructure/http/router.go
git commit -m "feat: wire MetaHandler route, mapper entries, and locale messages"
```

---

### Task 8: DI Wiring in cmd/server.go

**Files:**
- Modify: `cmd/server.go` (add fx.Provide entries)

**Interfaces:**
- Consumes: `meta.NewMetaService`, `meta.NewMetaHandler` (from Task 5, 6)

- [ ] **Step 1: Add import and FX providers**

Add import: `"CredChain_Golang/feature/meta"` to the import block.

Add to `fx.Provide(...)` in `cmd/server.go` — after the overview providers:

```go
				meta.NewMetaService,
				meta.NewMetaHandler,
```

- [ ] **Step 2: Verify build and tests**

Run: `go build ./... && go test ./...`
Expected: build succeeds, all tests pass

- [ ] **Step 3: Commit**

```bash
git add cmd/server.go
git commit -m "feat: register MetaService and MetaHandler in FX DI"
```

---

### Task 9: Final Verification

- [ ] **Step 1: Run full verification**

```bash
go test ./... && go vet ./... && gofmt -l .
```

Expected: all tests pass, `go vet` produces zero output, `gofmt -l .` produces zero output.

- [ ] **Step 2: Review git log**

```bash
git log --oneline -10
```
