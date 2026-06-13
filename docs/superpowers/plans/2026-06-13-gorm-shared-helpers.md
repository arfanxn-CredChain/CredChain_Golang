# GORM Repository Shared Helpers Refactoring — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract duplicated filter/sort/pagination/CASE helpers from `gorm_user_repository.go` and `gorm_credential_repository.go` into a shared `infrastructure/database/gorm/helpers.go`, invert soft-delete default to always include trashed rows in `Get()`, and change user default sort from `created_at DESC` to `updated_at DESC`.

**Architecture:** Five exported functions in `infrastructure/database/gorm/helpers.go` (package `gorm`). Repos import via alias `gormhelpers "CredChain_Golang/infrastructure/database/gorm"` to avoid naming conflict with `gorm.io/gorm`. Entity-specific helpers (`preloadByIncludes`, `needsHolderJoin`, `mapSortColumn`) remain in their repos. All call-site-level behavior preserved — only wiring changes.

**Tech Stack:** Go 1.25, GORM v1.31, SQLite in-memory (glebarez), samber/lo, testify

---

## File Map

| Action | Path | Responsibility |
|--------|------|----------------|
| Create | `infrastructure/database/gorm/helpers.go` | Five exported generic GORM helper functions |
| Create | `infrastructure/database/gorm/helpers_test.go` | Unit tests for all five helpers |
| Modify | `feature/user/gorm_user_repository.go` | Replace inline helpers with shared ones; always `Unscoped()` in `Get`; default sort `updated_at DESC`; remove `referencesDeletedAt`, `applyUserFilters`, `updateBatchCase` |
| Modify | `feature/credential/gorm_credential_repository.go` | Replace inline helpers with shared ones; remove `applyCredentialFilters`, `updateBatchCase` |

---

### Task 1: Create `helpers.go` with `ApplyFilters`

**Files:**
- Create: `infrastructure/database/gorm/helpers.go`
- Create: `infrastructure/database/gorm/helpers_test.go`

- [ ] **Step 1: Write the failing test for `ApplyFilters`**

Append to `infrastructure/database/gorm/helpers_test.go`:

```go
package gorm

import (
	"testing"

	domainQuery "CredChain_Golang/domain/query"
	"CredChain_Golang/infrastructure/database/gorm/model"
	"CredChain_Golang/infrastructure/testutil/db"

	"github.com/stretchr/testify/assert"
)

func TestApplyFilters_Equal(t *testing.T) {
	gdb := db.OpenInMemorySQLite(t)
	allowed := map[string]bool{"name": true}

	sql := gdb.ToSQL(func(tx *gorm.DB) *gorm.DB {
		tx = tx.Model(&model.User{})
		tx = ApplyFilters(tx, []domainQuery.Filter{
			{Column: "name", Operator: domainQuery.OperatorEqual, Values: []string{"alice"}},
		}, allowed, "")
		var out []model.User
		return tx.Find(&out)
	})
	assert.Contains(t, sql, "name =")
}

func TestApplyFilters_NotInAllowlist(t *testing.T) {
	gdb := db.OpenInMemorySQLite(t)
	allowed := map[string]bool{"name": true}

	sql := gdb.ToSQL(func(tx *gorm.DB) *gorm.DB {
		tx = tx.Model(&model.User{})
		tx = ApplyFilters(tx, []domainQuery.Filter{
			{Column: "secret", Operator: domainQuery.OperatorEqual, Values: []string{"x"}},
		}, allowed, "")
		var out []model.User
		return tx.Find(&out)
	})
	assert.NotContains(t, sql, "secret")
}

func TestApplyFilters_ColumnPrefix(t *testing.T) {
	gdb := db.OpenInMemorySQLite(t)
	allowed := map[string]bool{"name": true}

	sql := gdb.ToSQL(func(tx *gorm.DB) *gorm.DB {
		tx = tx.Model(&model.User{})
		tx = ApplyFilters(tx, []domainQuery.Filter{
			{Column: "name", Operator: domainQuery.OperatorEqual, Values: []string{"cert"}},
		}, allowed, "credentials.")
		var out []model.User
		return tx.Find(&out)
	})
	assert.Contains(t, sql, "credentials.name")
}

func TestApplyFilters_NullAndNotNull(t *testing.T) {
	gdb := db.OpenInMemorySQLite(t)
	allowed := map[string]bool{"deleted_at": true}

	sql := gdb.ToSQL(func(tx *gorm.DB) *gorm.DB {
		tx = tx.Model(&model.User{})
		tx = ApplyFilters(tx, []domainQuery.Filter{
			{Column: "deleted_at", Operator: domainQuery.OperatorNull},
			{Column: "deleted_at", Operator: domainQuery.OperatorNotNull},
		}, allowed, "")
		var out []model.User
		return tx.Find(&out)
	})
	assert.Contains(t, sql, "IS NULL")
	assert.Contains(t, sql, "IS NOT NULL")
}

func TestApplyFilters_LikeCaseInsensitive(t *testing.T) {
	gdb := db.OpenInMemorySQLite(t)
	allowed := map[string]bool{"name": true}

	sql := gdb.ToSQL(func(tx *gorm.DB) *gorm.DB {
		tx = tx.Model(&model.User{})
		tx = ApplyFilters(tx, []domainQuery.Filter{
			{Column: "name", Operator: domainQuery.OperatorLike, Values: []string{"alice"}},
		}, allowed, "")
		var out []model.User
		return tx.Find(&out)
	})
	assert.Contains(t, sql, "LOWER(name) LIKE LOWER")
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd CredChain_Golang && go test ./infrastructure/database/gorm/... -run TestApplyFilters -v -count 1
```

