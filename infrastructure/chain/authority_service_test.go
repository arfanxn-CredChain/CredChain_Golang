package chain

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/chain/contracts"
	cryptoInfra "CredChain_Golang/infrastructure/crypto"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// localAuthorityBinding is an inline mock to avoid import cycle with testutil/mocks.
type localAuthorityBinding struct{ mock.Mock }

func (m *localAuthorityBinding) UserToRole(opts *bind.CallOpts, addr common.Address) (uint8, error) {
	args := m.Called(opts, addr)
	return uint8(args.Int(0)), args.Error(1)
}

func (m *localAuthorityBinding) BatchUpdateUserRoleWithSignature(opts *bind.TransactOpts, params contracts.CredentialAuthorityBatchUpdateUserRoleWithSignatureParams) (*types.Transaction, error) {
	args := m.Called(opts, params)
	if v := args.Get(0); v != nil {
		return v.(*types.Transaction), args.Error(1)
	}
	return nil, args.Error(1)
}

// localRegistryBinding is an inline mock to avoid import cycle with testutil/mocks.
type localRegistryBinding struct{ mock.Mock }

func (m *localRegistryBinding) UserToNonce(opts *bind.CallOpts, addr common.Address) (*big.Int, error) {
	args := m.Called(opts, addr)
	if v := args.Get(0); v != nil {
		return v.(*big.Int), args.Error(1)
	}
	return nil, args.Error(1)
}

// testEncKey returns a 32-byte encryption key as a string.
func testEncKey() string {
	raw := make([]byte, 32)
	for i := range raw {
		raw[i] = 0xAA
	}
	return string(raw)
}

func mkConfig() *config.Config {
	s := testEncKey()
	return &config.Config{WalletEncryptionKey: &s}
}

func mkClient(auth AuthorityBinding, reg RegistryBinding) *Client {
	return &Client{
		Authority: auth,
		Registry:  reg,
		Relayer:   &bind.TransactOpts{},
	}
}

func TestAuthorityService_FindRole_Success(t *testing.T) {
	authMock := &localAuthorityBinding{}
	regMock := &localRegistryBinding{}

	authMock.On("UserToRole", mock.Anything, mock.Anything).Return(int(2), nil)

	svc := NewAuthorityService(mkClient(authMock, regMock), mkConfig())
	role, err := svc.FindRole(context.Background(), "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb1")
	assert.NoError(t, err)
	assert.Equal(t, domain.RoleIssuer, role)
}

func TestAuthorityService_FindRole_PropagatesError(t *testing.T) {
	authMock := &localAuthorityBinding{}
	regMock := &localRegistryBinding{}

	authMock.On("UserToRole", mock.Anything, mock.Anything).Return(int(0), errors.New("rpc down"))

	svc := NewAuthorityService(mkClient(authMock, regMock), mkConfig())
	_, err := svc.FindRole(context.Background(), "0x0000000000000000000000000000000000000000")
	assert.Error(t, err)
}

func TestAuthorityService_HasRoleOrAbove_True(t *testing.T) {
	authMock := &localAuthorityBinding{}
	regMock := &localRegistryBinding{}
	authMock.On("UserToRole", mock.Anything, mock.Anything).Return(int(3), nil) // Admin

	svc := NewAuthorityService(mkClient(authMock, regMock), mkConfig())
	assert.True(t, svc.HasRoleOrAbove(context.Background(), "0x0000000000000000000000000000000000000000", domain.RoleIssuer))
}

func TestAuthorityService_HasRoleOrAbove_FalseOnError(t *testing.T) {
	authMock := &localAuthorityBinding{}
	regMock := &localRegistryBinding{}
	authMock.On("UserToRole", mock.Anything, mock.Anything).Return(int(0), errors.New("err"))

	svc := NewAuthorityService(mkClient(authMock, regMock), mkConfig())
	assert.False(t, svc.HasRoleOrAbove(context.Background(), "0x0000000000000000000000000000000000000000", domain.RoleHolder))
}

