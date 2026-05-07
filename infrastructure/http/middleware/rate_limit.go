package middleware

import (
	"sync"
	"time"

	"CredChain_Golang/domain"
	httpContext "CredChain_Golang/infrastructure/http/context"
	"CredChain_Golang/infrastructure/http/responder"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// Named wrapper types for rate limit middlewares.
// Follows the same pattern as AdminRoleMiddleware, IssuerRoleMiddleware, etc.
type LoginRateLimitMiddleware gin.HandlerFunc
type RefreshRateLimitMiddleware gin.HandlerFunc
type LogoutRateLimitMiddleware gin.HandlerFunc
type ApiRateLimitMiddleware gin.HandlerFunc

// keyType determines how the rate limit key is extracted.
type keyType int

const (
	keyIP keyType = iota
	keyUserIDOrIP
)

// NewLoginRateLimitMiddleware creates an IP-based rate limiter for the login endpoint.
// 10 requests per minute per IP, burst of 5.
func NewLoginRateLimitMiddleware() LoginRateLimitMiddleware {
	return LoginRateLimitMiddleware(newRateLimitMiddleware(
		rate.Every(time.Minute/10),
		5,
		keyIP,
	))
}

// NewRefreshRateLimitMiddleware creates an IP-based rate limiter for the refresh endpoint.
// 5 requests per minute per IP, burst of 3.
func NewRefreshRateLimitMiddleware() RefreshRateLimitMiddleware {
	return RefreshRateLimitMiddleware(newRateLimitMiddleware(
		rate.Every(time.Minute/5),
		3,
		keyIP,
	))
}

// NewLogoutRateLimitMiddleware creates a user-based rate limiter for the logout endpoint.
// 3 requests per minute per user ID, burst of 1.
func NewLogoutRateLimitMiddleware() LogoutRateLimitMiddleware {
	return LogoutRateLimitMiddleware(newRateLimitMiddleware(
		rate.Every(time.Minute/3),
		1,
		keyUserIDOrIP,
	))
}

// NewApiRateLimitMiddleware creates a general API rate limiter applied globally.
// 60 requests per minute per user ID (if authenticated) or IP (fallback), burst of 10.
func NewApiRateLimitMiddleware() ApiRateLimitMiddleware {
	return ApiRateLimitMiddleware(newRateLimitMiddleware(
		rate.Every(time.Minute/60),
		10,
		keyUserIDOrIP,
	))
}

// newRateLimitMiddleware is the shared implementation.
// Key extraction is hardcoded based on keyType — no keyFn parameter.
func newRateLimitMiddleware(limit rate.Limit, burst int, kt keyType) gin.HandlerFunc {
	type client struct {
		limiter *rate.Limiter
	}

	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	return func(c *gin.Context) {
		var key string
		switch kt {
		case keyIP:
			key = c.ClientIP()
		case keyUserIDOrIP:
			user, _ := httpContext.GetUser(c.Request.Context())
			if user != nil {
				key = user.Id
			} else {
				key = c.ClientIP()
			}
		}

		mu.Lock()
		cl, exists := clients[key]
		if !exists {
			cl = &client{limiter: rate.NewLimiter(limit, burst)}
			clients[key] = cl
		}
		mu.Unlock()

		if !cl.limiter.Allow() {
			responder.SendError(c, domain.NewError(domain.CodeAuthRateLimitExceeded))
			c.Abort()
			return
		}

		c.Next()
	}
}
