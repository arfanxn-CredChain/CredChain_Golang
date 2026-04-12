package middleware

import (
	"fmt"
	"strings"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/http/responder"
	"CredChain_Golang/infrastructure/security"

	"github.com/gin-gonic/gin"
)

const (
	UserContextKey = "user_claims"
)

// RequireAuth extracts the Bearer token, validates it, and injects JWTClaims into gin context.
func RequireAuth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			responder.SendError(c, domain.CodeAuthLoginUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		claims, err := security.ValidateJWT(tokenString, []byte(cfg.JWTSecret))
		if err != nil {
			c.Error(fmt.Errorf("RequireAuth: JWT validation failed: %w", err)) //nolint:errcheck
			responder.SendError(c, domain.CodeAuthLoginInvalidToken)
			return
		}

		c.Set(UserContextKey, claims)
		c.Next()
	}
}

// RequireMinRole ensures the authenticated user's role rank is at least the required role's rank.
func RequireMinRole(minRole domain.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsInterface, exists := c.Get(UserContextKey)
		if !exists {
			responder.SendError(c, domain.CodeAuthLoginUnauthorized)
			return
		}

		claims, ok := claimsInterface.(*security.JWTClaims)
		if !ok || claims == nil {
			c.Error(fmt.Errorf("RequireMinRole: type assertion failed, got %T", claimsInterface)) //nolint:errcheck
			responder.SendError(c, domain.CodeSystemInternal)
			return
		}

		userRole := domain.Role(claims.Role)
		if userRole.Rank() >= minRole.Rank() {
			c.Next()
			return
		}

		responder.SendError(c, domain.CodeAuthLoginForbidden)
	}
}

// GetUserClaims retrieves the typed JWTClaims from the gin context.
func GetUserClaims(c *gin.Context) *security.JWTClaims {
	claimsInterface, exists := c.Get(UserContextKey)
	if !exists {
		return nil
	}
	claims, ok := claimsInterface.(*security.JWTClaims)
	if !ok {
		c.Error(fmt.Errorf("GetUserClaims: type assertion failed, got %T", claimsInterface)) //nolint:errcheck
		return nil
	}
	return claims
}
