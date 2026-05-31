package chain

import (
	"math/big"

	"CredChain_Golang/infrastructure/chain/contracts"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// AuthorityBinding abstracts the methods used from the abigen-generated
// CredentialAuthority contract binding so authority_service.go can be unit-tested.
// The concrete *contracts.Authority satisfies this interface structurally.
type AuthorityBinding interface {
	UserToRole(opts *bind.CallOpts, addr common.Address) (uint8, error)
	UserToNonce(opts *bind.CallOpts, addr common.Address) (*big.Int, error)
	BatchUpdateUserRoleWithSignature(
		opts *bind.TransactOpts,
		params contracts.CredentialAuthorityBatchUpdateUserRoleWithSignatureParams,
	) (*types.Transaction, error)
	TransferSuperAdminWithSignature(
		opts *bind.TransactOpts,
		params contracts.CredentialAuthorityTransferSuperAdminWithSignatureParams,
	) (*types.Transaction, error)
}

// RegistryBinding abstracts the methods used from the abigen-generated
// CredentialRegistry contract binding. The concrete *contracts.Registry satisfies
// this interface structurally.
type RegistryBinding interface {
	UserToNonce(opts *bind.CallOpts, addr common.Address) (*big.Int, error)
}
