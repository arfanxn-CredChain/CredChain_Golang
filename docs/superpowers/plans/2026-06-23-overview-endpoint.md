# Overview Endpoint Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `GET /api/overview` endpoint returning role-conditional dashboard data (credential counts, user counts, recents, chain details) for Holder and Issuer+ roles.

**Architecture:** Domain interfaces in `domain/overview.go`, GORM implementation in `feature/overview/gorm_overview_repository.go`. The overview repo handles **aggregate counts only** — recents are queried via existing `CredentialRepository.Get()` and `UserRepository.Get()`. Service orchestrates across repos, detects role, assembles response. Response DTOs live in `infrastructure/http/response/overview.go` (alongside `response.Credential`, `response.User`). Single endpoint on `/api/overview` with `AuthMiddleware` (no role gate — service checks role). Date filter via custom `date` key using the standard QueryRequest `..` BETWEEN syntax. `extractDateRange(q *Query)` helper lives in `feature/overview/helpers.go` — shared by repo and service.

**Tech Stack:** Go 1.25.1, Gin v1.12, GORM v1.31, Uber FX, testify, in-memory SQLite (test). Uses existing `response.Credential` / `response.User` DTOs for recents — no new credential/user DTOs needed.

---

## File Structure

```
domain/
  overview.go                         → OverviewCredentialCounts, OverviewUserCounts, OverviewRepository interface

infrastructure/http/response/
  overview.go                         → Overview, OverviewCredentialCounts (JSON tags), OverviewUserCounts (JSON tags),
                                        OverviewRecents, OverviewChainDetails. FromDomain*() mapping methods.
  overview_test.go                    → DTO mapping tests

feature/overview/
  helpers.go                          → extractDateRange(q *Query) (time.Time, time.Time) — shared by repo + service
  gorm_overview_repository.go         → gormOverviewRepository implementing domain.OverviewRepository (counts only)
  overview_service.go                 → OverviewService interface + implementation (role detection, query orchestration,
                                        injects overviewRepo + credRepo + userRepo + *chain.Client)
  overview_handler.go                 → OverviewHandler interface + Gin handler
  overview_handler_test.go            → Handler tests (mock service)
  overview_service_test.go            → Service tests (mock repos)
  gorm_overview_repository_test.go    → Repository tests (in-memory SQLite)

Shared file edits:
  domain/codes.go                     → CodeOverviewSuccess, CodeOverviewInternal
  infrastructure/chain/client.go      → Add BlockNumber method
  infrastructure/http/responder/mapper.go → Register codes + allDomainCodes
  infrastructure/http/router.go       → Register GET /api/overview
  cmd/server.go                       → FX providers
  locales/en.json, locales/id.json    → Message keys
  AGENTS.md, ROLES.md, CREDENTIAL.md  → API route docs
  CredChain_postman_collection.json   → Endpoint examples
```

---

### Task 1: Domain Codes + Domain Types + Client Method

**Files:**
- Modify: `domain/codes.go:14-15`
- Create: `domain/overview.go`
- Modify: `infrastructure/chain/client.go:116`

- [ ] **Step 1: Add overview response codes**

```go
// domain/codes.go — add after line 13 (CodeSystemInternal)
CodeOverviewSuccess  = 100100
CodeOverviewInternal = 100150
```

- [ ] **Step 2: Create domain overview types and interface**

```go
// domain/overview.go
package domain

import (
	"context"

	domainQuery "CredChain_Golang/domain/query"
)

type OverviewCredentialCounts struct {
	Total, Active, Revoked, Pending, Failed int
}

type OverviewUserCounts struct {
	Total, Holder, Issuer, Admin, SuperAdmin, Active, Trashed int
}

type OverviewRepository interface {
	CredentialCounts(ctx context.Context, q *domainQuery.Query, holderUserID *string) (*OverviewCredentialCounts, error)
	UserCounts(ctx context.Context, q *domainQuery.Query) (*OverviewUserCounts, error)
}
```

- [ ] **Step 3: Add BlockNumber method to chain.Client**

```go
// infrastructure/chain/client.go — add after NewClient() (after line 116)

func (c *Client) BlockNumber(ctx context.Context) (uint64, error) {
	return c.EthClient.BlockNumber(ctx)
}
```

- [ ] **Step 4: Verify compilation**

```bash
go build ./...
```

Expected: successful build.

- [ ] **Step 5: Commit**

```bash
git add domain/codes.go domain/overview.go infrastructure/chain/client.go
git commit -m "feat: add domain overview types, codes, and chain.BlockNumber"
```

---

### Task 2: Response DTOs

**Files:**
- Create: `infrastructure/http/response/overview.go`
- Create: `infrastructure/http/response/overview_test.go`

- [ ] **Step 1: Create response DTO file**

```go
// infrastructure/http/response/overview.go
package response

import "CredChain_Golang/domain"

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

func FromDomainOverviewCredentialCounts(d domain.OverviewCredentialCounts) OverviewCredentialCounts {
	return OverviewCredentialCounts{
		Total: d.Total, Active: d.Active, Revoked: d.Revoked,
		Pending: d.Pending, Failed: d.Failed,
	}
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

func FromDomainOverviewUserCounts(d domain.OverviewUserCounts) OverviewUserCounts {
	return OverviewUserCounts{
		Total: d.Total, Holder: d.Holder, Issuer: d.Issuer,
		Admin: d.Admin, SuperAdmin: d.SuperAdmin,
		Active: d.Active, Trashed: d.Trashed,
	}
}

type OverviewRecents struct {
	ActiveCredentials  []Credential `json:"active_credentials"`
	RevokedCredentials []Credential `json:"revoked_credentials"`
	StoredUsers        []User       `json:"stored_users"`
}

type OverviewChainDetails struct {
	AuthorityContract string `json:"authority_contract"`
	RegistryContract  string `json:"registry_contract"`
	LastBlock         uint64 `json:"last_block"`
}
```