Expected: compilation error — `undefined: ApplyFilters`

- [ ] **Step 3: Create `helpers.go` with `ApplyFilters` implementation**

Write to `infrastructure/database/gorm/helpers.go`:

```go
package gorm

import (
	"strings"

	domainQuery "CredChain_Golang/domain/query"

	"gorm.io/gorm"
)

func ApplyFilters(db *gorm.DB, filters []domainQuery.Filter, allowedColumns map[string]bool, columnPrefix string) *gorm.DB {
	for _, f := range filters {
		if !allowedColumns[f.Column] {
			continue
		}
		col := columnPrefix + f.Column
		switch f.Operator {
		case domainQuery.OperatorEqual:
			db = db.Where(col+" = ?", f.GetValue())
		case domainQuery.OperatorNotEqual:
			db = db.Where(col+" != ?", f.GetValue())
		case domainQuery.OperatorGreaterThan:
			db = db.Where(col+" > ?", f.GetValue())
		case domainQuery.OperatorLessThan:
			db = db.Where(col+" < ?", f.GetValue())
		case domainQuery.OperatorGreaterThanOrEqual:
			db = db.Where(col+" >= ?", f.GetValue())
		case domainQuery.OperatorLessThanOrEqual:
			db = db.Where(col+" <= ?", f.GetValue())
		case domainQuery.OperatorLike, domainQuery.OperatorILike:
			db = db.Where("LOWER("+col+") LIKE LOWER(?)", "%"+f.GetValue()+"%")
		case domainQuery.OperatorNotLike, domainQuery.OperatorNotILike:
			db = db.Where("LOWER("+col+") NOT LIKE LOWER(?)", "%"+f.GetValue()+"%")
		case domainQuery.OperatorIn:
			if len(f.Values) > 0 {
				db = db.Where(col+" IN ?", f.Values)
			}
		case domainQuery.OperatorNotIn:
			if len(f.Values) > 0 {
				db = db.Where(col+" NOT IN ?", f.Values)
			}
		case domainQuery.OperatorBetween:
			if len(f.Values) == 2 {
				db = db.Where(col+" BETWEEN ? AND ?", f.Values[0], f.Values[1])
			}
		case domainQuery.OperatorNotBetween:
			if len(f.Values) == 2 {
				db = db.Where(col+" NOT BETWEEN ? AND ?", f.Values[0], f.Values[1])
			}
		case domainQuery.OperatorNull:
			db = db.Where(col + " IS NULL")
		case domainQuery.OperatorNotNull:
			db = db.Where(col + " IS NOT NULL")
		}
	}
	return db
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd CredChain_Golang && go test ./infrastructure/database/gorm/... -run TestApplyFilters -v -count 1
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd CredChain_Golang
git add infrastructure/database/gorm/helpers.go infrastructure/database/gorm/helpers_test.go
git commit -m "feat: add ApplyFilters shared helper with GORM operator mapping"
```

---

### Task 2: Add `ApplySorts` helper

**Files:**
- Modify: `infrastructure/database/gorm/helpers_test.go` — add tests
- Modify: `infrastructure/database/gorm/helpers.go` — add function

- [ ] **Step 1: Write the failing test for `ApplySorts`**

Append to `infrastructure/database/gorm/helpers_test.go`:

