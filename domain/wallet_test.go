package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWalletFromUser_PopulatesFields(t *testing.T) {
	u := User{
		WalletAddress:             "0x1234567890abcdef1234567890abcdef12345678",
		EncryptedWalletPrivateKey: "encrypted-hex-blob",
	}
	w := WalletFromUser(u)
	assert.Equal(t, "0x1234567890abcdef1234567890abcdef12345678", w.Address)
	assert.Equal(t, "encrypted-hex-blob", w.EncryptedPrivateKey)
}

func TestWalletFromUser_EmptyUserYieldsZeroWallet(t *testing.T) {
	w := WalletFromUser(User{})
	assert.Equal(t, "", w.Address)
	assert.Equal(t, "", w.EncryptedPrivateKey)
}
