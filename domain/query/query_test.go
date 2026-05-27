package query

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewQuery_Defaults(t *testing.T) {
	q := NewQuery()
	assert.Equal(t, 1, q.Page)
	assert.Equal(t, 10, q.Limit)
	assert.Empty(t, q.Filters)
	assert.Empty(t, q.Sorts)
	assert.Empty(t, q.Includes)
	assert.Empty(t, q.Groups)
}

func TestQuery_Offset(t *testing.T) {
	q := Query{Page: 3, Limit: 20}
	assert.Equal(t, 40, q.Offset())
}

func TestQuery_Offset_NormalizesInvalid(t *testing.T) {
	q := Query{Page: 0, Limit: 0}
	assert.Equal(t, 0, q.Offset())
	assert.Equal(t, 1, q.Page)
	assert.Equal(t, 10, q.Limit)
}

func TestQuery_HasFilters(t *testing.T) {
	q1 := Query{}
	assert.False(t, q1.HasFilters())
	q2 := Query{Filters: []Filter{{Column: "name"}}}
	assert.True(t, q2.HasFilters())
}

func TestQuery_HasSorts(t *testing.T) {
	q1 := Query{}
	assert.False(t, q1.HasSorts())
	q2 := Query{Sorts: []Sort{{Column: "name"}}}
	assert.True(t, q2.HasSorts())
}

func TestQuery_HasSearch(t *testing.T) {
	q1 := Query{}
	assert.False(t, q1.HasSearch())
	q2 := Query{Search: "x"}
	assert.True(t, q2.HasSearch())
}

func TestQuery_HasPagination(t *testing.T) {
	q1 := Query{}
	assert.False(t, q1.HasPagination())
	q2 := Query{Page: 1}
	assert.True(t, q2.HasPagination())
	q3 := Query{Limit: 10}
	assert.True(t, q3.HasPagination())
}

func TestQuery_HasIncludes(t *testing.T) {
	assert.False(t, (&Query{}).HasIncludes())
	assert.True(t, (&Query{Includes: []string{"x"}}).HasIncludes())
}

func TestQuery_HasGroups(t *testing.T) {
	assert.False(t, (&Query{}).HasGroups())
	assert.True(t, (&Query{Groups: []string{"x"}}).HasGroups())
}

func TestQuery_GetFilter(t *testing.T) {
	op := Operator("=")
	q := Query{Filters: []Filter{
		{Column: "Email", Operator: op, Values: []string{"a@x"}},
	}}
	got := q.GetFilter("email", op)
	assert.NotNil(t, got)
	assert.Equal(t, "Email", got.Column)
	assert.Nil(t, q.GetFilter("name", op))
}

func TestQuery_GetSort(t *testing.T) {
	q := Query{Sorts: []Sort{{Column: "Name", Order: SortAsc}}}
	got := q.GetSort("name")
	assert.NotNil(t, got)
	assert.Nil(t, q.GetSort("missing"))
}

func TestFilter_GetValue_Empty(t *testing.T) {
	f := Filter{}
	assert.Equal(t, "", f.GetValue())
}

func TestFilter_GetValue_FirstValue(t *testing.T) {
	f := Filter{Values: []string{"a", "b"}}
	assert.Equal(t, "a", f.GetValue())
}