```go
func TestApplySorts_WithSorts(t *testing.T) {
	gdb := db.OpenInMemorySQLite(t)
	allowed := map[string]bool{"id": true, "name": true}

	sql := gdb.ToSQL(func(tx *gorm.DB) *gorm.DB {
		tx = tx.Model(&model.User{})
		tx = ApplySorts(tx, &domainQuery.Query{
			Sorts: []domainQuery.Sort{
				{Column: "name", Order: domainQuery.SortAsc},
			},
		}, allowed, "created_at DESC", nil)
		var out []model.User
		return tx.Find(&out)
	})
	assert.Contains(t, sql, "ORDER BY name ASC")
}

func TestApplySorts_DefaultSort(t *testing.T) {
	gdb := db.OpenInMemorySQLite(t)
	allowed := map[string]bool{"id": true}

	sql := gdb.ToSQL(func(tx *gorm.DB) *gorm.DB {
		tx = tx.Model(&model.User{})
		tx = ApplySorts(tx, &domainQuery.Query{
			Sorts: []domainQuery.Sort{},
		}, allowed, "updated_at DESC", nil)
		var out []model.User
		return tx.Find(&out)
	})
	assert.Contains(t, sql, "ORDER BY updated_at DESC")
}

func TestApplySorts_WithColumnMapper(t *testing.T) {
	gdb := db.OpenInMemorySQLite(t)
	allowed := map[string]bool{"holder_name": true}

	mapper := func(col string) string {
		if col == "holder_name" {
			return "holder.name"
		}
		return col
	}

	sql := gdb.ToSQL(func(tx *gorm.DB) *gorm.DB {
		tx = tx.Model(&model.User{})
		tx = ApplySorts(tx, &domainQuery.Query{
			Sorts: []domainQuery.Sort{
				{Column: "holder_name", Order: domainQuery.SortDesc},
			},
		}, allowed, "issued_at DESC", mapper)
		var out []model.User
		return tx.Find(&out)
	})
	assert.Contains(t, sql, "ORDER BY holder.name DESC")
}

func TestApplySorts_NotInAllowlist(t *testing.T) {
	gdb := db.OpenInMemorySQLite(t)
	allowed := map[string]bool{"id": true}

	sql := gdb.ToSQL(func(tx *gorm.DB) *gorm.DB {
		tx = tx.Model(&model.User{})
		tx = ApplySorts(tx, &domainQuery.Query{
			Sorts: []domainQuery.Sort{
				{Column: "secret", Order: domainQuery.SortDesc},
			},
		}, allowed, "updated_at DESC", nil)
		var out []model.User
		return tx.Find(&out)
	})
	assert.NotContains(t, sql, "secret")
	assert.Contains(t, sql, "updated_at DESC")
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd CredChain_Golang && go test ./infrastructure/database/gorm/... -run TestApplySorts -v -count 1
```

Expected: compilation error — `undefined: ApplySorts`

- [ ] **Step 3: Add `ApplySorts` to `helpers.go`**

Append to `infrastructure/database/gorm/helpers.go`:

```go
func ApplySorts(db *gorm.DB, q *domainQuery.Query, allowedColumns map[string]bool, defaultSort string, mapper func(string) string) *gorm.DB {
	if q.HasSorts() {
		for _, s := range q.Sorts {
			if !allowedColumns[s.Column] {
				continue
			}
			col := s.Column
			if mapper != nil {
				col = mapper(col)
			}
			db = db.Order(col + " " + string(s.Order))
		}
	} else {
		db = db.Order(defaultSort)
	}
	return db
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd CredChain_Golang && go test ./infrastructure/database/gorm/... -run TestApplySorts -v -count 1
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd CredChain_Golang
git add infrastructure/database/gorm/helpers.go infrastructure/database/gorm/helpers_test.go
git commit -m "feat: add ApplySorts shared helper with allowlist, mapper, and default sort"
```

---

### Task 3: Add `ApplyPagination` helper

**Files:**
- Modify: `infrastructure/database/gorm/helpers_test.go` — add tests
- Modify: `infrastructure/database/gorm/helpers.go` — add function

- [ ] **Step 1: Write the failing test for `ApplyPagination`**

Append to `infrastructure/database/gorm/helpers_test.go`:

```go
func TestApplyPagination_WithPagination(t *testing.T) {
	gdb := db.OpenInMemorySQLite(t)

	sql := gdb.ToSQL(func(tx *gorm.DB) *gorm.DB {
		tx = tx.Model(&model.User{})
		tx = ApplyPagination(tx, &domainQuery.Query{Page: 2, Limit: 20})
		var out []model.User
		return tx.Find(&out)
	})
	assert.Contains(t, sql, "LIMIT 20")
}

func TestApplyPagination_NoPagination(t *testing.T) {
	gdb := db.OpenInMemorySQLite(t)

	sql := gdb.ToSQL(func(tx *gorm.DB) *gorm.DB {
		tx = tx.Model(&model.User{})
		tx = ApplyPagination(tx, &domainQuery.Query{})
		var out []model.User
		return tx.Find(&out)
	})
	assert.NotContains(t, sql, "LIMIT")
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd CredChain_Golang && go test ./infrastructure/database/gorm/... -run TestApplyPagination -v -count 1
```

