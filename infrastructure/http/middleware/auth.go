package middleware

import (
	"context"
	"strings"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	httpContext "CredChain_Golang/infrastructure/http/context"
	"CredChain_Golang/infrastructure/http/responder"
	"CredChain_Golang/infrastructure/security"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware validates JWT token and stores claims in both Gin and Go contexts
func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Abort()
			responder.SendError(c, domain.NewError(domain.CodeAuthLoginUnauthorized))
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.Abort()
			responder.SendError(c, domain.NewError(domain.CodeAuthLoginUnauthorized))
			return
		}

		claims, err := parseJWTClaims(tokenString, cfg.JWTSecret)
		if err != nil {
			c.Abort()
			responder.SendError(c, domain.NewError(domain.CodeAuthLoginInvalidToken))
			return
		}

		// Create user claims for context (map from JWT claims to UserClaims)
		userClaims := &httpContext.UserClaims{
			Id:    claims.UserID,
			Role:  domain.Role(claims.Role),
			Email: claims.Email,
		}

		// Store in Gin context using type-safe key
		c.Set(httpContext.UserClaimsKey, userClaims)

		// Store in Go context (for service layer)
		ctx := context.WithValue(c.Request.Context(), httpContext.UserClaimsKey, userClaims)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// parseJWTClaims parses and validates the JWT token, extracting claims
func parseJWTClaims(tokenString string, secret string) (*security.JWTClaims, error) {
	claims, err := security.ValidateJWT(tokenString, []byte(secret))
	if err != nil {
		return nil, err
	}
	return claims, nil
}

// RequireMinRole returns a middleware that enforces a minimum role requirement
func RequireMinRole(minRole domain.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := httpContext.GetUserClaims(c.Request.Context())
		if err != nil {
			responder.SendError(c, domain.NewError(domain.CodeAuthLoginUnauthorized))
			c.Abort()
			return
		}

		if claims.Role.Rank() < minRole.Rank() {
			responder.SendError(c, domain.NewError(domain.CodeAuthLoginForbidden))
			c.Abort()
			return
		}

		c.Next()
	}
}
