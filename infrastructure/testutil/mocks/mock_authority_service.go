package mocks

import (
	"context"
	"math/big"

	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/chain"

	"github.com/stretchr/testify/mock"
)

type MockAuthorityService struct {
	mock.Mock
}

func (m *MockAuthorityService) FindRole(ctx context.Context, addr string) (domain.Role, error) {
	args := m.Called(ctx, addr)
	return args.Get(0).(domain.Role), args.Error(1)
}

func (m *MockAuthorityService) HasRoleOrAbove(ctx context.Context, addr string, minRole domain.Role) bool {
	args := m.Called(ctx, addr, minRole)
	return args.Bool(0)
}

func (m *MockAuthorityService) UpdateUserRole(ctx context.Context, signer domain.Wallet, users ...domain.User) error {
	args := m.Called(ctx, signer, users)
	return args.Error(0)
}

func (m *MockAuthorityService) FindNonce(ctx context.Context, addr string) (*big.Int, error) {
	args := m.Called(ctx, addr)
	if v := args.Get(0); v != nil {
		return v.(*big.Int), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAuthorityService) TransferSuperAdmin(ctx context.Context, signer domain.Wallet, newSuperAdmin domain.User) error {
	args := m.Called(ctx, signer, newSuperAdmin)
	return args.Error(0)
}

var _ chain.AuthorityService = (*MockAuthorityService)(nil)
