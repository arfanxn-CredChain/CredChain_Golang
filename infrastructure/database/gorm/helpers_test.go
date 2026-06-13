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
