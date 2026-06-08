package mocks

import (
	"context"
	"math/big"

	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/chain"

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

func (m *MockRegistryService) FindCredentialByHash(ctx context.Context, hash string) (string, bool, error) {
	args := m.Called(ctx, hash)
	return args.String(0), args.Bool(1), args.Error(2)
}

var _ chain.RegistryService = (*MockRegistryService)(nil)