- [ ] **Step 2: Create response DTO test file**

```go
// infrastructure/http/response/overview_test.go
package response

import (
	"testing"

	"CredChain_Golang/domain"

	"github.com/stretchr/testify/assert"
)

func TestFromDomainOverviewCredentialCounts(t *testing.T) {
	d := domain.OverviewCredentialCounts{Total: 10, Active: 8, Revoked: 1, Pending: 1, Failed: 1}
	r := FromDomainOverviewCredentialCounts(d)

	assert.Equal(t, 10, r.Total)
	assert.Equal(t, 8, r.Active)
	assert.Equal(t, 1, r.Revoked)
	assert.Equal(t, 1, r.Pending)
	assert.Equal(t, 1, r.Failed)
}

func TestFromDomainOverviewUserCounts(t *testing.T) {
	d := domain.OverviewUserCounts{Total: 5, Holder: 3, Issuer: 1, Admin: 1, Active: 4, Trashed: 1}
	r := FromDomainOverviewUserCounts(d)

	assert.Equal(t, 5, r.Total)
	assert.Equal(t, 3, r.Holder)
	assert.Equal(t, 1, r.Issuer)
	assert.Equal(t, 1, r.Admin)
	assert.Equal(t, 4, r.Active)
	assert.Equal(t, 1, r.Trashed)
}
```

- [ ] **Step 3: Run response tests**

```bash
go test ./infrastructure/http/response/... -v -run "TestFromDomain"
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add infrastructure/http/response/overview.go infrastructure/http/response/overview_test.go
git commit -m "feat: add overview response DTOs with FromDomain mappers"
```

---

### Task 3: Helpers + GORM Overview Repository (counts only)

**Files:**
- Create: `feature/overview/helpers.go`
- Create: `feature/overview/gorm_overview_repository.go`

- [ ] **Step 1: Create helpers file**

```go
// feature/overview/helpers.go
package overview

import (
	"time"

	domainQuery "CredChain_Golang/domain/query"
)

func extractDateRange(q *domainQuery.Query) (time.Time, time.Time) {
	if q == nil {
		return defaultDateRange()
	}
	for _, f := range q.Filters {
		if f.Column == "date" && f.Operator == domainQuery.OperatorBetween && len(f.Values) == 2 {
			from, err1 := time.Parse("2006-01-02", f.Values[0])
			to, err2 := time.Parse("2006-01-02", f.Values[1])
			if err1 == nil && err2 == nil {
				return from, to.Add(24*time.Hour - time.Second)
			}
		}
	}
	return defaultDateRange()
}

func defaultDateRange() (time.Time, time.Time) {
	return time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
}
```

- [ ] **Step 2: Create repository file**

```go
// feature/overview/gorm_overview_repository.go
package overview

import (
	"context"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"

	"gorm.io/gorm"
)

type gormOverviewRepository struct {
	db *gorm.DB
}

func NewGormOverviewRepository(db *gorm.DB) domain.OverviewRepository {
	return &gormOverviewRepository{db: db}
}

func (r *gormOverviewRepository) CredentialCounts(ctx context.Context, q *domainQuery.Query, holderUserID *string) (*domain.OverviewCredentialCounts, error) {
	dateFrom, dateTo := extractDateRange(q)

	type row struct{ Total, Active, Revoked, Pending, Failed int }

	if holderUserID == nil {
		var result row
		err := r.db.WithContext(ctx).Raw(`
			SELECT
				(SELECT COUNT(*) FROM credentials WHERE issued_at BETWEEN ? AND ?) AS total,
				(SELECT COUNT(*) FROM credentials
				 WHERE revoked_at IS NULL AND extract_status IN ('pending', 'succeeded')
				 AND issued_at BETWEEN ? AND ?) AS active,
				(SELECT COUNT(*) FROM credentials
				 WHERE revoked_at IS NOT NULL AND extract_status IN ('pending', 'succeeded')
				 AND issued_at BETWEEN ? AND ?) AS revoked,
				(SELECT COUNT(*) FROM credentials
				 WHERE extract_status = 'pending' AND issued_at BETWEEN ? AND ?) AS pending,
				(SELECT COUNT(*) FROM credentials
				 WHERE extract_status = 'failed' AND issued_at BETWEEN ? AND ?) AS failed
		`, dateFrom, dateTo, dateFrom, dateTo, dateFrom, dateTo, dateFrom, dateTo, dateFrom, dateTo).Scan(&result).Error
		if err != nil {
			return nil, err
		}
		return &domain.OverviewCredentialCounts{
			Total: result.Total, Active: result.Active, Revoked: result.Revoked,
			Pending: result.Pending, Failed: result.Failed,
		}, nil
	}

	var result row
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			(SELECT COUNT(*) FROM credentials WHERE holder_user_id = ? AND issued_at BETWEEN ? AND ?) AS total,
			(SELECT COUNT(*) FROM credentials
			 WHERE holder_user_id = ? AND revoked_at IS NULL AND extract_status IN ('pending', 'succeeded')
			 AND issued_at BETWEEN ? AND ?) AS active,
			(SELECT COUNT(*) FROM credentials
			 WHERE holder_user_id = ? AND revoked_at IS NOT NULL AND extract_status IN ('pending', 'succeeded')
			 AND issued_at BETWEEN ? AND ?) AS revoked,
			(SELECT COUNT(*) FROM credentials
			 WHERE holder_user_id = ? AND extract_status = 'pending' AND issued_at BETWEEN ? AND ?) AS pending,
			(SELECT COUNT(*) FROM credentials
			 WHERE holder_user_id = ? AND extract_status = 'failed' AND issued_at BETWEEN ? AND ?) AS failed
	`,
		*holderUserID, dateFrom, dateTo,
		*holderUserID, dateFrom, dateTo,
		*holderUserID, dateFrom, dateTo,
		*holderUserID, dateFrom, dateTo,
		*holderUserID, dateFrom, dateTo,
	).Scan(&result).Error
	if err != nil {
		return nil, err
	}
	return &domain.OverviewCredentialCounts{
		Total: result.Total, Active: result.Active, Revoked: result.Revoked,
		Pending: result.Pending, Failed: result.Failed,
	}, nil
}

