package crypto

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMustGenerateRandomHex32_Length(t *testing.T) {
	got := MustGenerateRandomHex32()
	assert.Len(t, got, 64, "32 bytes hex-encoded == 64 chars")
}

func TestMustGenerateRandomHex32_ValidHex(t *testing.T) {
	got := MustGenerateRandomHex32()
	bytes, err := hex.DecodeString(got)
	assert.NoError(t, err)
	assert.Len(t, bytes, 32)
}

func TestMustGenerateRandomHex32_Uniqueness(t *testing.T) {
	a := MustGenerateRandomHex32()
	b := MustGenerateRandomHex32()
	assert.NotEqual(t, a, b)
}
