package query

import (
	"testing"

	domainQuery "CredChain_Golang/domain/query"

	"github.com/stretchr/testify/assert"
)

func TestParseFilterString_Equals(t *testing.T) {
	col, op, vals, err := parseFilterString("name=alice")
	assert.NoError(t, err)
	assert.Equal(t, "name", col)
	assert.Equal(t, domainQuery.OperatorEqual, op)
	assert.Equal(t, []string{"alice"}, vals)
}

func TestParseFilterString_NotEquals(t *testing.T) {
	col, op, _, err := parseFilterString("name!=bob")
	assert.NoError(t, err)
	assert.Equal(t, "name", col)
	assert.Equal(t, domainQuery.OperatorNotEqual, op)
}

func TestParseFilterString_GreaterThan(t *testing.T) {
	_, op, _, err := parseFilterString("age>10")
	assert.NoError(t, err)
	assert.Equal(t, domainQuery.OperatorGreaterThan, op)
}

func TestParseFilterString_LessThan(t *testing.T) {
	_, op, _, err := parseFilterString("age<10")
	assert.NoError(t, err)
	assert.Equal(t, domainQuery.OperatorLessThan, op)
}

func TestParseFilterString_GreaterThanOrEqual(t *testing.T) {
	_, op, _, err := parseFilterString("age>=10")
	assert.NoError(t, err)
	assert.Equal(t, domainQuery.OperatorGreaterThanOrEqual, op)
}

func TestParseFilterString_LessThanOrEqual(t *testing.T) {
	_, op, _, err := parseFilterString("age<=10")
	assert.NoError(t, err)
	assert.Equal(t, domainQuery.OperatorLessThanOrEqual, op)
}

func TestParseFilterString_Like(t *testing.T) {
	col, op, vals, err := parseFilterString("name~alice")
	assert.NoError(t, err)
	assert.Equal(t, "name", col)
	assert.Equal(t, domainQuery.OperatorLike, op)
	assert.Equal(t, []string{"alice"}, vals)
}

func TestParseFilterString_ILike(t *testing.T) {
	_, op, _, err := parseFilterString("name~*alice")
	assert.NoError(t, err)
	assert.Equal(t, domainQuery.OperatorILike, op)
}

func TestParseFilterString_IN(t *testing.T) {
	col, op, vals, err := parseFilterString("id$a,b,c")
	assert.NoError(t, err)
	assert.Equal(t, "id", col)
	assert.Equal(t, domainQuery.OperatorIn, op)
	assert.Equal(t, []string{"a", "b", "c"}, vals)
}

func TestParseFilterString_IN_EmptyValues_Error(t *testing.T) {
	_, _, _, err := parseFilterString("id$ ")
	assert.Error(t, err)
}

func TestParseFilterString_BETWEEN(t *testing.T) {
	col, op, vals, err := parseFilterString("age..10,20")
	assert.NoError(t, err)
	assert.Equal(t, "age", col)
	assert.Equal(t, domainQuery.OperatorBetween, op)
	assert.Equal(t, []string{"10", "20"}, vals)
}

func TestParseFilterString_BETWEEN_AndKeyword(t *testing.T) {
	_, _, vals, err := parseFilterString("age..10 AND 20")
	assert.NoError(t, err)
	assert.Equal(t, []string{"10", "20"}, vals)
}

func TestParseFilterString_BETWEEN_WrongCount_Error(t *testing.T) {
	_, _, _, err := parseFilterString("age..10")
	assert.Error(t, err)
}

func TestParseFilterString_NULL(t *testing.T) {
	col, op, vals, err := parseFilterString("status_")
	assert.NoError(t, err)
	assert.Equal(t, "status", col)
	assert.Equal(t, domainQuery.OperatorNull, op)
	assert.Nil(t, vals)
}

func TestParseFilterString_NotNULL(t *testing.T) {
	col, op, vals, err := parseFilterString("status!_")
	assert.NoError(t, err)
	assert.Equal(t, "status", col)
	assert.Equal(t, domainQuery.OperatorNotNull, op)
	assert.Nil(t, vals)
}

func TestParseFilterString_Empty_Error(t *testing.T) {
	_, _, _, err := parseFilterString("")
	assert.Error(t, err)
}

func TestParseFilterString_InvalidSyntax_Error(t *testing.T) {
	_, _, _, err := parseFilterString("nocolumnoperator")
	assert.Error(t, err)
}

func TestParseFilterString_OperatorRequiresValue(t *testing.T) {
	_, _, _, err := parseFilterString("name=")
	assert.Error(t, err)
}

func TestParseSortString_DescPrefix(t *testing.T) {
	col, ord, err := parseSortString("-name")
	assert.NoError(t, err)
	assert.Equal(t, "name", col)
	assert.Equal(t, domainQuery.SortDesc, ord)
}

func TestParseSortString_AscPlusPrefix(t *testing.T) {
	col, ord, err := parseSortString("+name")
	assert.NoError(t, err)
	assert.Equal(t, "name", col)
	assert.Equal(t, domainQuery.SortAsc, ord)
}

func TestParseSortString_DefaultAsc(t *testing.T) {
	col, ord, err := parseSortString("name")
	assert.NoError(t, err)
	assert.Equal(t, "name", col)
	assert.Equal(t, domainQuery.SortAsc, ord)
}

func TestParseSortString_Empty_Error(t *testing.T) {
	_, _, err := parseSortString("")
	assert.Error(t, err)
}

func TestParseSortString_OnlyPrefix_Error(t *testing.T) {
	_, _, err := parseSortString("-")
	assert.Error(t, err)
	_, _, err = parseSortString("+")
	assert.Error(t, err)
}
