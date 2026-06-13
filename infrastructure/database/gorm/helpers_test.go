package gorm

import (
	"testing"

	domainQuery "CredChain_Golang/domain/query"
	"CredChain_Golang/infrastructure/database/gorm/model"
	"CredChain_Golang/infrastructure/testutil/db"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
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