func (r *gormOverviewRepository) UserCounts(ctx context.Context, q *domainQuery.Query) (*domain.OverviewUserCounts, error) {
	dateFrom, dateTo := extractDateRange(q)

	type row struct{ Total, Holder, Issuer, Admin, SuperAdmin, Active, Trashed int }

	var result row
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			(SELECT COUNT(*) FROM users WHERE created_at BETWEEN ? AND ?) AS total,
			(SELECT COUNT(*) FROM users WHERE role = 'holder' AND created_at BETWEEN ? AND ?) AS holder,
			(SELECT COUNT(*) FROM users WHERE role = 'issuer' AND created_at BETWEEN ? AND ?) AS issuer,
			(SELECT COUNT(*) FROM users WHERE role = 'admin' AND created_at BETWEEN ? AND ?) AS admin,
			(SELECT COUNT(*) FROM users WHERE role = 'super_admin' AND created_at BETWEEN ? AND ?) AS super_admin,
			(SELECT COUNT(*) FROM users WHERE deleted_at IS NULL AND created_at BETWEEN ? AND ?) AS active,
			(SELECT COUNT(*) FROM users WHERE deleted_at IS NOT NULL AND created_at BETWEEN ? AND ?) AS trashed
	`,
		dateFrom, dateTo, dateFrom, dateTo, dateFrom, dateTo, dateFrom, dateTo,
		dateFrom, dateTo, dateFrom, dateTo, dateFrom, dateTo,
	).Scan(&result).Error
	if err != nil {
		return nil, err
	}
	return &domain.OverviewUserCounts{
		Total: result.Total, Holder: result.Holder, Issuer: result.Issuer,
		Admin: result.Admin, SuperAdmin: result.SuperAdmin,
		Active: result.Active, Trashed: result.Trashed,
	}, nil
}
```

- [ ] **Step 3: Verify compilation**

```bash
go build ./feature/overview/...
```

Expected: successful build.

- [ ] **Step 4: Commit**

```bash
git add feature/overview/helpers.go feature/overview/gorm_overview_repository.go
git commit -m "feat: add overview helpers and repository with aggregate counts"
```

---

### Task 4: Repository Tests

**Files:**
- Create: `feature/overview/gorm_overview_repository_test.go`

- [ ] **Step 1: Create repository test file**

```go
// feature/overview/gorm_overview_repository_test.go
package overview

import (
	"context"
	"testing"
	"time"

	domainQuery "CredChain_Golang/domain/query"
	"CredChain_Golang/infrastructure/database/gorm/model"
	"CredChain_Golang/infrastructure/testutil/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func seedRepoTestData(t *testing.T, gormDB *gorm.DB) {
	t.Helper()
	now := time.Now()

	users := []model.User{
		{ID: "01J00000000000000000000001", Name: ptr("John"), Email: "john@test.com", Role: "holder", WalletAddress: "0x1", CreatedAt: now},
		{ID: "01J00000000000000000000002", Name: ptr("IssuerA"), Email: "issuer@test.com", Role: "issuer", WalletAddress: "0x2", CreatedAt: now.Add(-24 * time.Hour)},
		{ID: "01J00000000000000000000003", Name: ptr("Admin"), Email: "admin@test.com", Role: "admin", WalletAddress: "0x3", CreatedAt: now.Add(-48 * time.Hour)},
		{ID: "01J00000000000000000000004", Name: ptr("Trashed"), Email: "trashed@test.com", Role: "holder", WalletAddress: "0x4", CreatedAt: now.Add(-72 * time.Hour), DeletedAt: gorm.DeletedAt{Time: now, Valid: true}},
	}
	require.NoError(t, gormDB.Create(&users).Error)

	creds := []model.Credential{
		{ID: "01J000000000000000000000C1", HolderUserID: "01J00000000000000000000001", IssuerUserID: "01J00000000000000000000002", Name: "Active-Degree", FileHash: "0xaa", ExtractStatus: "succeeded", IssuedAt: now},
		{ID: "01J000000000000000000000C2", HolderUserID: "01J00000000000000000000001", IssuerUserID: "01J00000000000000000000002", Name: "Revoked-Diploma", FileHash: "0xbb", ExtractStatus: "succeeded", IssuedAt: now.Add(-1 * time.Hour), RevokedAt: ptr(now), RevokerUserID: ptr("01J00000000000000000000002")},
		{ID: "01J000000000000000000000C3", HolderUserID: "01J00000000000000000000001", IssuerUserID: "01J00000000000000000000002", Name: "Pending-Extract", FileHash: "0xcc", ExtractStatus: "pending", IssuedAt: now.Add(-2 * time.Hour)},
		{ID: "01J000000000000000000000C4", HolderUserID: "01J00000000000000000000001", IssuerUserID: "01J00000000000000000000002", Name: "Failed-Extract", FileHash: "0xdd", ExtractStatus: "failed", IssuedAt: now.Add(-3 * time.Hour)},
		{ID: "01J000000000000000000000C5", HolderUserID: "01J00000000000000000000003", IssuerUserID: "01J00000000000000000000002", Name: "Admin-Active", FileHash: "0xee", ExtractStatus: "succeeded", IssuedAt: now.Add(-4 * time.Hour)},
	}
	require.NoError(t, gormDB.Create(&creds).Error)
}

func TestCredentialCounts_Issuer(t *testing.T) {
	gormDB := db.OpenInMemorySQLite(t)
	seedRepoTestData(t, gormDB)
	repo := NewGormOverviewRepository(gormDB)
	ctx := context.Background()

	q := &domainQuery.Query{}

	counts, err := repo.CredentialCounts(ctx, q, nil)
	require.NoError(t, err)

	assert.Equal(t, 5, counts.Total)
	assert.Equal(t, 2, counts.Active)
	assert.Equal(t, 1, counts.Revoked)
	assert.Equal(t, 1, counts.Pending)
	assert.Equal(t, 1, counts.Failed)
}

func TestCredentialCounts_Holder(t *testing.T) {
	gormDB := db.OpenInMemorySQLite(t)
	seedRepoTestData(t, gormDB)
	repo := NewGormOverviewRepository(gormDB)
	ctx := context.Background()

	q := &domainQuery.Query{}
	holderID := "01J00000000000000000000001"

	counts, err := repo.CredentialCounts(ctx, q, &holderID)
	require.NoError(t, err)

	assert.Equal(t, 4, counts.Total)
	assert.Equal(t, 2, counts.Active)
	assert.Equal(t, 1, counts.Revoked)
	assert.Equal(t, 1, counts.Pending)
	assert.Equal(t, 1, counts.Failed)
}

func TestCredentialCounts_DateFilter(t *testing.T) {
	gormDB := db.OpenInMemorySQLite(t)
	seedRepoTestData(t, gormDB)
	repo := NewGormOverviewRepository(gormDB)
	ctx := context.Background()

	now := time.Now()
	from := now.Add(-30 * time.Minute)
	to := now.Add(24 * time.Hour)
	fromStr := from.Format("2006-01-02")
	toStr := to.Format("2006-01-02")

	q := &domainQuery.Query{
		Filters: []domainQuery.Filter{
			{Column: "date", Operator: domainQuery.OperatorBetween, Values: []string{fromStr, toStr}},
		},
	}

	counts, err := repo.CredentialCounts(ctx, q, nil)
	require.NoError(t, err)
	assert.Greater(t, counts.Total, 0)
}

func TestUserCounts(t *testing.T) {
	gormDB := db.OpenInMemorySQLite(t)
	seedRepoTestData(t, gormDB)
	repo := NewGormOverviewRepository(gormDB)
	ctx := context.Background()

	q := &domainQuery.Query{}

	counts, err := repo.UserCounts(ctx, q)
	require.NoError(t, err)

	assert.Equal(t, 4, counts.Total)
	assert.Equal(t, 2, counts.Holder)
	assert.Equal(t, 1, counts.Issuer)
	assert.Equal(t, 1, counts.Admin)
	assert.Equal(t, 0, counts.SuperAdmin)
	assert.Equal(t, 3, counts.Active)
	assert.Equal(t, 1, counts.Trashed)
}

func TestExtractDateRange_NilQuery(t *testing.T) {
	from, to := extractDateRange(nil)
	assert.True(t, from.Before(to))
	assert.Equal(t, 1, from.Year())
	assert.Equal(t, 9999, to.Year())
}

func ptr[T any](v T) *T { return &v }
```

- [ ] **Step 2: Run repo tests**

```bash
go test ./feature/overview/... -v -run "TestCredentialCounts|TestUserCounts|TestExtract"
```

Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add feature/overview/gorm_overview_repository_test.go
git commit -m "test: add overview repository tests"
```

---

### Task 5: Overview Service

**Files:**
- Create: `feature/overview/overview_service.go`

- [ ] **Step 1: Create service file**

```go
// feature/overview/overview_service.go
package overview

import (
	"context"
	"time"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	"CredChain_Golang/infrastructure/chain"
	httpContext "CredChain_Golang/infrastructure/http/context"
	"CredChain_Golang/infrastructure/http/response"

	"go.uber.org/fx"
)

type OverviewService interface {
	Get(ctx context.Context, q *domainQuery.Query) (*response.Overview, error)
}

type overviewService struct {
	overviewRepo domain.OverviewRepository
	credRepo     domain.CredentialRepository
	userRepo     domain.UserRepository
	cfg          *config.Config
	chainClient  *chain.Client
}

type OverviewServiceParams struct {
	fx.In
	OverviewRepo domain.OverviewRepository
	CredRepo     domain.CredentialRepository
	UserRepo     domain.UserRepository
	Config       *config.Config
	ChainClient  *chain.Client
}

func NewOverviewService(p OverviewServiceParams) OverviewService {
	return &overviewService{
		overviewRepo: p.OverviewRepo,
		credRepo:     p.CredRepo,
		userRepo:     p.UserRepo,
		cfg:          p.Config,
		chainClient:  p.ChainClient,
	}
}

func (s *overviewService) Get(ctx context.Context, q *domainQuery.Query) (*response.Overview, error) {
	authUser := httpContext.MustGetUser(ctx)
	isIssuer := authUser.Role.Rank() >= domain.RoleIssuer.Rank()

	dateFrom, dateTo := extractDateRange(q)

	var holderID *string
	if !isIssuer {
		holderID = &authUser.Id
	}

	credCounts, err := s.overviewRepo.CredentialCounts(ctx, q, holderID)
	if err != nil {
		return nil, domain.NewError(domain.CodeOverviewInternal, domain.WithError(err))
	}

	activeQ := buildRecentActiveCredentialQuery(dateFrom, dateTo, 5, holderID)
	activeCreds, _, err := s.credRepo.Get(ctx, activeQ)
	if err != nil {
		return nil, domain.NewError(domain.CodeOverviewInternal, domain.WithError(err))
	}

	revokedQ := buildRecentRevokedCredentialQuery(dateFrom, dateTo, 5, holderID)
	revokedCreds, _, err := s.credRepo.Get(ctx, revokedQ)
	if err != nil {
		return nil, domain.NewError(domain.CodeOverviewInternal, domain.WithError(err))
	}

	dtoCredCounts := response.FromDomainOverviewCredentialCounts(*credCounts)
	ov := &response.Overview{
		CredentialCounts: &dtoCredCounts,
		Recents: &response.OverviewRecents{
			ActiveCredentials:  mapCredentials(activeCreds),
			RevokedCredentials: mapCredentials(revokedCreds),
		},
	}

	if isIssuer {
		userCounts, err := s.overviewRepo.UserCounts(ctx, q)
		if err != nil {
			return nil, domain.NewError(domain.CodeOverviewInternal, domain.WithError(err))
		}
		dtoUserCounts := response.FromDomainOverviewUserCounts(*userCounts)
		ov.UserCounts = &dtoUserCounts

		recentQ := buildRecentUsersQuery(dateFrom, dateTo, 5)
		recentUsers, _, err := s.userRepo.Get(ctx, recentQ)
		if err != nil {
			return nil, domain.NewError(domain.CodeOverviewInternal, domain.WithError(err))
		}
		ov.Recents.StoredUsers = mapUsers(recentUsers)

		var lastBlock uint64
		if s.chainClient != nil {
			lastBlock, err = s.chainClient.BlockNumber(ctx)
			if err != nil {
				lastBlock = 0
			}
		}
		ov.ChainDetails = &response.OverviewChainDetails{
			AuthorityContract: *s.cfg.AuthorityContract,
			RegistryContract:  *s.cfg.RegistryContract,
			LastBlock:         lastBlock,
		}
	}

	return ov, nil
}

func buildRecentActiveCredentialQuery(dateFrom, dateTo time.Time, limit int, holderID *string) *domainQuery.Query {
	q := &domainQuery.Query{
		Includes: []string{"holder", "issuer"},
		Sorts:    []domainQuery.Sort{{Column: "issued_at", Direction: domainQuery.SortDesc}},
		Pagination: &domainQuery.Pagination{Limit: limit},
	}
	if holderID != nil {
		q.Filters = append(q.Filters, domainQuery.Filter{Column: "holder_user_id", Operator: domainQuery.OperatorEqual, Values: []string{*holderID}})
	}
	return q
}

func buildRecentRevokedCredentialQuery(dateFrom, dateTo time.Time, limit int, holderID *string) *domainQuery.Query {
	q := &domainQuery.Query{
		Includes: []string{"holder", "revoker"},
		Sorts:    []domainQuery.Sort{{Column: "revoked_at", Direction: domainQuery.SortDesc}},
		Pagination: &domainQuery.Pagination{Limit: limit},
	}
	if holderID != nil {
		q.Filters = append(q.Filters, domainQuery.Filter{Column: "holder_user_id", Operator: domainQuery.OperatorEqual, Values: []string{*holderID}})
	}
	return q
}

func buildRecentUsersQuery(dateFrom, dateTo time.Time, limit int) *domainQuery.Query {
	return &domainQuery.Query{
		Sorts:      []domainQuery.Sort{{Column: "created_at", Direction: domainQuery.SortDesc}},
		Pagination: &domainQuery.Pagination{Limit: limit},
	}
}

func mapCredentials(creds []domain.Credential) []response.Credential {
	out := make([]response.Credential, len(creds))
	for i, c := range creds {
		out[i] = response.FromDomainCredential(c)
	}
	return out
}

func mapUsers(users []domain.User) []response.User {
	out := make([]response.User, len(users))
	for i, u := range users {
		out[i] = response.FromDomainUser(u)
	}
	return out
}
```

Note: The `Get` method from `CredentialRepository` and `UserRepository` returns `([]domain.Credential, int, error)` / `([]domain.User, int, error)`. The int (total) is ignored for recents. The `Pagination.Limit` field — verify it's just `Limit int` (not `Limit *int` or another type). If the field name differs, adjust.

- [ ] **Step 2: Verify compilation**

```bash
go build ./feature/overview/...
```

Expected: successful build.

- [ ] **Step 3: Commit**

```bash
git add feature/overview/overview_service.go
git commit -m "feat: add overview service with role detection and query orchestration"
```

---

### Task 6: Service Tests

**Files:**
- Create: `feature/overview/overview_service_test.go`

- [ ] **Step 1: Create service test file**

```go
// feature/overview/overview_service_test.go
package overview

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	"CredChain_Golang/infrastructure/http/response"
	"CredChain_Golang/infrastructure/testutil/gintest"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockOverviewRepo struct{ mock.Mock }

func (m *mockOverviewRepo) CredentialCounts(ctx context.Context, q *domainQuery.Query, holderUserID *string) (*domain.OverviewCredentialCounts, error) {
	args := m.Called(ctx, q, holderUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.OverviewCredentialCounts), args.Error(1)
}

func (m *mockOverviewRepo) UserCounts(ctx context.Context, q *domainQuery.Query) (*domain.OverviewUserCounts, error) {
	args := m.Called(ctx, q)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.OverviewUserCounts), args.Error(1)
}

