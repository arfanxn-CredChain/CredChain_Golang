package chain

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"CredChain_Golang/config"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/fx"
)

// Client wraps the ethclient and bound contract instances.
// It provides a facade for blockchain operations including RPC calls,
// contract interactions, and relayer transaction management.
//
// # Responsibilities
//
//   - Manages RPC connection to Ethereum node
//   - Holds bound contract instances (Registry, Authority)
//   - Manages relayer wallet for transaction signing
//   - Provides chain ID for transaction replay protection
//
// # Usage
//
// Client is created via NewClient() and injected via Uber FX.
// Services wrap Client with domain-specific interfaces (e.g., AuthorityService).
type Client struct {
	EthClient *ethclient.Client
	Registry  *Registry
	Authority *Authority
	ChainID   *big.Int
	Relayer   *bind.TransactOpts
}

type ChainParams struct {
	fx.In
	Config *config.Config
}

// NewClient establishes the connection to the RPC endpoint and initializes
// contract bindings for Registry and Authority. It also sets up the Relayer
// transaction signer using the configured private key.
//
// # Configuration Requirements
//
// The following Config fields must be set:
//   - RPCURL: Ethereum node RPC endpoint
//   - RegistryContract: Deployed CredentialRegistry address
//   - AuthorityContract: Deployed CredentialAuthority address
//   - RelayerPrivateKey: Private key for transaction relaying
//
// # Lifecycle
//
// Client is created once at application startup and lives for the entire
// application lifetime. It is safe for concurrent use.
//
// Parameters:
//   - p: ChainParams with Config from FX container
//
// Returns:
//   - *Client: Initialized client instance
//   - error: If RPC connection, contract binding, or key setup fails
func NewClient(p ChainParams) (*Client, error) {
	if p.Config.RPCURL == nil {
		return nil, fmt.Errorf("RPC_URL not configured")
	}

	ethClient, err := ethclient.Dial(*p.Config.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("failed to dial RPC: %w", err)
	}

	chainID, err := ethClient.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	if p.Config.RegistryContract == nil || p.Config.AuthorityContract == nil || p.Config.RelayerPrivateKey == nil {
		return nil, fmt.Errorf("missing blockchain contract or relayer config")
	}

	registryAddr := common.HexToAddress(*p.Config.RegistryContract)
	authorityAddr := common.HexToAddress(*p.Config.AuthorityContract)

	registry, err := NewRegistry(registryAddr, ethClient)
	if err != nil {
		return nil, fmt.Errorf("failed to bind registry: %w", err)
	}

	authority, err := NewAuthority(authorityAddr, ethClient)
	if err != nil {
		return nil, fmt.Errorf("failed to bind authority: %w", err)
	}

	// Setup Relayer Wallet
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(*p.Config.RelayerPrivateKey, "0x"))
	if err != nil {
		return nil, fmt.Errorf("invalid relayer private key: %w", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create relayer transactor: %w", err)
	}

	return &Client{
		EthClient: ethClient,
		Registry:  registry,
		Authority: authority,
		ChainID:   chainID,
		Relayer:   auth,
	}, nil
}

// FetchNonce gets the deterministic nonce from the Registry for the given user address.
//
// Deprecated: Use AuthorityService.FindNonce() instead. This method will be removed
// in a future refactoring.
func (c *Client) FetchNonce(ctx context.Context, userWallet common.Address) (*big.Int, error) {
	nonce, err := c.Registry.UserToNonce(&bind.CallOpts{Context: ctx}, userWallet)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch nonce from registry: %w", err)
	}
	return nonce, nil
}