Expected: compilation error — `undefined: ApplyPagination`

- [ ] **Step 3: Add `ApplyPagination` to `helpers.go`**

Append to `infrastructure/database/gorm/helpers.go`:

```go
func ApplyPagination(db *gorm.DB, q *domainQuery.Query) *gorm.DB {
	if q.HasPagination() {
		return db.Limit(q.Limit).Offset(q.Offset())
	}
	return db
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd CredChain_Golang && go test ./infrastructure/database/gorm/... -run TestApplyPagination -v -count 1
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd CredChain_Golang
git add infrastructure/database/gorm/helpers.go infrastructure/database/gorm/helpers_test.go
git commit -m "feat: add ApplyPagination shared helper"
```

---

### Task 4: Add `BuildCaseColumnSQL` helper

**Files:**
- Modify: `infrastructure/database/gorm/helpers_test.go` — add tests
- Modify: `infrastructure/database/gorm/helpers.go` — add function

- [ ] **Step 1: Write the failing test for `BuildCaseColumnSQL`**

Append to `infrastructure/database/gorm/helpers_test.go`:

```go
func TestBuildCaseColumnSQL_SinglePair(t *testing.T) {
	clause, args := BuildCaseColumnSQL("id", "name", []interface{}{"user-1", "alice"})
	assert.Equal(t, "name = CASE id WHEN ? THEN ? ELSE name END", clause)
	assert.Equal(t, []interface{}{"user-1", "alice"}, args)
}

func TestBuildCaseColumnSQL_MultiplePairs(t *testing.T) {
	clause, args := BuildCaseColumnSQL("id", "role", []interface{}{
		"user-1", "admin",
		"user-2", "holder",
		"user-3", "issuer",
	})
	assert.Equal(t, "role = CASE id WHEN ? THEN ? WHEN ? THEN ? WHEN ? THEN ? ELSE role END", clause)
	assert.Equal(t, []interface{}{"user-1", "admin", "user-2", "holder", "user-3", "issuer"}, args)
}

func TestBuildCaseColumnSQL_EmptyPairs(t *testing.T) {
	clause, args := BuildCaseColumnSQL("id", "name", []interface{}{})
	assert.Equal(t, "", clause)
	assert.Nil(t, args)
}

func TestBuildCaseColumnSQL_CustomIDColumn(t *testing.T) {
	clause, args := BuildCaseColumnSQL("user_id", "role", []interface{}{"user-1", "admin"})
	assert.Contains(t, clause, "CASE user_id")
	assert.Contains(t, clause, "ELSE role END")
	assert.Equal(t, []interface{}{"user-1", "admin"}, args)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd CredChain_Golang && go test ./infrastructure/database/gorm/... -run TestBuildCaseColumnSQL -v -count 1
```

Expected: compilation error — `undefined: BuildCaseColumnSQL`

- [ ] **Step 3: Add `BuildCaseColumnSQL` to `helpers.go`**

Append to `infrastructure/database/gorm/helpers.go`:

```go
func BuildCaseColumnSQL(idColumn, col string, pairs []interface{}) (string, []interface{}) {
	if len(pairs) == 0 {
		return "", nil
	}
	var b strings.Builder
	b.WriteString("CASE ")
	b.WriteString(idColumn)
	for i := 0; i < len(pairs)/2; i++ {
		b.WriteString(" WHEN ? THEN ?")
	}
	b.WriteString(" ELSE ")
	b.WriteString(col)
	b.WriteString(" END")
	return col + " = " + b.String(), pairs
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd CredChain_Golang && go test ./infrastructure/database/gorm/... -run TestBuildCaseColumnSQL -v -count 1
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd CredChain_Golang
git add infrastructure/database/gorm/helpers.go infrastructure/database/gorm/helpers_test.go
git commit -m "feat: add BuildCaseColumnSQL helper for batch CASE UPDATE clauses"
```

---

### Task 5: Add `BuildBatchUpdateSQL` helper

**Files:**
- Modify: `infrastructure/database/gorm/helpers_test.go` — add tests
- Modify: `infrastructure/database/gorm/helpers.go` — add function

- [ ] **Step 1: Write the failing test for `BuildBatchUpdateSQL`**

Append to `infrastructure/database/gorm/helpers_test.go`:

```go
func TestBuildBatchUpdateSQL_WithExtraClauses(t *testing.T) {
	clause1, args1 := BuildCaseColumnSQL("id", "name", []interface{}{"user-1", "alice"})
	clause2, args2 := BuildCaseColumnSQL("id", "role", []interface{}{"user-1", "admin"})

	sql, finalArgs := BuildBatchUpdateSQL("users", "id",
		[]string{clause1, clause2},
		[][]interface{}{args1, args2},
		[]interface{}{"user-1", "user-2"},
		"updated_at = CURRENT_TIMESTAMP",
	)

	assert.Contains(t, sql, "UPDATE users SET")
	assert.Contains(t, sql, "name = CASE id WHEN ? THEN ? ELSE name END")
	assert.Contains(t, sql, "role = CASE id WHEN ? THEN ? ELSE role END")
	assert.Contains(t, sql, "updated_at = CURRENT_TIMESTAMP")
	assert.Contains(t, sql, "WHERE id IN (?)")
	assert.Equal(t, []interface{}{"user-1", "alice", "user-1", "admin", "user-1", "user-2"}, finalArgs)
}

func TestBuildBatchUpdateSQL_NoExtraClauses(t *testing.T) {
	clause, args := BuildCaseColumnSQL("id", "role", []interface{}{"id-1", "holder"})

	sql, finalArgs := BuildBatchUpdateSQL("credentials", "id",
		[]string{clause},
		[][]interface{}{args},
		[]interface{}{"id-1"},
	)

	assert.Contains(t, sql, "UPDATE credentials SET")
	assert.Contains(t, sql, "role = CASE id WHEN ? THEN ? ELSE role END")
	assert.Contains(t, sql, "WHERE id IN (?)")
	assert.Equal(t, []interface{}{"id-1", "holder", "id-1"}, finalArgs)
}

func TestBuildBatchUpdateSQL_EmptyClauses(t *testing.T) {
	sql, finalArgs := BuildBatchUpdateSQL("users", "id",
		[]string{},
		[][]interface{}{},
		[]interface{}{"id-1"},
	)

	assert.Contains(t, sql, "UPDATE users SET")
	assert.Contains(t, sql, "WHERE id IN (?)")
	assert.Equal(t, []interface{}{"id-1"}, finalArgs)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd CredChain_Golang && go test ./infrastructure/database/gorm/... -run TestBuildBatchUpdateSQL -v -count 1
```

Expected: compilation error — `undefined: BuildBatchUpdateSQL`

- [ ] **Step 3: Add `BuildBatchUpdateSQL` to `helpers.go`**

Append to `infrastructure/database/gorm/helpers.go`:

```go
func BuildBatchUpdateSQL(table, idColumn string, setClauses []string, setArgs [][]interface{}, ids []interface{}, extraSetClauses ...string) (string, []interface{}) {
	var allArgs []interface{}
	for _, a := range setArgs {
		allArgs = append(allArgs, a...)
	}
	allClauses := make([]string, len(setClauses))
	copy(allClauses, setClauses)
	allClauses = append(allClauses, extraSetClauses...)
	sql := "UPDATE " + table + " SET " + strings.Join(allClauses, ", ") + " WHERE " + idColumn + " IN (?)"
	allArgs = append(allArgs, ids)
	return sql, allArgs
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd CredChain_Golang && go test ./infrastructure/database/gorm/... -run TestBuildBatchUpdateSQL -v -count 1
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd CredChain_Golang
git add infrastructure/database/gorm/helpers.go infrastructure/database/gorm/helpers_test.go
git commit -m "feat: add BuildBatchUpdateSQL helper for assembling batch UPDATE statements"
```

---

### Task 6: Refactor `gorm_user_repository.go` — `Get()` (filters, sorts, pagination, soft-delete, default sort)

**Files:**
- Modify: `feature/user/gorm_user_repository.go` — replace `Get()` method, remove `referencesDeletedAt`, remove `applyUserFilters`

- [ ] **Step 1: Run existing tests to establish baseline**

```bash
cd CredChain_Golang && go test ./feature/user/... -v -count 1
```

Expected: All existing tests pass (baseline)

- [ ] **Step 2: Replace `Get()` and remove `referencesDeletedAt` and `applyUserFilters`**

In `feature/user/gorm_user_repository.go`:

**2a. Update imports** — add `gormhelpers "CredChain_Golang/infrastructure/database/gorm"` and remove `"github.com/samber/lo"` and `"strings"`.

New imports block:
```go
import (
	"context"
	"encoding/json"
	"errors"
	"sort"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	gormhelpers "CredChain_Golang/infrastructure/database/gorm"
	"CredChain_Golang/infrastructure/database/gorm/model"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/oklog/ulid/v2"
	"gorm.io/gorm"
)
```

**2b. Delete** lines 62-68 — the entire `referencesDeletedAt` function:
```go
func referencesDeletedAt(q *domainQuery.Query) bool {
	return lo.ContainsBy(q.Filters, func(f domainQuery.Filter) bool { return f.Column == "deleted_at" }) ||
		lo.ContainsBy(q.Sorts, func(s domainQuery.Sort) bool { return s.Column == "deleted_at" })
}
```

