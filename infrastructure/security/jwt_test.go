package security

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

var testSecret = []byte("test-secret-key-please-do-not-use-in-prod")

func TestGenerateJWT_RoundTrip(t *testing.T) {
	tok, err := GenerateJWT("user-123", testSecret, time.Hour)
	assert.NoError(t, err)
	assert.NotEmpty(t, tok)

	claims, err := ValiparseJWT(tok, testSecret)
	assert.NoError(t, err)
	assert.Equal(t, "user-123", claims.UserId)
	assert.Equal(t, "credchain-backend", claims.Issuer)
}

func TestValiparseJWT_Expired(t *testing.T) {
	tok, err := GenerateJWT("u", testSecret, -time.Hour)
	assert.NoError(t, err)
	_, err = ValiparseJWT(tok, testSecret)
	assert.Error(t, err)
}

func TestValiparseJWT_WrongSecret(t *testing.T) {
	tok, _ := GenerateJWT("u", testSecret, time.Hour)
	_, err := ValiparseJWT(tok, []byte("different-secret"))
	assert.Error(t, err)
}

func TestValiparseJWT_Malformed(t *testing.T) {
	_, err := ValiparseJWT("not-a-valid-token", testSecret)
	assert.Error(t, err)
}

func TestValiparseJWT_RejectsNonHMAC(t *testing.T) {
	claims := &JWTClaims{
		UserId: "u",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	str, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	assert.NoError(t, err)

	_, err = ValiparseJWT(str, testSecret)
	assert.Error(t, err, "non-HMAC signing method must be rejected")
}

func TestGenerateJWT_ContainsIssuedAt(t *testing.T) {
	tok, _ := GenerateJWT("u", testSecret, time.Hour)
	claims, err := ValiparseJWT(tok, testSecret)
	assert.NoError(t, err)
	assert.NotNil(t, claims.IssuedAt)
	assert.WithinDuration(t, time.Now(), claims.IssuedAt.Time, 5*time.Second)
}
