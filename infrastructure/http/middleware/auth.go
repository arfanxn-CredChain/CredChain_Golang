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
	"go.uber.org/fx"
)

type AuthMiddlewareParams struct {
	fx.In
	Config   *config.Config
	UserRepo domain.UserRepository
}

type RoleMiddlewareParams struct {
	fx.In
	// No dependencies - reads user from context
}

// Named wrapper types for role-based middlewares
type AdminRoleMiddleware gin.HandlerFunc
type IssuerRoleMiddleware gin.HandlerFunc
type SuperAdminRoleMiddleware gin.HandlerFunc

// NewAuthMiddleware validates JWT token and stores full user in context
func NewAuthMiddleware(p AuthMiddlewareParams) gin.HandlerFunc {
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

		claims, err := parseJWTClaims(tokenString, p.Config.JWTSecret)
		if err != nil {
			c.Abort()
			responder.SendError(c, domain.NewError(domain.CodeAuthLoginInvalidToken))
			return
		}

		// Fetch full user from DB
		user, err := p.UserRepo.Find(c.Request.Context(), claims.UserId)
		if err != nil {
			c.Abort()
			responder.SendError(c, domain.NewError(domain.CodeUserFetchNotFound))
			return
		}

		// Store in both Gin context and Go context with UserKey
		c.Set(httpContext.UserKey, user)
		ctx := context.WithValue(c.Request.Context(), httpContext.UserKey, user)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// NewAdminRoleMiddleware enforces Admin or higher role requirement
func NewAdminRoleMiddleware(p RoleMiddlewareParams) AdminRoleMiddleware {
	return AdminRoleMiddleware(requireMinRoleMiddleware(domain.RoleAdmin))
}

// NewIssuerRoleMiddleware enforces Issuer or higher role requirement
func NewIssuerRoleMiddleware(p RoleMiddlewareParams) IssuerRoleMiddleware {
	return IssuerRoleMiddleware(requireMinRoleMiddleware(domain.RoleIssuer))
}

// NewSuperAdminRoleMiddleware enforces SuperAdmin role requirement
func NewSuperAdminRoleMiddleware(p RoleMiddlewareParams) SuperAdminRoleMiddleware {
	return SuperAdminRoleMiddleware(requireMinRoleMiddleware(domain.RoleSuperAdmin))
}

// requireMinRoleMiddleware returns a middleware that enforces a minimum role requirement
func requireMinRoleMiddleware(minRole domain.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := httpContext.GetUser(c.Request.Context())
		if err != nil {
			responder.SendError(c, domain.NewError(domain.CodeAuthLoginUnauthorized))
			c.Abort()
			return
		}

		if user.Role.Rank() < minRole.Rank() {
			responder.SendError(c, domain.NewError(domain.CodeAuthLoginForbidden))
			c.Abort()
			return
		}

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