type mockCredRepo struct{ mock.Mock }

func (m *mockCredRepo) Get(ctx context.Context, q *domainQuery.Query) ([]domain.Credential, int, error) {
	args := m.Called(ctx, q)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]domain.Credential), args.Int(1), args.Error(2)
}

func (m *mockCredRepo) Find(ctx context.Context, id string, q *domainQuery.Query) (*domain.Credential, error) { return nil, nil }
func (m *mockCredRepo) FindByIds(ctx context.Context, ids []string, q *domainQuery.Query) ([]domain.Credential, error) { return nil, nil }
func (m *mockCredRepo) Store(ctx context.Context, creds ...domain.Credential) ([]domain.Credential, error) { return nil, nil }
func (m *mockCredRepo) Update(ctx context.Context, creds ...domain.Credential) ([]domain.Credential, error) { return nil, nil }
func (m *mockCredRepo) FindByFileHashes(ctx context.Context, hashes []string, q *domainQuery.Query) ([]domain.Credential, error) { return nil, nil }
func (m *mockCredRepo) FindByHolderId(ctx context.Context, holderID string, q *domainQuery.Query) ([]domain.Credential, error) { return nil, nil }

type mockUserRepo struct{ mock.Mock }

func (m *mockUserRepo) Get(ctx context.Context, q *domainQuery.Query) ([]domain.User, int, error) {
	args := m.Called(ctx, q)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]domain.User), args.Int(1), args.Error(2)
}

