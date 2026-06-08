package chain

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/chain/contracts"
	cryptoInfra "CredChain_Golang/infrastructure/crypto"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// RegistryService provides access to the CredentialRegistry blockchain contract.
// It is the infrastructure layer's interface for credential issuance and revocation.
//
// Responsibilities:
//   - Issuing soulbound credential NFTs with signature-based authentication
//   - Revoking credentials on-chain
//   - Reading credential state from the registry
//
// This is an infrastructure layer service. It wraps the auto-generated Registry
// contract binding and provides a higher-level API for feature services.
// All methods return raw errors; feature layer translates to domain codes.
type RegistryService interface {
	// IssueCredentials issues multiple credentials in a single transaction.
	// It handles the complete signature-based authentication flow:
	//  1. Fetches the current nonce from CredentialRegistry for the signer
	//  2. Packs transaction data: issuer || nonce || (holder || hash || uri)[]
	//  3. Signs with the signer's encrypted private key
	//  4. Executes BatchIssueCredentialsWithSignature on CredentialRegistry
	//
	// Parameters:
	//   - ctx: Context for timeout/cancellation
	//   - signer: Wallet containing Address and EncryptedPrivateKey
	//   - credentials: Variadic list of CredentialIssuance structs
	//
	// Returns token IDs (on-chain derived from keccak256(hash)) and error.
	IssueCredentials(ctx context.Context, signer domain.Wallet, credentials ...CredentialIssuance) ([]*big.Int, error)

	// RevokeCredentials revokes multiple credentials by token ID.
	// Sets revokedAt = block.timestamp on-chain; the NFT is soulbound and cannot be burned.
	//
	// Parameters:
	//   - ctx: Context for timeout/cancellation
	//   - signer: Wallet containing Address and EncryptedPrivateKey
	//   - tokenIds: Variadic list of on-chain credential token IDs
	RevokeCredentials(ctx context.Context, signer domain.Wallet, tokenIds ...*big.Int) error

	// FindNonce retrieves the nonce for the given address from the Registry contract.
	FindNonce(ctx context.Context, addr string) (*big.Int, error)

	// FindCredentialByHash reads the on-chain credential whose token id is
	// derived from the file hash. Returns (hashOnChain, found, error).
	FindCredentialByHash(ctx context.Context, hash string) (string, bool, error)
}

// CredentialIssuance is the input struct for credential issuance.
type CredentialIssuance struct {
	HolderAddress string
	Hash          string
	URI           string
}

// registryWaitMinedFunc matches bind.WaitMined for test injection in registry_service.
type registryWaitMinedFunc func(ctx context.Context, b bind.DeployBackend, tx *types.Transaction) (*types.Receipt, error)

type registryService struct {
	client    *Client
	cfg       *config.Config
	waitMined registryWaitMinedFunc
}

func NewRegistryService(client *Client, cfg *config.Config) RegistryService {
	return &registryService{
		client:    client,
		cfg:       cfg,
		waitMined: bind.WaitMined,
	}
}

func (s *registryService) FindNonce(ctx context.Context, addr string) (*big.Int, error) {
	nonce, err := s.client.Registry.UserToNonce(&bind.CallOpts{Context: ctx}, mustHexToAddress(addr))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch nonce from registry: %w", err)
	}
	return nonce, nil
}

func (s *registryService) FindCredentialByHash(ctx context.Context, hash string) (string, bool, error) {
	id := tokenIdFromHash(hash)
	cred, err := s.client.Registry.FindCredential(&bind.CallOpts{Context: ctx}, id)
	if err != nil {
		return "", false, fmt.Errorf("find credential on-chain: %w", err)
	}
	if cred.Hash == "" {
		return "", false, nil
	}
	return cred.Hash, true, nil
}

