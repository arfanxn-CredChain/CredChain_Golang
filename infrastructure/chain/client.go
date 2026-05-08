package chain

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"CredChain_Golang/config"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/fx"
)

// Client wraps the ethclient and the bound contracts
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

// NewClient establishes the connection to the RPC and sets up the TransactOpts for the Relayer
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

// FetchNonce gets the deterministic nonce from the Registry for the given user address
func (c *Client) FetchNonce(ctx context.Context, userWallet common.Address) (*big.Int, error) {
	nonce, err := c.Registry.UserToNonce(&bind.CallOpts{Context: ctx}, userWallet)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch nonce from registry: %w", err)
	}
	return nonce, nil
}

// DispatchRelayerTx broadcasts a generic custom transaction using Relayer auth
func (c *Client) DispatchRelayerTx(ctx context.Context, method func(*bind.TransactOpts) (*types.Transaction, error)) (*types.Transaction, error) {
	relayerAddr := c.Relayer.From
	nonce, err := c.EthClient.PendingNonceAt(ctx, relayerAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch relayer nonce: %w", err)
	}

	c.Relayer.Nonce = big.NewInt(int64(nonce))

	tx, err := method(c.Relayer)
	if err != nil {
		return nil, fmt.Errorf("relayer transaction failed: %w", err)
	}

	return tx, nil
}
