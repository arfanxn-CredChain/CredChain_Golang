package chain

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	cryptoInfra "CredChain_Golang/infrastructure/crypto"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// AuthorityService provides access to the CredentialAuthority blockchain contract.
// It is the infrastructure layer's interface for role-based authorization operations.
//
// # Responsibilities
//
// This service handles all interactions with the CredentialAuthority smart contract:
//   - Reading user roles from the on-chain registry (read-only calls)
//   - Verifying minimum role requirements for authorization decisions
//   - Batch updating user roles with signature-based authentication (write calls)
//
// # Architecture
//
// This is an infrastructure layer service. It wraps the auto-generated Authority
// contract binding and provides a higher-level API for feature services.
//
// # Error Handling
//
// All methods return raw errors (fmt.Errorf with %w wrapping). Feature layer services
// are responsible for translating to domain error codes.
//
// # Dependencies
//
// Uses the Client facade for RPC connections, contract bindings, and relayer credentials.
// The Client must be initialized with valid Registry and Authority contract addresses.
type AuthorityService interface {
	// FindRole retrieves the on-chain role for the given wallet address from the
	// CredentialAuthority contract.
	//
	// This is a read-only call (no gas cost, no transaction).
	//
	// Parameters:
	//   - ctx: Context for timeout/cancellation control
	//   - addr: Ethereum wallet address as a hex string with "0x" prefix (42 characters, e.g., "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb")
	//
	// Returns:
	//   - domain.Role: One of RoleHolder, RoleIssuer, RoleAdmin, RoleSuperAdmin
	//   - error: Non-nil if the blockchain call fails
	//
	// Example:
	//   role, err := service.FindRole(ctx, "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb")
	//   if err != nil {
	//       return domain.NewError(domain.CodeChainReadFailed, domain.WithError(err))
	//   }
	FindRole(ctx context.Context, addr string) (domain.Role, error)

	// HasRoleOrAbove checks if the given wallet address has at least the specified
	// minimum role in the on-chain hierarchy.
	//
	// The role hierarchy is: None(0) < Holder(1) < Issuer(2) < Admin(3) < SuperAdmin(4)
	//
	// Parameters:
	//   - addr: Ethereum wallet address as a hex string (e.g., "0x123...")
	//   - minRole: The minimum required role (e.g., domain.RoleAdmin)
	//
	// Returns:
	//   - true: If the user's on-chain role rank >= minRole.Rank()
	//   - false: If the user's role is lower, or if any error occurs during lookup
	//
	// Note: This method swallows errors and returns false on failure. For explicit
	// error handling, use FindRole() directly.
	HasRoleOrAbove(ctx context.Context, addr string, minRole domain.Role) bool

	// UpdateUserRole performs a batch role update for multiple users in a single
	// blockchain transaction. It handles the complete signature-based authentication flow:
	//
	//   1. Fetches the current nonce from CredentialRegistry for the signer
	//   2. Packs transaction data: signer || nonce || userRoles[]
	//   3. Signs with the signer's encrypted private key
	//   4. Executes BatchUpdateUserRoleWithSignature on CredentialAuthority
	//
	// Parameters:
	//   - ctx: Context for timeout/cancellation control
	//   - signer: Wallet containing Address (hex string with "0x" prefix, 42 characters) and EncryptedPrivateKey for signing
	//   - users: Variadic list of domain.User entities to update
	//
	// Returns:
	//   - error: If any step fails (nonce fetch, decryption, signing, or transaction)
	//
	// Requirements:
	//   - Signer must have sufficient role to update target users' roles
	//   - Signer's wallet must be decrypted and available
	//   - All users must have valid WalletAddress and Role fields
	//
	// Example:
	//   err := service.UpdateUserRole(ctx, wallet, user1, user2, user3)
	//   if err != nil {
	//       return domain.NewError(domain.CodeUserStoreBlockchainSyncFailed, domain.WithError(err))
	//   }
	UpdateUserRole(ctx context.Context, signer domain.Wallet, users ...domain.User) error

	// FindNonce retrieves the deterministic nonce for the given wallet address from
	// the CredentialRegistry contract.
	//
	// The nonce is used for replay protection in signature-based transactions.
	// It increments with each successful transaction from the given address.
	//
	// Parameters:
	//   - ctx: Context for timeout/cancellation control
	//   - addr: Ethereum wallet address as a hex string with "0x" prefix (42 characters)
	//
	// Returns:
	//   - *big.Int: The current nonce value
	//   - error: If the blockchain call fails
	//
	// Note: This method fetches from Registry.UserToNonce(), not Authority.
	FindNonce(ctx context.Context, addr string) (*big.Int, error)
}

