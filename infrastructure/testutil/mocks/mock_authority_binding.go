package mocks

import (
	"CredChain_Golang/infrastructure/chain/contracts"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/mock"
)

type MockAuthorityBinding struct {
	mock.Mock
}

func (m *MockAuthorityBinding) UserToRole(opts *bind.CallOpts, addr common.Address) (uint8, error) {
	args := m.Called(opts, addr)
	return uint8(args.Int(0)), args.Error(1)
}

func (m *MockAuthorityBinding) BatchUpdateUserRoleWithSignature(opts *bind.TransactOpts, params contracts.CredentialAuthorityBatchUpdateUserRoleWithSignatureParams) (*types.Transaction, error) {
	args := m.Called(opts, params)
	if v := args.Get(0); v != nil {
		return v.(*types.Transaction), args.Error(1)
	}
	return nil, args.Error(1)
}