**2c. Delete** lines 120-172 — the entire `applyUserFilters` function.

**2d. Replace** the `Get()` method (lines 72-118) with:

```go
func (r *gormUserRepository) Get(ctx context.Context, query *domainQuery.Query) ([]domain.User, int, error) {
	db := r.db.WithContext(ctx).Unscoped().Model(&model.User{})

	if query.HasSearch() {
		db = db.Where("LOWER(name) LIKE LOWER(?) OR LOWER(email) LIKE LOWER(?)",
			"%"+query.Search+"%", "%"+query.Search+"%")
	}

	if query.HasFilters() {
		db = gormhelpers.ApplyFilters(db, query.Filters, allowedFilterColumns, "")
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	db = gormhelpers.ApplySorts(db, query, allowedSortColumns, "updated_at DESC", nil)
	db = gormhelpers.ApplyPagination(db, query)

	var users []model.User
	if err := db.Find(&users).Error; err != nil {
		return nil, 0, err
	}

	domainUsers := make([]domain.User, len(users))
	for i, u := range users {
		domainUsers[i] = u.ToDomain()
	}

	return domainUsers, int(total), nil
}
```

- [ ] **Step 3: Run tests to verify refactoring is correct**

```bash
cd CredChain_Golang && go test ./feature/user/... -v -count 1
```

Expected: All existing tests pass (behavior preserved, only wiring changed)

- [ ] **Step 4: Commit**

```bash
cd CredChain_Golang
git add feature/user/gorm_user_repository.go
git commit -m "refactor(user): use shared helpers in Get(), always unscoped, default sort updated_at DESC"
```

---

### Task 7: Refactor `gorm_user_repository.go` — `updateBatchCase` and `UpdateRole`

**Files:**
- Modify: `feature/user/gorm_user_repository.go` — replace `updateBatchCase`, `UpdateRole`

- [ ] **Step 1: Replace `updateBatchCase` method (lines 256-360)**

Delete the entire `updateBatchCase` method and replace with:

```go
func (r *gormUserRepository) updateBatchCase(ctx context.Context, users []domain.User) error {
	sort.Slice(users, func(i, j int) bool { return users[i].Id < users[j].Id })

	var clauses []string
	var allArgs [][]interface{}

	addCol := func(col string, getValue func(domain.User) (interface{}, bool)) {
		var pairs []interface{}
		for _, u := range users {
			if v, ok := getValue(u); ok {
				pairs = append(pairs, u.Id, v)
			}
		}
		if clause, args := gormhelpers.BuildCaseColumnSQL("id", col, pairs); clause != "" {
			clauses = append(clauses, clause)
			allArgs = append(allArgs, args)
		}
	}

	addCol("name", func(u domain.User) (interface{}, bool) {
		if u.Name != nil {
			return *u.Name, true
		}
		return nil, false
	})
	addCol("number", func(u domain.User) (interface{}, bool) {
		if u.Number != nil {
			return *u.Number, true
		}
		return nil, false
	})
	addCol("phone_number", func(u domain.User) (interface{}, bool) {
		if u.PhoneNumber != nil {
			return *u.PhoneNumber, true
		}
		return nil, false
	})
	addCol("email", func(u domain.User) (interface{}, bool) {
		if u.Email != "" {
			return u.Email, true
		}
		return nil, false
	})
	addCol("birth_date", func(u domain.User) (interface{}, bool) {
		if u.BirthDate != nil {
			return *u.BirthDate, true
		}
		return nil, false
	})
	addCol("gender", func(u domain.User) (interface{}, bool) {
		if u.Gender != nil {
			return string(*u.Gender), true
		}
		return nil, false
	})
	addCol("meta", func(u domain.User) (interface{}, bool) {
		if u.Meta == nil {
			return nil, false
		}
		b, err := json.Marshal(u.Meta)
		if err != nil {
			return nil, false
		}
		return string(b), true
	})
	addCol("role", func(u domain.User) (interface{}, bool) {
		if u.Role != "" {
			return string(u.Role), true
		}
		return nil, false
	})
	addCol("wallet_address", func(u domain.User) (interface{}, bool) {
		if u.WalletAddress != "" {
			return u.WalletAddress, true
		}
		return nil, false
	})
	addCol("encrypted_wallet_private_key", func(u domain.User) (interface{}, bool) {
		if u.EncryptedWalletPrivateKey != "" {
			return u.EncryptedWalletPrivateKey, true
		}
		return nil, false
	})

	if len(clauses) == 0 {
		return nil
	}

	ids := make([]interface{}, len(users))
	for i, u := range users {
		ids[i] = u.Id
	}
	sql, finalArgs := gormhelpers.BuildBatchUpdateSQL("users", "id", clauses, allArgs, ids, "updated_at = CURRENT_TIMESTAMP")
	return r.db.WithContext(ctx).Exec(sql, finalArgs...).Error
}
```

