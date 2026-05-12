package domain

// Wallet represents an Ethereum wallet address and its encrypted private key.
// It serves as a transport structure between the feature layer and infrastructure,
// decoupling infrastructure from HTTP context and domain entities.
//
// # Fields
//
//   - Address: Ethereum wallet address (hex string, e.g., "0x123...")
//   - EncryptedPrivateKey: AES-encrypted private key (hex string, may include "0x" prefix)
//
// # Security
//
// This struct intentionally does NOT contain the decrypted private key.
// Decryption is performed by infrastructure services using the wallet encryption key.
// The decrypted key exists only transiently during signing operations and is never
// persisted or exposed to the domain layer.
//
// # Usage
//
// Wallet is created from a User entity using WalletFromUser(), then passed to
// infrastructure services for blockchain operations:
//
//	wallet := domain.WalletFromUser(authUser)
//	err := authorityService.UpdateUserRole(ctx, wallet, users...)
type Wallet struct {
	Address             string
	EncryptedPrivateKey string
}

// WalletFromUser creates a Wallet from a User entity.
// The EncryptedPrivateKey field is populated from the user's encrypted wallet data.
// The private key must be decrypted by infrastructure services before use.
//
// Parameters:
//   - user: User with WalletAddress and EncryptedWalletPrivateKey populated
//
// Returns:
//   - Wallet: New wallet instance with Address and EncryptedPrivateKey set
func WalletFromUser(user User) Wallet {
	return Wallet{
		Address:             user.WalletAddress,
		EncryptedPrivateKey: user.EncryptedWalletPrivateKey,
	}
}
