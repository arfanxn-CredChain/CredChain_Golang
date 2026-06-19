package mocks

import (
	"context"
	"math/big"

	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/chain"
	"CredChain_Golang/infrastructure/chain/contracts"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"
)

type MockRegistryService struct {
	mock.Mock
}

func (m *MockRegistryService) IssueCredentials(ctx context.Context, signer domain.Wallet, credentials ...chain.CredentialIssuance) ([]*big.Int, error) {
	args := m.Called(ctx, signer, credentials)
	if v := args.Get(0); v != nil {
		return v.([]*big.Int), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockRegistryService) RevokeCredentials(ctx context.Context, signer domain.Wallet, tokenIds ...*big.Int) error {
	args := m.Called(ctx, signer, tokenIds)
	return args.Error(0)
}

func (m *MockRegistryService) FindNonce(ctx context.Context, addr string) (*big.Int, error) {
	args := m.Called(ctx, addr)
	if v := args.Get(0); v != nil {
		return v.(*big.Int), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockRegistryService) GetCredentialsByIds(ctx context.Context, ids []*big.Int) ([]contracts.CredentialRegistryCredential, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).([]contracts.CredentialRegistryCredential), args.Error(1)
}

func (m *MockRegistryService) GetCredentialHashPerHolderStatuses(ctx context.Context, holders []common.Address, hashes [][32]byte) ([]contracts.CredentialRegistryCredentialHashStatus, error) {
	args := m.Called(ctx, holders, hashes)
	return args.Get(0).([]contracts.CredentialRegistryCredentialHashStatus), args.Error(1)
}

var _ chain.RegistryService = (*MockRegistryService)(nil)
