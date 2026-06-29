package meta

import (
	"context"
	"errors"
	"testing"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockChainClient struct {
	mock.Mock
}

func (m *mockChainClient) BlockNumber(ctx context.Context) (uint64, error) {
	args := m.Called(ctx)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *mockChainClient) ChainID(ctx context.Context) (uint64, error) {
	args := m.Called(ctx)
	return args.Get(0).(uint64), args.Error(1)
}

func TestMetaService_Get_Success(t *testing.T) {
	chainMock := new(mockChainClient)
	cfg := &config.Config{
		IssuingOrganizationName: ptrStr("University of Indonesia"),
		AuthorityContract:       ptrStr("0xAAA"),
		RegistryContract:        ptrStr("0xBBB"),
	}
	svc := newMetaService(cfg, chainMock)

	chainMock.On("BlockNumber", mock.Anything).Return(uint64(42000000), nil)
	chainMock.On("ChainID", mock.Anything).Return(uint64(137), nil)

	result, err := svc.Get(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "University of Indonesia", result.IssuingOrganizationName)
	assert.Equal(t, "0xAAA", result.AuthorityContract)
	assert.Equal(t, "0xBBB", result.RegistryContract)
	assert.Equal(t, uint64(137), result.ChainID)
	assert.Equal(t, uint64(42000000), result.LastBlock)
	chainMock.AssertExpectations(t)
}

func TestMetaService_Get_BlockNumberError(t *testing.T) {
	chainMock := new(mockChainClient)
	cfg := &config.Config{
		IssuingOrganizationName: ptrStr("University of Indonesia"),
		AuthorityContract:       ptrStr("0xAAA"),
		RegistryContract:        ptrStr("0xBBB"),
	}
	svc := newMetaService(cfg, chainMock)

	chainMock.On("BlockNumber", mock.Anything).Return(uint64(0), errors.New("rpc down"))

	_, err := svc.Get(context.Background())
	require.Error(t, err)

	var dErr *domain.Error
	assert.True(t, errors.As(err, &dErr))
	assert.Equal(t, domain.CodeMetaInternal, dErr.Code)
}

func TestMetaService_Get_ChainIDError(t *testing.T) {
	chainMock := new(mockChainClient)
	cfg := &config.Config{
		IssuingOrganizationName: ptrStr("University of Indonesia"),
		AuthorityContract:       ptrStr("0xAAA"),
		RegistryContract:        ptrStr("0xBBB"),
	}
	svc := newMetaService(cfg, chainMock)

	chainMock.On("BlockNumber", mock.Anything).Return(uint64(42000000), nil)
	chainMock.On("ChainID", mock.Anything).Return(uint64(0), errors.New("rpc down"))

	_, err := svc.Get(context.Background())
	require.Error(t, err)

	var dErr *domain.Error
	assert.True(t, errors.As(err, &dErr))
	assert.Equal(t, domain.CodeMetaInternal, dErr.Code)
}

func ptrStr(s string) *string { return &s }