func (m *mockUserRepo) Find(ctx context.Context, id string) (*domain.User, error)              { return nil, nil }
func (m *mockUserRepo) FindByIds(ctx context.Context, ids ...string) ([]domain.User, error)   { return nil, nil }
func (m *mockUserRepo) FindByEmails(ctx context.Context, emails ...string) ([]domain.User, error) { return nil, nil }
func (m *mockUserRepo) FindByRole(ctx context.Context, role domain.Role) ([]domain.User, error) { return nil, nil }
func (m *mockUserRepo) Store(ctx context.Context, users ...domain.User) ([]domain.User, error) { return nil, nil }
func (m *mockUserRepo) Update(ctx context.Context, users ...domain.User) ([]domain.User, error) { return nil, nil }
func (m *mockUserRepo) UpdateRole(ctx context.Context, updates ...domain.UserRoleUpdate) ([]domain.User, int64, error) { return nil, 0, nil }
func (m *mockUserRepo) Delete(ctx context.Context, ids ...string) (int64, error)              { return 0, nil }
func (m *mockUserRepo) Restore(ctx context.Context, ids ...string) (int64, error)             { return 0, nil }

func setupTestService(t *testing.T, user *domain.User) (*overviewService, *mockOverviewRepo, *mockCredRepo, *mockUserRepo, *gin.Context) {
	t.Helper()
	repo := new(mockOverviewRepo)
	credRepo := new(mockCredRepo)
	userRepo := new(mockUserRepo)
	cfg := &config.Config{
		AuthorityContract: ptrStr("0xAuthority"),
		RegistryContract:  ptrStr("0xRegistry"),
	}
	ginCtx, _ := gintest.NewContext(t, gintest.WithUser(user))
	ctx := ginCtx.Request.Context()
	ginCtx.Request = httptest.NewRequest("GET", "/overview", nil).WithContext(ctx)
	svc := &overviewService{
		overviewRepo: repo,
		credRepo:     credRepo,
		userRepo:     userRepo,
		cfg:          cfg,
		chainClient:  nil,
	}
	return svc, repo, credRepo, userRepo, ginCtx
}