- [ ] **Step 2: Replace `UpdateRole` (lines 362-405)**

Replace the current `UpdateRole` with:

```go
func (r *gormUserRepository) UpdateRole(ctx context.Context, users ...domain.User) ([]domain.User, int64, error) {
	if len(users) == 0 {
		return []domain.User{}, 0, nil
	}

	var pairs []interface{}
	for _, user := range users {
		pairs = append(pairs, user.Id, user.Role)
	}
	clause, clauseArgs := gormhelpers.BuildCaseColumnSQL("id", "role", pairs)

	ids := make([]interface{}, len(users))
	for i, user := range users {
		ids[i] = user.Id
	}

	sql, finalArgs := gormhelpers.BuildBatchUpdateSQL("users", "id", []string{clause}, [][]interface{}{clauseArgs}, ids)

	result := r.db.WithContext(ctx).Exec(sql, finalArgs...)
	if err := result.Error; err != nil {
		return nil, 0, err
	}

	userIDs := make([]string, len(users))
	for i, user := range users {
		userIDs[i] = user.Id
	}
	updatedUsers, err := r.FindByIds(ctx, userIDs...)
	if err != nil {
		return nil, 0, err
	}

	return updatedUsers, result.RowsAffected, nil
}
```

- [ ] **Step 3: Run tests to verify refactoring is correct**

```bash
cd CredChain_Golang && go test ./feature/user/... -v -count 1
```

Expected: All existing tests pass

- [ ] **Step 4: Verify unused imports are removed**

```bash
cd CredChain_Golang && go vet ./feature/user/...
```

Expected: No errors

- [ ] **Step 5: Commit**

```bash
cd CredChain_Golang
git add feature/user/gorm_user_repository.go
git commit -m "refactor(user): use shared BUILD CASE helpers in updateBatchCase and UpdateRole"
```

---

### Task 8: Refactor `gorm_credential_repository.go` — `Get()` (filters, sorts, pagination)

**Files:**
- Modify: `feature/credential/gorm_credential_repository.go` — replace `Get()`, remove `applyCredentialFilters`

- [ ] **Step 1: Run existing tests to establish baseline**

```bash
cd CredChain_Golang && go test ./feature/credential/... -v -count 1
```

Expected: All existing tests pass (baseline — credential feature has stub tests only)

- [ ] **Step 2: Add import and remove `applyCredentialFilters`**

**2a. Update imports** — add `gormhelpers "CredChain_Golang/infrastructure/database/gorm"` and remove `"fmt"`.

New imports block:
```go
import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	gormhelpers "CredChain_Golang/infrastructure/database/gorm"
	"CredChain_Golang/infrastructure/database/gorm/model"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"gorm.io/gorm"
)
```

**2b. Delete** lines 172-221 — the entire `applyCredentialFilters` function.

**2c. Replace** the `Get()` method (lines 113-170) with:

```go
func (r *gormCredentialRepository) Get(ctx context.Context, query *domainQuery.Query) ([]domain.Credential, int, error) {
	db := r.db.WithContext(ctx).Model(&model.Credential{})

	if needsHolderJoin(query) {
		db = db.Joins("LEFT JOIN users AS holder ON holder.id = credentials.holder_user_id")
	}

	if query.HasSearch() {
		needle := "%" + query.Search + "%"
		db = db.Where(
			"LOWER(credentials.name) LIKE LOWER(?) OR "+
				"LOWER(CAST(credentials.meta AS TEXT)) LIKE LOWER(?) OR "+
				"LOWER(holder.name) LIKE LOWER(?) OR "+
				"LOWER(holder.email) LIKE LOWER(?) OR "+
				"LOWER(holder.number) LIKE LOWER(?) OR "+
				"LOWER(holder.phone_number) LIKE LOWER(?)",
			needle, needle, needle, needle, needle, needle,
		)
	}

	if query.HasFilters() {
		db = gormhelpers.ApplyFilters(db, query.Filters, allowedFilterColumns, "credentials.")
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	db = gormhelpers.ApplySorts(db, query, allowedSortColumns, "credentials.issued_at DESC", mapSortColumn)
	db = preloadByIncludes(db, query)
	db = gormhelpers.ApplyPagination(db, query)

	var credentials []model.Credential
	if err := db.Find(&credentials).Error; err != nil {
		return nil, 0, err
	}

	out := make([]domain.Credential, len(credentials))
	for i, c := range credentials {
		out[i] = c.ToDomain()
	}
	return out, int(total), nil
}
```

