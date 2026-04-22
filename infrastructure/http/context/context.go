package context

import (
	"CredChain_Golang/domain"
	"context"
	"fmt"
)

// contextKey is a private type for context keys (type-safe)
type contextKey string

const (
	// UserClaimsKey is the key for user claims in context
	UserClaimsKey contextKey = "user_claims"
)

// UserClaims represents user data extracted from JWT and stored in context
type UserClaims struct {
	Id    string
	Role  domain.Role
	Email string
}

// GetUserClaims retrieves user claims from context
func GetUserClaims(ctx context.Context) (*UserClaims, error) {
	claims, ok := ctx.Value(UserClaimsKey).(*UserClaims)
	if !ok || claims == nil {
		return nil, fmt.Errorf("user claims not found")
	}
	return claims, nil
}

// MustGetUserClaims retrieves user claims from context or panics if not found.
// This should only be used in handlers protected by AuthMiddleware.
func MustGetUserClaims(ctx context.Context) *UserClaims {
	claims, err := GetUserClaims(ctx)
	if err != nil {
		panic("user claims missing in authenticated context - AuthMiddleware not configured?")
	}
	return claims
}

// GetUserId retrieves user ID from context
func GetUserId(ctx context.Context) (string, error) {
	claims, err := GetUserClaims(ctx)
	if err != nil {
		return "", err
	}
	return claims.Id, nil
}

// GetUserRole retrieves user role from context
func GetUserRole(ctx context.Context) (domain.Role, error) {
	claims, err := GetUserClaims(ctx)
	if err != nil {
		return "", err
	}
	return claims.Role, nil
}

// GetUserEmail retrieves user email from context
func GetUserEmail(ctx context.Context) (string, error) {
	claims, err := GetUserClaims(ctx)
	if err != nil {
		return "", err
	}
	return claims.Email, nil
}
