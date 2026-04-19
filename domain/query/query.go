package query

import "strings"

type Filter struct {
	Column   string
	Operator Operator
	Values   []string
}

func (f *Filter) GetValue() string {
	if len(f.Values) > 0 {
		return f.Values[0]
	}
	return ""
}

type Sort struct {
	Column string
	Order  SortOrder
}

type Query struct {
	Page     int
	Limit    int
	Search   string
	Filters  []Filter
	Sorts    []Sort
	Includes []string
	Groups   []string
}

func NewQuery() Query {
	return Query{
		Page:     1,
		Limit:    10,
		Filters:  []Filter{},
		Sorts:    []Sort{},
		Includes: []string{},
		Groups:   []string{},
	}
}

func (q *Query) Offset() int {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 {
		q.Limit = 10
	}
	return (q.Page - 1) * q.Limit
}

func (q *Query) HasFilters() bool    { return len(q.Filters) > 0 }
func (q *Query) HasSorts() bool      { return len(q.Sorts) > 0 }
func (q *Query) HasSearch() bool     { return q.Search != "" }
func (q *Query) HasPagination() bool { return q.Page > 0 || q.Limit > 0 }
func (q *Query) HasIncludes() bool   { return len(q.Includes) > 0 }
func (q *Query) HasGroups() bool     { return len(q.Groups) > 0 }

func (q *Query) GetFilter(column string, operator Operator) *Filter {
	columnLower := strings.ToLower(column)
	for i := range q.Filters {
		f := &q.Filters[i]
		if strings.ToLower(f.Column) == columnLower && f.Operator == operator {
			return f
		}
	}
	return nil
}

func (q *Query) GetSort(column string) *Sort {
	columnLower := strings.ToLower(column)
	for i := range q.Sorts {
		s := &q.Sorts[i]
		if strings.ToLower(s.Column) == columnLower {
			return s
		}
	}
	return nil
}

func (q *Query) GetInclude(name string) *string {
	nameLower := strings.ToLower(name)
	for i := range q.Includes {
		if strings.ToLower(q.Includes[i]) == nameLower {
			return &q.Includes[i]
		}
	}
	return nil
}

func (q *Query) GetGroup(name string) *string {
	nameLower := strings.ToLower(name)
	for i := range q.Groups {
		if strings.ToLower(q.Groups[i]) == nameLower {
			return &q.Groups[i]
		}
	}
	return nil
}
