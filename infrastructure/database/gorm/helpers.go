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

func ApplySorts(db *gorm.DB, q *domainQuery.Query, allowedColumns map[string]bool, defaultSort string, mapper func(string) string, tiebreaker string) *gorm.DB {
	if q == nil {
		db = db.Order(defaultSort)
		if tiebreaker != "" {
			db = db.Order(tiebreaker)
		}
		return db
	}
	appliedAny := false
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
			appliedAny = true
		}
	}
	if !appliedAny {
		db = db.Order(defaultSort)
	}
	if tiebreaker != "" {
		db = db.Order(tiebreaker)
	}
	return db
}

func ApplyPagination(db *gorm.DB, q *domainQuery.Query) *gorm.DB {
	if q == nil {
		return db
	}
	if q.HasPagination() {
		return db.Limit(q.Limit).Offset(q.Offset())
	}
	return db
}

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
