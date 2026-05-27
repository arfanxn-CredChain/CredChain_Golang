package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultQuery(t *testing.T) {
	q := DefaultQuery()
	assert.Equal(t, 1, q.Page)
	assert.Equal(t, 10, q.Limit)
}

func TestQuery_Offset_FirstPage(t *testing.T) {
	q := Query{Page: 1, Limit: 10}
	assert.Equal(t, 0, q.Offset())
}

func TestQuery_Offset_SecondPage(t *testing.T) {
	q := Query{Page: 2, Limit: 10}
	assert.Equal(t, 10, q.Offset())
}

func TestQuery_Offset_ThirdPageLargeLimit(t *testing.T) {
	q := Query{Page: 3, Limit: 25}
	assert.Equal(t, 50, q.Offset())
}

func TestQuery_Offset_NormalizesPageBelowOne(t *testing.T) {
	q := Query{Page: 0, Limit: 10}
	assert.Equal(t, 0, q.Offset())
	assert.Equal(t, 1, q.Page, "Offset should normalize Page to 1")
}

func TestQuery_Offset_NormalizesLimitBelowOne(t *testing.T) {
	q := Query{Page: 1, Limit: 0}
	assert.Equal(t, 0, q.Offset())
	assert.Equal(t, 10, q.Limit, "Offset should normalize Limit to 10")
}

func TestQuery_Offset_NegativePage(t *testing.T) {
	q := Query{Page: -5, Limit: 10}
	assert.Equal(t, 0, q.Offset())
	assert.Equal(t, 1, q.Page)
}
