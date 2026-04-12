package security

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims represents the custom claims tightly coupling Postgres state into the token
type JWTClaims struct {
	UserID        string `json:"user_id"`
	Email         string `json:"email"`
	WalletAddress string `json:"wallet_address"`
	Role          string `json:"role"` // Maps to domain.Role string value
	jwt.RegisteredClaims
}

// GenerateJWT creates a new signed token
func GenerateJWT(secretKey []byte, userID, email, walletAddress, role string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour) // 1 day validity

	claims := &JWTClaims{
		UserID:        userID,
		Email:         email,
		WalletAddress: walletAddress,
		Role:          role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "credchain-backend",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

// ValidateJWT parses and validates the token, returning the claims
func ValidateJWT(tokenString string, secretKey []byte) (*JWTClaims, error) {
	claims := &JWTClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
