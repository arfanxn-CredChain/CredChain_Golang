package crypto

import (
	"crypto/rand"
	"encoding/hex"
)

// MustGenerateRandomToken creates a cryptographically secure random token.
// Panics if the random source fails.
func MustGenerateRandomToken() string {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}