func TestGet_Issuer(t *testing.T) {
	issuer := &domain.User{Id: "issuer1", Role: domain.RoleIssuer, Email: "issuer@test.com", WalletAddress: "0x1"}
	svc, repo, credRepo, userRepo, ginCtx := setupTestService(t, issuer)
	q := &domainQuery.Query{}

	repo.On("CredentialCounts", mock.Anything, q, (*string)(nil)).Return(&domain.OverviewCredentialCounts{Total: 10, Active: 8, Revoked: 1, Pending: 1, Failed: 1}, nil)
	repo.On("UserCounts", mock.Anything, q).Return(&domain.OverviewUserCounts{Total: 5, Holder: 3, Issuer: 1, Admin: 1, Active: 4, Trashed: 1}, nil)
	credRepo.On("Get", mock.Anything, mock.Anything).Return([]domain.Credential{}, 0, nil)
	userRepo.On("Get", mock.Anything, mock.Anything).Return([]domain.User{}, 0, nil)

	result, err := svc.Get(ginCtx.Request.Context(), q)
	require.NoError(t, err)

	assert.NotNil(t, result.CredentialCounts)
	assert.Equal(t, 10, result.CredentialCounts.Total)
	assert.NotNil(t, result.UserCounts)
	assert.Equal(t, 5, result.UserCounts.Total)
	assert.NotNil(t, result.Recents)
	assert.NotNil(t, result.ChainDetails)
	assert.Equal(t, "0xAuthority", result.ChainDetails.AuthorityContract)
	assert.Equal(t, uint64(0), result.ChainDetails.LastBlock)
	assert.NotEmpty(t, result.Recents.StoredUsers)

	repo.AssertExpectations(t)
}

func TestGet_Holder(t *testing.T) {
	holder := &domain.User{Id: "holder1", Role: domain.RoleHolder, Email: "holder@test.com", WalletAddress: "0x2"}
	svc, repo, credRepo, _, ginCtx := setupTestService(t, holder)
	q := &domainQuery.Query{}

	repo.On("CredentialCounts", mock.Anything, q, &holder.Id).Return(&domain.OverviewCredentialCounts{Total: 5, Active: 4, Revoked: 1, Pending: 0, Failed: 0}, nil)
	credRepo.On("Get", mock.Anything, mock.Anything).Return([]domain.Credential{}, 0, nil)

	result, err := svc.Get(ginCtx.Request.Context(), q)
	require.NoError(t, err)

	assert.NotNil(t, result.CredentialCounts)
	assert.Equal(t, 5, result.CredentialCounts.Total)
	assert.Nil(t, result.UserCounts)
	assert.NotNil(t, result.Recents)
	assert.Nil(t, result.ChainDetails)
	assert.Empty(t, result.Recents.StoredUsers)

	repo.AssertExpectations(t)
}

