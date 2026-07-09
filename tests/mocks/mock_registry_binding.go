package mocks

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"
)

type MockRegistryBinding struct {
	mock.Mock
}

func (m *MockRegistryBinding) UserToNonce(opts *bind.CallOpts, addr common.Address) (*big.Int, error) {
	args := m.Called(opts, addr)
	if v := args.Get(0); v != nil {
		return v.(*big.Int), args.Error(1)
	}
	return nil, args.Error(1)
}
