package crypto_test

import (
	"testing"

	cryptoInfra "CredChain_Golang/infrastructure/crypto"
	"github.com/stretchr/testify/assert"
)

func TestDeriveKeyFromMnemonic_HardhatAccount0(t *testing.T) {
	mnemonic := "test test test test test test test test test test test junk"
	expectedAddress := "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
	expectedPrivKey := "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

	privKey, addr, err := cryptoInfra.DeriveKeyFromMnemonic(mnemonic, 0)
	assert.NoError(t, err)
	assert.Equal(t, expectedPrivKey, privKey)
	assert.Equal(t, expectedAddress, addr)
}

func TestDeriveKeyFromMnemonic_HardhatAccount1(t *testing.T) {
	mnemonic := "test test test test test test test test test test test junk"
	expectedAddress := "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
	expectedPrivKey := "59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"

	privKey, addr, err := cryptoInfra.DeriveKeyFromMnemonic(mnemonic, 1)
	assert.NoError(t, err)
	assert.Equal(t, expectedPrivKey, privKey)
	assert.Equal(t, expectedAddress, addr)
}

func TestDeriveKeyFromMnemonic_HardhatAccount15(t *testing.T) {
	mnemonic := "test test test test test test test test test test test junk"

	privKey, addr, err := cryptoInfra.DeriveKeyFromMnemonic(mnemonic, 15)
	assert.NoError(t, err)
	assert.NotEmpty(t, privKey)
	assert.NotEmpty(t, addr)
	assert.Len(t, privKey, 64)
	assert.True(t, addr[:2] == "0x")
	assert.Len(t, addr, 42)
}

func TestDeriveKeyFromMnemonic_InvalidMnemonic(t *testing.T) {
	_, _, err := cryptoInfra.DeriveKeyFromMnemonic("this is not a valid mnemonic phrase", 0)
	assert.Error(t, err)
}
