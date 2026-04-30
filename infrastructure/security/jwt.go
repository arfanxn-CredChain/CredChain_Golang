package security

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims represents the custom claims tightly coupling Postgres state into the token
type JWTClaims struct {
	UserId string `json:"user_id"`
	jwt.RegisteredClaims
}

// GenerateJWT creates a new signed token
func GenerateJWT(secretKey []byte, userId string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour) // 1 day validity

	claims := &JWTClaims{
		UserId: userId,
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