// authorityService implements AuthorityService using the Client facade.
//
// This is the internal implementation. Use NewAuthorityService() to create instances.
// The Client facade provides RPC connections, contract bindings, and relayer credentials.
type authorityService struct {
	client *Client
	cfg    *config.Config
}

// NewAuthorityService creates a new AuthorityService instance using the provided
// Client facade and config.
//
// The Client must be fully initialized with:
//   - Valid RPC endpoint connection
//   - Bound Registry and Authority contract instances
//   - Relayer transaction signer (bind.TransactOpts)
//
// Parameters:
//   - client: The chain.Client facade instance
//   - cfg: Configuration containing WalletEncryptionKey
//
// Returns:
//   - AuthorityService: A new service instance ready for use
func NewAuthorityService(client *Client, cfg *config.Config) AuthorityService {
	return &authorityService{
		client: client,
		cfg:    cfg,
	}
}

// FindRole retrieves the on-chain role for the given address.
func (s *authorityService) FindRole(ctx context.Context, addr common.Address) (domain.Role, error) {
	roleUint8, err := s.client.Authority.UserToRole(&bind.CallOpts{Context: ctx}, addr)
	if err != nil {
		return domain.Role(""), fmt.Errorf("failed to fetch on-chain role: %w", err)
	}
	return domain.RoleFromUint8(roleUint8), nil
}

// HasRoleOrAbove checks if the address has at least the minimum role.
func (s *authorityService) HasRoleOrAbove(ctx context.Context, addr string, minRole domain.Role) bool {
	role, err := s.FindRole(ctx, common.HexToAddress(addr))
	if err != nil {
		return false
	}
	return role.Rank() >= minRole.Rank()
}

// FindNonce fetches the deterministic nonce from the Registry contract.
func (s *authorityService) FindNonce(ctx context.Context, addr common.Address) (*big.Int, error) {
	nonce, err := s.client.Registry.UserToNonce(&bind.CallOpts{Context: ctx}, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch nonce from registry: %w", err)
	}
	return nonce, nil
}

// UpdateUserRole performs batch role update with signature authentication.
func (s *authorityService) UpdateUserRole(ctx context.Context, signer domain.Wallet, users ...domain.User) error {
	if len(users) == 0 {
		return nil
	}

	// Convert domain.User to contract-compatible struct
	userRoles := make([]CredentialAuthorityUserRoleUpdation, len(users))
	for i, user := range users {
		addr := common.HexToAddress(user.WalletAddress)
		role := user.Role.ToUint8()
		userRoles[i] = CredentialAuthorityUserRoleUpdation{
			Addr: addr,
			Role: role,
		}
	}

	signerAddr := common.HexToAddress(signer.Address)

	// Fetch nonce from Registry for replay protection
	nonce, err := s.FindNonce(ctx, signerAddr)
	if err != nil {
		return fmt.Errorf("failed to fetch nonce: %w", err)
	}

	// Decrypt signer's wallet private key
	decryptedKey, err := cryptoInfra.Decrypt(signer.EncryptedPrivateKey, []byte(*s.cfg.WalletEncryptionKey))
	if err != nil {
		return fmt.Errorf("failed to decrypt wallet: %w", err)
	}

	// Parse ECDSA private key from hex
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(string(decryptedKey), "0x"))
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	// Pack data for signing: signer || nonce || userRoles[]
	// This order MUST match the Solidity contract's expected packing
	var packed []byte
	packed = append(packed, signerAddr.Bytes()...)
	packed = append(packed, common.LeftPadBytes(nonce.Bytes(), 32)...)
	for _, userRole := range userRoles {
		packed = append(packed, userRole.Addr.Bytes()...)
		packed = append(packed, userRole.Role)
	}

	// Create EIP-191 compliant signature
	digest := crypto.Keccak256(packed)
	signature, err := crypto.Sign(accounts.TextHash(digest), privateKey)
	if err != nil {
		return fmt.Errorf("failed to sign: %w", err)
	}
	signature[64] += 27 // Adjust v value for Ethereum (27/28)

	// Execute transaction on Authority contract
	tx, err := s.client.Authority.BatchUpdateUserRoleWithSignature(
		s.client.Relayer,
		CredentialAuthorityBatchUpdateUserRoleWithSignatureParams{
			Signer:    signerAddr,
			UserRoles: userRoles,
			Nonce:     nonce,
			Signature: signature,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to execute batch update: %w", err)
	}

	_ = tx // Transaction broadcast successful
	return nil
}