func TestGet_RepoError(t *testing.T) {
	issuer := &domain.User{Id: "issuer1", Role: domain.RoleIssuer, Email: "issuer@test.com", WalletAddress: "0x1"}
	svc, repo, _, _, ginCtx := setupTestService(t, issuer)
	q := &domainQuery.Query{}

	repo.On("CredentialCounts", mock.Anything, q, (*string)(nil)).Return(nil, errors.New("db down"))

	_, err := svc.Get(ginCtx.Request.Context(), q)
	require.Error(t, err)
}

func ptrStr(s string) *string { return &s }
```

Note: Verify `gintest.NewContext(t, gintest.WithUser(user))` API. Check `infrastructure/testutil/gintest/gintest.go`. Also verify `domain.CredentialRepository` and `domain.UserRepository` interfaces — all methods must be stubbed.

- [ ] **Step 2: Run service tests**

```bash
go test ./feature/overview/... -v -run "TestGet"
```

Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add feature/overview/overview_service_test.go
git commit -m "test: add overview service tests"
```

---

### Task 7: Overview Handler

**Files:**
- Create: `feature/overview/overview_handler.go`

- [ ] **Step 1: Create handler file**

```go
// feature/overview/overview_handler.go
package overview

import (
	"CredChain_Golang/domain"
	queryRequest "CredChain_Golang/infrastructure/http/request/query"
	"CredChain_Golang/infrastructure/http/responder"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type OverviewHandler interface {
	Get(c *gin.Context)
}

type overviewHandler struct {
	svc OverviewService
}

type OverviewHandlerParams struct {
	fx.In
	Svc OverviewService
}

func NewOverviewHandler(p OverviewHandlerParams) OverviewHandler {
	return &overviewHandler{svc: p.Svc}
}

func (h *overviewHandler) Get(c *gin.Context) {
	var req queryRequest.QueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	query, err := req.ToDomain()
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}

	ov, err := h.svc.Get(c.Request.Context(), query)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}

	responder.Send(c, domain.CodeOverviewSuccess, ov)
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./feature/overview/...
```

Expected: successful build.

- [ ] **Step 3: Commit**

```bash
git add feature/overview/overview_handler.go
git commit -m "feat: add overview handler"
```

---

### Task 8: Handler Tests

**Files:**
- Create: `feature/overview/overview_handler_test.go`

- [ ] **Step 1: Create handler test file**

```go
// feature/overview/overview_handler_test.go
package overview

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	"CredChain_Golang/infrastructure/http/response"
	"CredChain_Golang/infrastructure/testutil/gintest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockSvc struct{ mock.Mock }

func (m *mockSvc) Get(ctx context.Context, q *domainQuery.Query) (*response.Overview, error) {
	args := m.Called(ctx, q)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*response.Overview), args.Error(1)
}

func TestHandlerGet_IssuerSuccess(t *testing.T) {
	svc := new(mockSvc)
	handler := NewOverviewHandler(OverviewHandlerParams{Svc: svc})

	user := &domain.User{Id: "issuer1", Role: domain.RoleIssuer, Email: "issuer@test.com", WalletAddress: "0x1"}
	ginCtx, _ := gintest.NewContext(t, gintest.WithUser(user))

	now := time.Now()
	svc.On("Get", mock.Anything, mock.Anything).Return(&response.Overview{
		CredentialCounts: &response.OverviewCredentialCounts{Total: 10, Active: 8, Revoked: 1, Pending: 1, Failed: 1},
		UserCounts:       &response.OverviewUserCounts{Total: 5, Holder: 3, Issuer: 1, Admin: 1, Active: 4, Trashed: 1},
		Recents: &response.OverviewRecents{
			ActiveCredentials: []response.Credential{{ID: "c1", Name: "Degree", IssuedAt: now}},
		},
		ChainDetails: &response.OverviewChainDetails{AuthorityContract: "0xAA", RegistryContract: "0xBB", LastBlock: 100},
	}, nil)

	handler.Get(ginCtx)

	assert.Equal(t, http.StatusOK, ginCtx.Writer.Status())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(ginCtx.Writer.Body.Bytes(), &body))
	assert.Equal(t, float64(domain.CodeOverviewSuccess), body["code"])

	data := body["data"].(map[string]interface{})
	assert.NotNil(t, data["credential_counts"])
	assert.NotNil(t, data["user_counts"])
	assert.NotNil(t, data["chain_details"])
}

func TestHandlerGet_HolderSuccess(t *testing.T) {
	svc := new(mockSvc)
	handler := NewOverviewHandler(OverviewHandlerParams{Svc: svc})

	user := &domain.User{Id: "holder1", Role: domain.RoleHolder, Email: "holder@test.com", WalletAddress: "0x2"}
	ginCtx, _ := gintest.NewContext(t, gintest.WithUser(user))

	svc.On("Get", mock.Anything, mock.Anything).Return(&response.Overview{
		CredentialCounts: &response.OverviewCredentialCounts{Total: 5, Active: 4, Revoked: 1, Pending: 0, Failed: 0},
		Recents:          &response.OverviewRecents{ActiveCredentials: []response.Credential{}},
	}, nil)

	handler.Get(ginCtx)

	assert.Equal(t, http.StatusOK, ginCtx.Writer.Status())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(ginCtx.Writer.Body.Bytes(), &body))

	data := body["data"].(map[string]interface{})
	assert.NotNil(t, data["credential_counts"])
	assert.Nil(t, data["user_counts"])
	assert.Nil(t, data["chain_details"])
}
```

Note: Verify `gintest.NewContext(t, gintest.WithUser(user))` API from `infrastructure/testutil/gintest/gintest.go`. The handler uses `responder.Send` which needs i18n bundle — gintest may auto-load it.

- [ ] **Step 2: Run handler tests**