func (s *registryService) IssueCredentials(ctx context.Context, signer domain.Wallet, credentials ...CredentialIssuance) ([]*big.Int, error) {
	if len(credentials) == 0 {
		return nil, nil
	}

	issuerAddr := mustHexToAddress(signer.Address)

	issuances := make([]contracts.CredentialRegistryCredentialIssuance, len(credentials))
	for i, c := range credentials {
		issuances[i] = contracts.CredentialRegistryCredentialIssuance{
			Holder: mustHexToAddress(c.HolderAddress),
			Hash:   c.Hash,
			Uri:    c.URI,
		}
	}

	nonce, err := s.FindNonce(ctx, signer.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch nonce: %w", err)
	}

	decryptedKey, err := cryptoInfra.Decrypt(signer.EncryptedPrivateKey, []byte(*s.cfg.WalletEncryptionKey))
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt wallet: %w", err)
	}

	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(string(decryptedKey), "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	var packed []byte
	packed = append(packed, issuerAddr.Bytes()...)
	packed = append(packed, common.LeftPadBytes(nonce.Bytes(), 32)...)
	for _, iss := range issuances {
		packed = append(packed, iss.Holder.Bytes()...)
		packed = append(packed, []byte(iss.Hash)...)
		packed = append(packed, []byte(iss.Uri)...)
	}

	digest := crypto.Keccak256(packed)
	signature, err := crypto.Sign(accounts.TextHash(digest), privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign: %w", err)
	}
	signature[64] += 27

	tx, err := s.client.Registry.BatchIssueCredentialsWithSignature(
		s.client.Relayer,
		contracts.CredentialRegistryBatchIssueCredentialsWithSignatureParams{
			Issuer:      issuerAddr,
			Credentials: issuances,
			Nonce:       nonce,
			Signature:   signature,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to execute batch issue: %w", err)
	}

	receipt, err := s.waitMined(ctx, s.client.EthClient, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for transaction: %w", err)
	}
	if receipt.Status == 0 {
		return nil, fmt.Errorf("transaction reverted on-chain: tx hash %s", tx.Hash().Hex())
	}

	tokenIds := make([]*big.Int, len(issuances))
	for i, iss := range issuances {
		tokenIds[i] = tokenIdFromHash(iss.Hash)
	}

	return tokenIds, nil
}

func (s *registryService) RevokeCredentials(ctx context.Context, signer domain.Wallet, tokenIds ...*big.Int) error {
	if len(tokenIds) == 0 {
		return nil
	}

	revokerAddr := mustHexToAddress(signer.Address)

	nonce, err := s.FindNonce(ctx, signer.Address)
	if err != nil {
		return fmt.Errorf("failed to fetch nonce: %w", err)
	}

	decryptedKey, err := cryptoInfra.Decrypt(signer.EncryptedPrivateKey, []byte(*s.cfg.WalletEncryptionKey))
	if err != nil {
		return fmt.Errorf("failed to decrypt wallet: %w", err)
	}

	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(string(decryptedKey), "0x"))
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	var packed []byte
	packed = append(packed, revokerAddr.Bytes()...)
	packed = append(packed, common.LeftPadBytes(nonce.Bytes(), 32)...)
	for _, id := range tokenIds {
		packed = append(packed, common.LeftPadBytes(id.Bytes(), 32)...)
	}

	digest := crypto.Keccak256(packed)
	signature, err := crypto.Sign(accounts.TextHash(digest), privateKey)
	if err != nil {
		return fmt.Errorf("failed to sign: %w", err)
	}
	signature[64] += 27

	tx, err := s.client.Registry.BatchRevokeCredentialsWithSignature(
		s.client.Relayer,
		contracts.CredentialRegistryBatchRevokeCredentialsWithSignatureParams{
			Revoker:       revokerAddr,
			CredentialIds: tokenIds,
			Nonce:         nonce,
			Signature:     signature,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to execute batch revoke: %w", err)
	}

	receipt, err := s.waitMined(ctx, s.client.EthClient, tx)
	if err != nil {
		return fmt.Errorf("failed to wait for transaction: %w", err)
	}
	if receipt.Status == 0 {
		return fmt.Errorf("transaction reverted on-chain: tx hash %s", tx.Hash().Hex())
	}

	return nil
}

// tokenIdFromHash derives the on-chain token ID from the file hash.
// Matches Solidity: id = uint256(keccak256(abi.encodePacked(hash)))
func tokenIdFromHash(hash string) *big.Int {
	return new(big.Int).SetBytes(crypto.Keccak256([]byte(hash)))
}