- [ ] **Step 3: Run tests to verify refactoring is correct**

```bash
cd CredChain_Golang && go test ./feature/credential/... -v -count 1
```

Expected: All tests pass (same as baseline)

- [ ] **Step 4: Commit**

```bash
cd CredChain_Golang
git add feature/credential/gorm_credential_repository.go
git commit -m "refactor(credential): use shared filter, sort, pagination helpers in Get()"
```

---

### Task 9: Refactor `gorm_credential_repository.go` — `updateBatchCase`

**Files:**
- Modify: `feature/credential/gorm_credential_repository.go` — replace `updateBatchCase`

- [ ] **Step 1: Replace `updateBatchCase` method (lines 349-443)**

Delete the entire `updateBatchCase` method and replace with:

```go
func (r *gormCredentialRepository) updateBatchCase(ctx context.Context, items []domain.Credential) error {
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })

	var clauses []string
	var allArgs [][]interface{}

	addCol := func(col string, getValue func(domain.Credential) (interface{}, bool)) {
		var pairs []interface{}
		for _, c := range items {
			if v, ok := getValue(c); ok {
				pairs = append(pairs, c.ID, v)
			}
		}
		if clause, args := gormhelpers.BuildCaseColumnSQL("id", col, pairs); clause != "" {
			clauses = append(clauses, clause)
			allArgs = append(allArgs, args)
		}
	}

	addCol("name", func(c domain.Credential) (interface{}, bool) {
		if c.Name != "" {
			return c.Name, true
		}
		return nil, false
	})
	addCol("meta", func(c domain.Credential) (interface{}, bool) {
		if c.Meta == nil {
			return nil, false
		}
		b, err := json.Marshal(c.Meta)
		if err != nil {
			return nil, false
		}
		return string(b), true
	})
	addCol("token_id", func(c domain.Credential) (interface{}, bool) {
		if c.TokenID == nil {
			return nil, false
		}
		return *c.TokenID, true
	})
	addCol("file_uri", func(c domain.Credential) (interface{}, bool) {
		if c.FileURI == nil {
			return nil, false
		}
		return *c.FileURI, true
	})
	addCol("revoked_at", func(c domain.Credential) (interface{}, bool) {
		if c.RevokedAt == nil {
			return nil, false
		}
		return *c.RevokedAt, true
	})
	addCol("revoker_user_id", func(c domain.Credential) (interface{}, bool) {
		if c.RevokerUserID == nil {
			return nil, false
		}
		return *c.RevokerUserID, true
	})
	addCol("extract_status", func(c domain.Credential) (interface{}, bool) {
		if c.ExtractStatus == "" {
			return nil, false
		}
		return string(c.ExtractStatus), true
	})
	addCol("extract_error", func(c domain.Credential) (interface{}, bool) {
		if c.ExtractError == nil {
			return nil, false
		}
		return *c.ExtractError, true
	})
	addCol("extracted_at", func(c domain.Credential) (interface{}, bool) {
		if c.ExtractedAt == nil {
			return nil, false
		}
		return *c.ExtractedAt, true
	})

	if len(clauses) == 0 {
		return nil
	}

	ids := make([]interface{}, len(items))
	for i, c := range items {
		ids[i] = c.ID
	}
	sql, finalArgs := gormhelpers.BuildBatchUpdateSQL("credentials", "id", clauses, allArgs, ids)
	return r.db.WithContext(ctx).Exec(sql, finalArgs...).Error
}
```

- [ ] **Step 2: Run tests to verify refactoring is correct**

```bash
cd CredChain_Golang && go test ./feature/credential/... -v -count 1
```

Expected: All tests pass (same as baseline)

- [ ] **Step 3: Verify unused imports are removed**

```bash
cd CredChain_Golang && go vet ./feature/credential/...
```

Expected: No errors

- [ ] **Step 4: Commit**

```bash
cd CredChain_Golang
git add feature/credential/gorm_credential_repository.go
git commit -m "refactor(credential): use shared BUILD CASE helpers in updateBatchCase"
```

---

### Task 10: Final verification

**Files:** All modified files

- [ ] **Step 1: Run all tests**

```bash
cd CredChain_Golang && go test ./... -count 1
```

Expected: All tests pass

- [ ] **Step 2: Run all helpers tests specifically**

```bash
cd CredChain_Golang && go test ./infrastructure/database/gorm/... -v -count 1
```

Expected: All helper tests pass

- [ ] **Step 3: Run vet and format check**

```bash
cd CredChain_Golang && go vet ./... && gofmt -l .
```

Expected: Zero output from both commands

- [ ] **Step 4: Commit final verification**

```bash
cd CredChain_Golang && git add -A && git commit -m "chore: final vet and fmt after shared helpers refactoring"
```