```bash
go test ./feature/overview/... -v -run "TestHandlerGet"
```

Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add feature/overview/overview_handler_test.go
git commit -m "test: add overview handler tests"
```

---

### Task 9: Router + FX Wiring

**Files:**
- Modify: `infrastructure/http/router.go:47-66,68-108`
- Modify: `cmd/server.go:31-91`

- [ ] **Step 1: Add OverviewHandler to RouteParams**

In `router.go`, add to `RouteParams` struct:

```go
OverviewHandler overview.OverviewHandler
```

Add import for overview package at top of `router.go`:
```go
"CredChain_Golang/feature/overview"
```

- [ ] **Step 2: Register the route**

In `router.go`, inside the `secure` group (using `p.AuthMiddleware`), add:

```go
secure.GET("/overview", p.OverviewHandler.Get)
```

- [ ] **Step 3: Add FX providers to server.go**

In `cmd/server.go`, add to `fx.Provide()` list — after credential providers, before middleware:

```go
overview.NewGormOverviewRepository,
overview.NewOverviewService,
overview.NewOverviewHandler,
```

Add import for overview package at top of `server.go`.

- [ ] **Step 4: Run build and vet**

```bash
go build ./...
go vet ./...
```

Expected: successful build, zero vet output.

- [ ] **Step 5: Commit**

```bash
git add infrastructure/http/router.go cmd/server.go
git commit -m "feat: register overview route and FX wiring"
```

---

### Task 10: Locale Keys + Mapper Registration

**Files:**
- Modify: `locales/en.json`
- Modify: `locales/id.json`
- Modify: `infrastructure/http/responder/mapper.go`

- [ ] **Step 1: Add locale message keys to en.json**

```json
"success_overview": "Overview fetched successfully",
"error_overview_internal": "Failed to fetch overview data"
```

- [ ] **Step 2: Add locale message keys to id.json**

```json
"success_overview": "Overview berhasil diambil",
"error_overview_internal": "Gagal mengambil data overview"
```

- [ ] **Step 3: Register codes in mapper.go**

In `infrastructure/http/responder/mapper.go`:

**CodeToMessageKey** — add after system entries:
```go
domain.CodeOverviewSuccess:  "success_overview",
domain.CodeOverviewInternal: "error_overview_internal",
```

**HttpCodes** — add after system entries:
```go
domain.CodeOverviewSuccess:  http.StatusOK,
domain.CodeOverviewInternal: http.StatusInternalServerError,
```

**In `mapper_test.go`'s `allDomainCodes` slice** — add after system entries:
```go
domain.CodeOverviewSuccess,
domain.CodeOverviewInternal,
```

- [ ] **Step 4: Run locale enforcement test**

```bash
go test ./infrastructure/http/responder/... -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add locales/en.json locales/id.json infrastructure/http/responder/mapper.go infrastructure/http/responder/mapper_test.go
git commit -m "feat: register overview locale keys and mapper codes"
```

---

### Task 11: Documentation Updates

**Files:**
- Modify: `AGENTS.md`
- Modify: `ROLES.md`
- Modify: `CREDENTIAL.md`

- [ ] **Step 1: Update AGENTS.md API Routes table**

Add after `GET /api/credentials/verify` row:

```
| GET | `/api/overview` | Authenticated (no role gate) | Role-conditional dashboard: credential counts, user counts, recents, chain details. Issuer+ get system-wide data; Holder get own only. |
```

- [ ] **Step 2: Update ROLES.md**

Add to API Route Authorization table:
```
| `/api/overview` | GET | Authenticated | Any — role-conditional response (Holder: own data; Issuer+: system-wide) |
```

Add to Per-Role Capability Matrix:
```
View overview dashboard            —      ✓       ✓       ✓       ✓
```

- [ ] **Step 3: Update CREDENTIAL.md**

Add to API Routes section:
```
| `/api/overview` | GET | Authenticated (no role gate) | `Get` | Role-conditional dashboard: credential_counts + recents (Holder: own, Issuer+: system-wide) |
```

- [ ] **Step 4: Commit**

```bash
git add AGENTS.md ROLES.md CREDENTIAL.md
git commit -m "docs: add overview endpoint to API documentation"
```

---

### Task 12: Postman Collection Update

**Files:**
- Modify: `CredChain_postman_collection.json`

- [ ] **Step 1: Add overview endpoint to Postman collection**

Add a new item under the API folder, following the existing pattern with response examples for both Issuer+ and Holder.

- [ ] **Step 2: Commit**

```bash
git add CredChain_postman_collection.json
git commit -m "docs: add overview endpoint to Postman collection"
```

---

## Verification

```bash
go test ./... -cover
go vet ./...
gofmt -l .
```

Expected: all tests pass, zero vet output, zero unformatted files.

---

## Notes for Implementer

1. **`gintest.NewContext`** — verify API in `infrastructure/testutil/gintest/gintest.go`. May need i18n bundle for handler tests.

2. **`domain.CredentialRepository` / `domain.UserRepository`** — verify all interface method signatures. Mock must implement every method.

3. **Query DSL operators** — `buildRecentActiveCredentialQuery` and `buildRecentRevokedCredentialQuery` use `Includes` + `Sorts` + `Pagination` without explicit `revoked_at IS NULL` / `IS NOT NULL` filters. This relies on the credential repo's `Get` method handling these. If the repo doesn't auto-filter, add explicit filters using `OperatorEqual` / `OperatorNotEqual`.

4. **`Pagination.Limit`** — verify field type. May be `*int` in which case use `&limit`.

5. **`extractDateRange`** — lives in `feature/overview/helpers.go`. Shared by repo (`gorm_overview_repository.go`) and service (`overview_service.go`). All in same package. Defined once.

6. **`response.Overview`** — lives in `infrastructure/http/response/overview.go`. Service returns `*response.Overview`. Handler calls `responder.Send(c, CodeOverviewSuccess, ov)`.

7. **Locale enforcement** — static messages (no `{{.X}}` placeholders), no `WithMetadata` calls needed.
