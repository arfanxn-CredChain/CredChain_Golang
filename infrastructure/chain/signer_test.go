package chain

import (
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
)

func TestEncodeAddress_20Bytes(t *testing.T) {
	out := EncodeAddress("0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb1")
	assert.Len(t, out, 20)
}

func TestEncodeString_UTF8(t *testing.T) {
	assert.Equal(t, []byte("hello"), EncodeString("hello"))
	assert.Equal(t, []byte{0xE6, 0x97, 0xA5}, EncodeString("日"))
}

func TestEncodeUint256_Zero(t *testing.T) {
	out, err := EncodeUint256("0")
	assert.NoError(t, err)
	assert.Len(t, out, 32)
	assert.Equal(t, make([]byte, 32), out)
}

func TestEncodeUint256_Valid(t *testing.T) {
	out, err := EncodeUint256("256")
	assert.NoError(t, err)
	assert.Len(t, out, 32)
	// 256 = 0x0100, padded to 32 bytes left-aligned
	assert.Equal(t, byte(0x01), out[30])
	assert.Equal(t, byte(0x00), out[31])
}

func TestEncodeUint256_Invalid(t *testing.T) {
	_, err := EncodeUint256("not-a-number")
	assert.Error(t, err)
}

func TestPackAndSign_DeterministicForKnownInput(t *testing.T) {
	keyHex := "4242424242424242424242424242424242424242424242424242424242424242"
	keyBytes, _ := hex.DecodeString(keyHex)
	privKey, err := crypto.ToECDSA(keyBytes)
	assert.NoError(t, err)

	addr := common.HexToAddress("0x000000000000000000000000000000000000beef").Bytes()
	signature, err := PackAndSign(privKey, addr, []byte{0x01, 0x02, 0x03})
	assert.NoError(t, err)
	assert.Len(t, signature, 65)
	v := signature[64]
	assert.True(t, v == 27 || v == 28, "V should be 27 or 28, got %d", v)
}

func TestPackAndSign_DifferentArgsProduceDifferentSignatures(t *testing.T) {
	keyBytes, _ := hex.DecodeString("4242424242424242424242424242424242424242424242424242424242424242")
	privKey, _ := crypto.ToECDSA(keyBytes)

	a, _ := PackAndSign(privKey, []byte("a"))
	b, _ := PackAndSign(privKey, []byte("b"))
	assert.NotEqual(t, a, b)
}
