package crypto

import (
	"crypto/rand"
	"encoding/hex"
)

// MustGenerateRandomHex32 creates a cryptographically secure 32-byte
// random value encoded as a hex string (64 characters).
// Panics if the random source fails.
func MustGenerateRandomHex32() string {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}
