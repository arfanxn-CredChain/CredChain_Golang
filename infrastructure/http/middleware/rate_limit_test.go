package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	httpContext "CredChain_Golang/infrastructure/http/context"
	"CredChain_Golang/infrastructure/testutil/fixtures"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

func testIPLimiter(burst int) gin.HandlerFunc {
	return newRateLimitMiddleware(rate.Every(time.Hour), burst, keyIP)
}

func testUserLimiter(burst int) gin.HandlerFunc {
	return newRateLimitMiddleware(rate.Every(time.Hour), burst, keyUserIDOrIP)
}

func TestRateLimit_BurstAllowsThenBlocks(t *testing.T) {
	mw := testIPLimiter(2)

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/", nil)
		c.Request.RemoteAddr = "1.2.3.4:1234"
		mw(c)
		assert.NotEqual(t, http.StatusTooManyRequests, w.Code, "request %d should pass", i+1)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.RemoteAddr = "1.2.3.4:1234"
	mw(c)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestRateLimit_DistinctIPsHaveSeparateBuckets(t *testing.T) {
	mw := testIPLimiter(1)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.RemoteAddr = "1.1.1.1:1"
	mw(c)
	assert.NotEqual(t, http.StatusTooManyRequests, w.Code)

	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest("GET", "/", nil)
	c2.Request.RemoteAddr = "2.2.2.2:1"
	mw(c2)
	assert.NotEqual(t, http.StatusTooManyRequests, w2.Code)
}

func TestRateLimit_UserIDOrIP_FallsBackToIP(t *testing.T) {
	mw := testUserLimiter(1)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.RemoteAddr = "9.9.9.9:1"
	mw(c)
	assert.NotEqual(t, http.StatusTooManyRequests, w.Code)

	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest("GET", "/", nil)
	c2.Request.RemoteAddr = "9.9.9.9:1"
	mw(c2)
	assert.Equal(t, http.StatusTooManyRequests, w2.Code)
}

func TestRateLimit_UserIDKeyed(t *testing.T) {
	mw := testUserLimiter(1)
	user := fixtures.NewDomainUser(fixtures.WithID("user-rl-1"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), httpContext.UserKey, &user)
	c.Request = req.WithContext(ctx)
	mw(c)
	assert.NotEqual(t, http.StatusTooManyRequests, w.Code)

	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	req2 := httptest.NewRequest("GET", "/", nil)
	ctx2 := context.WithValue(req2.Context(), httpContext.UserKey, &user)
	c2.Request = req2.WithContext(ctx2)
	mw(c2)
	assert.Equal(t, http.StatusTooManyRequests, w2.Code)
}

func TestNewLoginRateLimitMiddleware_ReturnsHandler(t *testing.T) {
	assert.NotNil(t, NewLoginRateLimitMiddleware())
}

func TestNewRefreshRateLimitMiddleware_ReturnsHandler(t *testing.T) {
	assert.NotNil(t, NewRefreshRateLimitMiddleware())
}

func TestNewLogoutRateLimitMiddleware_ReturnsHandler(t *testing.T) {
	assert.NotNil(t, NewLogoutRateLimitMiddleware())
}

func TestNewApiRateLimitMiddleware_ReturnsHandler(t *testing.T) {
	assert.NotNil(t, NewApiRateLimitMiddleware())
}