func TestAuthorityService_FindNonce(t *testing.T) {
	authMock := &localAuthorityBinding{}
	regMock := &localRegistryBinding{}
	regMock.On("UserToNonce", mock.Anything, mock.Anything).Return(big.NewInt(42), nil)

	svc := NewAuthorityService(mkClient(authMock, regMock), mkConfig())
	got, err := svc.FindNonce(context.Background(), "0x0000000000000000000000000000000000000000")
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(42), got)
}

func TestAuthorityService_UpdateUserRole_EmptyIsNoOp(t *testing.T) {
	authMock := &localAuthorityBinding{}
	regMock := &localRegistryBinding{}

	svc := NewAuthorityService(mkClient(authMock, regMock), mkConfig())
	err := svc.UpdateUserRole(context.Background(), domain.Wallet{})
	assert.NoError(t, err)
	authMock.AssertNotCalled(t, "BatchUpdateUserRoleWithSignature", mock.Anything, mock.Anything)
}

func TestAuthorityService_UpdateUserRole_NonceFetchFails(t *testing.T) {
	authMock := &localAuthorityBinding{}
	regMock := &localRegistryBinding{}
	regMock.On("UserToNonce", mock.Anything, mock.Anything).Return(nil, errors.New("nonce err"))

	svc := NewAuthorityService(mkClient(authMock, regMock), mkConfig())
	err := svc.UpdateUserRole(context.Background(), domain.Wallet{
		Address:             "0x0000000000000000000000000000000000000000",
		EncryptedPrivateKey: "irrelevant",
	}, domain.User{Role: domain.RoleHolder})
	assert.Error(t, err)
}

func TestAuthorityService_UpdateUserRole_DecryptionFails(t *testing.T) {
	authMock := &localAuthorityBinding{}
	regMock := &localRegistryBinding{}
	regMock.On("UserToNonce", mock.Anything, mock.Anything).Return(big.NewInt(0), nil)

	svc := NewAuthorityService(mkClient(authMock, regMock), mkConfig())
	err := svc.UpdateUserRole(context.Background(), domain.Wallet{
		Address:             "0x0000000000000000000000000000000000000000",
		EncryptedPrivateKey: "ZZZZ", // invalid hex
	}, domain.User{Role: domain.RoleHolder})
	assert.Error(t, err)
}

func TestAuthorityService_UpdateUserRole_Success(t *testing.T) {
	authMock := &localAuthorityBinding{}
	regMock := &localRegistryBinding{}

	privKey, err := ethCrypto.GenerateKey()
	assert.NoError(t, err)
	privKeyHex := common.Bytes2Hex(ethCrypto.FromECDSA(privKey))
	signerAddr := ethCrypto.PubkeyToAddress(privKey.PublicKey).Hex()

	cfg := mkConfig()
	encrypted, err := cryptoInfra.Encrypt([]byte(privKeyHex), []byte(*cfg.WalletEncryptionKey))
	assert.NoError(t, err)

	regMock.On("UserToNonce", mock.Anything, mock.Anything).Return(big.NewInt(7), nil)
	authMock.On("BatchUpdateUserRoleWithSignature",
		mock.Anything,
		mock.MatchedBy(func(p contracts.CredentialAuthorityBatchUpdateUserRoleWithSignatureParams) bool {
			return p.Nonce.Cmp(big.NewInt(7)) == 0 && len(p.UserRoles) == 1 && len(p.Signature) == 65
		})).
		Return(&types.Transaction{}, nil)

	svc := NewAuthorityService(mkClient(authMock, regMock), cfg)
	err = svc.UpdateUserRole(context.Background(), domain.Wallet{
		Address:             signerAddr,
		EncryptedPrivateKey: encrypted,
	}, domain.User{
		WalletAddress: "0x000000000000000000000000000000000000bEEf",
		Role:          domain.RoleIssuer,
	})
	assert.NoError(t, err)
	authMock.AssertExpectations(t)
}
