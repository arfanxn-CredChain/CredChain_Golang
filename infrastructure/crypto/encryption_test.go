package crypto

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testKey() []byte {
	k := make([]byte, 32)
	for i := range k {
		k[i] = 0x42
	}
	return k
}

func TestEncrypt_Decrypt_RoundTrip(t *testing.T) {
	plaintext := []byte("hello world, this is secret")
	cipher, err := Encrypt(plaintext, testKey())
	assert.NoError(t, err)
	assert.NotEmpty(t, cipher)

	decoded, err := Decrypt(cipher, testKey())
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decoded)
}

func TestEncrypt_ProducesHexString(t *testing.T) {
	cipher, err := Encrypt([]byte("x"), testKey())
	assert.NoError(t, err)
	_, decodeErr := hex.DecodeString(cipher)
	assert.NoError(t, decodeErr, "cipher must be valid hex")
}

func TestEncrypt_DifferentNonces(t *testing.T) {
	a, _ := Encrypt([]byte("same"), testKey())
	b, _ := Encrypt([]byte("same"), testKey())
	assert.NotEqual(t, a, b, "GCM nonce randomness should produce different outputs")
}

func TestDecrypt_WrongKey(t *testing.T) {
	cipher, _ := Encrypt([]byte("secret"), testKey())
	wrongKey := make([]byte, 32)
	for i := range wrongKey {
		wrongKey[i] = 0x99
	}
	_, err := Decrypt(cipher, wrongKey)
	assert.Error(t, err)
}

func TestDecrypt_InvalidHex(t *testing.T) {
	_, err := Decrypt("not-valid-hex!!", testKey())
	assert.Error(t, err)
}

func TestDecrypt_TooShortCiphertext(t *testing.T) {
	_, err := Decrypt("aabb", testKey())
	assert.Error(t, err)
}

func TestEncrypt_InvalidKeyLength(t *testing.T) {
	_, err := Encrypt([]byte("x"), []byte("short"))
	assert.Error(t, err)
}
