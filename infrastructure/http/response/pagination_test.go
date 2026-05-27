package response

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newPaginationCtx(query string) *gin.Context {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/users?"+query, nil)
	return c
}

func TestNewPaginationFromContext_NilItemsBecomesEmpty(t *testing.T) {
	c := newPaginationCtx("page=1&limit=10")
	p := NewPaginationFromContext[int](c, nil, 0)
	assert.NotNil(t, p.Items)
	assert.Len(t, p.Items, 0)
}

func TestNewPaginationFromContext_FirstPage_NoPrev(t *testing.T) {
	c := newPaginationCtx("page=1&limit=10")
	p := NewPaginationFromContext(c, []int{1, 2, 3}, 25)
	assert.Equal(t, 1, p.Page)
	assert.Nil(t, p.PrevPageURL)
	assert.NotNil(t, p.NextPageURL)
}

func TestNewPaginationFromContext_LastPage_NoNext(t *testing.T) {
	c := newPaginationCtx("page=3&limit=10")
	p := NewPaginationFromContext(c, []int{21, 22, 23}, 23)
	assert.Equal(t, 3, p.Page)
	assert.Nil(t, p.NextPageURL)
	assert.NotNil(t, p.PrevPageURL)
}

func TestNewPaginationFromContext_MiddlePage_BothURLs(t *testing.T) {
	c := newPaginationCtx("page=2&limit=10")
	p := NewPaginationFromContext(c, []int{11, 12}, 25)
	assert.NotNil(t, p.PrevPageURL)
	assert.NotNil(t, p.NextPageURL)
}

func TestNewPaginationFromContext_TotalZero(t *testing.T) {
	c := newPaginationCtx("page=1&limit=10")
	p := NewPaginationFromContext[int](c, []int{}, 0)
	assert.Equal(t, 0, p.From)
	assert.Equal(t, 0, p.To)
	assert.Equal(t, 1, p.LastPage)
}

func TestNewPaginationFromContext_ToCappedAtTotal(t *testing.T) {
	c := newPaginationCtx("page=3&limit=10")
	p := NewPaginationFromContext(c, []int{21, 22, 23}, 23)
	assert.Equal(t, 21, p.From)
	assert.Equal(t, 23, p.To)
}

func TestNewPaginationFromContext_DefaultsAppliedWhenInvalidQuery(t *testing.T) {
	c := newPaginationCtx("page=abc&limit=xyz")
	p := NewPaginationFromContext(c, []int{1}, 1)
	assert.Equal(t, 1, p.Page)
	assert.Equal(t, 10, p.Limit)
}

func TestNewPaginationFromContext_URLContainsPageAndLimit(t *testing.T) {
	c := newPaginationCtx("page=1&limit=10")
	p := NewPaginationFromContext(c, []int{1}, 1)
	assert.NotNil(t, p.FirstPageURL)
	assert.Contains(t, *p.FirstPageURL, "page=1")
	assert.Contains(t, *p.FirstPageURL, "limit=10")
}
