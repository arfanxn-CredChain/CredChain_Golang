package chain

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestValidateHexToAddress_Valid(t *testing.T) {
	addr, err := validateHexToAddress("0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb1")
	assert.NoError(t, err)
	assert.Equal(t, common.HexToAddress("0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb1"), addr)
}

func TestValidateHexToAddress_ZeroAddressIsValid(t *testing.T) {
	_, err := validateHexToAddress("0x0000000000000000000000000000000000000000")
	assert.NoError(t, err)
}

func TestValidateHexToAddress_TooShort(t *testing.T) {
	_, err := validateHexToAddress("0x123")
	assert.Error(t, err)
}

func TestValidateHexToAddress_Garbage(t *testing.T) {
	_, err := validateHexToAddress("garbage")
	assert.Error(t, err)
}

func TestValidateHexToAddress_NonHexChars(t *testing.T) {
	_, err := validateHexToAddress("0xZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ")
	assert.Error(t, err)
}

func TestMustHexToAddress_InvalidReturnsZero(t *testing.T) {
	got := mustHexToAddress("not-a-valid-address")
	assert.Equal(t, common.Address{}, got)
}

func TestMustHexToAddress_Valid(t *testing.T) {
	want := common.HexToAddress("0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb1")
	got := mustHexToAddress("0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb1")
	assert.Equal(t, want, got)
}
