package fixtures

import (
	"encoding/hex"
	"testing"

	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/crypto"

	ethCrypto "github.com/ethereum/go-ethereum/crypto"
)

// TestWalletEncryptionKey returns a deterministic 32-byte AES key for tests.
func TestWalletEncryptionKey() []byte {
	key := make([]byte, 32)
	for i := range key {
		key[i] = 0xAA
	}
	return key
}

// NewWallet generates an ECDSA keypair, derives the address, and encrypts the
// private key with TestWalletEncryptionKey.
func NewWallet(t *testing.T) domain.Wallet {
	t.Helper()
	key, err := ethCrypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	privateKeyHex := hex.EncodeToString(ethCrypto.FromECDSA(key))
	address := ethCrypto.PubkeyToAddress(key.PublicKey).Hex()
	encrypted, err := crypto.Encrypt([]byte(privateKeyHex), TestWalletEncryptionKey())
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	return domain.Wallet{
		Address:             address,
		EncryptedPrivateKey: encrypted,
	}
}
