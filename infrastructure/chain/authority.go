// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package chain

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// CredentialAuthorityBatchUpdateUserRoleWithSignatureParams is an auto generated low-level Go binding around an user-defined struct.
type CredentialAuthorityBatchUpdateUserRoleWithSignatureParams struct {
	Signer    common.Address
	UserRoles []CredentialAuthorityUserRoleUpdation
	Nonce     *big.Int
	Signature []byte
}

// CredentialAuthorityTransferSuperAdminWithSignatureParams is an auto generated low-level Go binding around an user-defined struct.
type CredentialAuthorityTransferSuperAdminWithSignatureParams struct {
	Signer        common.Address
	NewSuperAdmin common.Address
	Nonce         *big.Int
	Signature     []byte
}

// CredentialAuthorityUserRoleUpdation is an auto generated low-level Go binding around an user-defined struct.
type CredentialAuthorityUserRoleUpdation struct {
	Addr common.Address
	Role uint8
}

// AuthorityMetaData contains all meta data concerning the Authority contract.
var AuthorityMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"AdminUpdatePeerAdminRoleError\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ECDSAInvalidSignature\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"length\",\"type\":\"uint256\"}],\"name\":\"ECDSAInvalidSignatureLength\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"s\",\"type\":\"bytes32\"}],\"name\":\"ECDSAInvalidSignatureS\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidAddressError\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidInitialization\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidNonceError\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidSignatureError\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"MaxBatchExceededError\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotDeployerError\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotInitializing\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"RoleBelowAdminError\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"RoleBelowIssuerError\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"RoleNotSuperAdminError\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"SameRoleUpdateError\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"SuperAdminRoleNotUpdatableError\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"TransferSuperAdminToSelfError\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"version\",\"type\":\"uint64\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"oldSuperAdmin\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newSuperAdmin\",\"type\":\"address\"}],\"name\":\"SuperAdminTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"enumCredentialAuthority.Role\",\"name\":\"oldRole\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"enumCredentialAuthority.Role\",\"name\":\"newRole\",\"type\":\"uint8\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"updatedBy\",\"type\":\"address\"}],\"name\":\"UserRoleUpdated\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"MAX_BATCH_ROLE\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"signer\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"},{\"internalType\":\"enumCredentialAuthority.Role\",\"name\":\"role\",\"type\":\"uint8\"}],\"internalType\":\"structCredentialAuthority.UserRoleUpdation[]\",\"name\":\"userRoles\",\"type\":\"tuple[]\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"}],\"internalType\":\"structCredentialAuthority.BatchUpdateUserRoleWithSignatureParams\",\"name\":\"params\",\"type\":\"tuple\"}],\"name\":\"batchUpdateUserRoleWithSignature\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"config\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"},{\"internalType\":\"enumCredentialAuthority.Role\",\"name\":\"minimumRole\",\"type\":\"uint8\"}],\"name\":\"hasRoleOrAbove\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"superAdminUser\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_config\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"offset\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"limit\",\"type\":\"uint256\"}],\"name\":\"paginateUsers\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"\",\"type\":\"address[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"signer\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"newSuperAdmin\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"}],\"internalType\":\"structCredentialAuthority.TransferSuperAdminWithSignatureParams\",\"name\":\"params\",\"type\":\"tuple\"}],\"name\":\"transferSuperAdminWithSignature\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"userToIndex\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"userToNonce\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"userToRole\",\"outputs\":[{\"internalType\":\"enumCredentialAuthority.Role\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"users\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// AuthorityABI is the input ABI used to generate the binding from.
// Deprecated: Use AuthorityMetaData.ABI instead.
var AuthorityABI = AuthorityMetaData.ABI

// Authority is an auto generated Go binding around an Ethereum contract.
type Authority struct {
	AuthorityCaller     // Read-only binding to the contract
	AuthorityTransactor // Write-only binding to the contract
	AuthorityFilterer   // Log filterer for contract events
}

// AuthorityCaller is an auto generated read-only Go binding around an Ethereum contract.
type AuthorityCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AuthorityTransactor is an auto generated write-only Go binding around an Ethereum contract.
type AuthorityTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AuthorityFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type AuthorityFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AuthoritySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type AuthoritySession struct {
	Contract     *Authority        // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// AuthorityCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type AuthorityCallerSession struct {
	Contract *AuthorityCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts    // Call options to use throughout this session
}

// AuthorityTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type AuthorityTransactorSession struct {
	Contract     *AuthorityTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// AuthorityRaw is an auto generated low-level Go binding around an Ethereum contract.
type AuthorityRaw struct {
	Contract *Authority // Generic contract binding to access the raw methods on
}

// AuthorityCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type AuthorityCallerRaw struct {
	Contract *AuthorityCaller // Generic read-only contract binding to access the raw methods on
}

// AuthorityTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type AuthorityTransactorRaw struct {
	Contract *AuthorityTransactor // Generic write-only contract binding to access the raw methods on
}

// NewAuthority creates a new instance of Authority, bound to a specific deployed contract.
func NewAuthority(address common.Address, backend bind.ContractBackend) (*Authority, error) {
	contract, err := bindAuthority(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Authority{AuthorityCaller: AuthorityCaller{contract: contract}, AuthorityTransactor: AuthorityTransactor{contract: contract}, AuthorityFilterer: AuthorityFilterer{contract: contract}}, nil
}

// NewAuthorityCaller creates a new read-only instance of Authority, bound to a specific deployed contract.
func NewAuthorityCaller(address common.Address, caller bind.ContractCaller) (*AuthorityCaller, error) {
	contract, err := bindAuthority(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &AuthorityCaller{contract: contract}, nil
}

// NewAuthorityTransactor creates a new write-only instance of Authority, bound to a specific deployed contract.
func NewAuthorityTransactor(address common.Address, transactor bind.ContractTransactor) (*AuthorityTransactor, error) {
	contract, err := bindAuthority(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &AuthorityTransactor{contract: contract}, nil
}

// NewAuthorityFilterer creates a new log filterer instance of Authority, bound to a specific deployed contract.
func NewAuthorityFilterer(address common.Address, filterer bind.ContractFilterer) (*AuthorityFilterer, error) {
	contract, err := bindAuthority(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &AuthorityFilterer{contract: contract}, nil
}

// bindAuthority binds a generic wrapper to an already deployed contract.
func bindAuthority(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := AuthorityMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Authority *AuthorityRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Authority.Contract.AuthorityCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Authority *AuthorityRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Authority.Contract.AuthorityTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Authority *AuthorityRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Authority.Contract.AuthorityTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Authority *AuthorityCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Authority.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Authority *AuthorityTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Authority.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Authority *AuthorityTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Authority.Contract.contract.Transact(opts, method, params...)
}

// MAXBATCHROLE is a free data retrieval call binding the contract method 0xd07e9e0e.
//
// Solidity: function MAX_BATCH_ROLE() view returns(uint256)
func (_Authority *AuthorityCaller) MAXBATCHROLE(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Authority.contract.Call(opts, &out, "MAX_BATCH_ROLE")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MAXBATCHROLE is a free data retrieval call binding the contract method 0xd07e9e0e.
//
// Solidity: function MAX_BATCH_ROLE() view returns(uint256)
func (_Authority *AuthoritySession) MAXBATCHROLE() (*big.Int, error) {
	return _Authority.Contract.MAXBATCHROLE(&_Authority.CallOpts)
}

// MAXBATCHROLE is a free data retrieval call binding the contract method 0xd07e9e0e.
//
// Solidity: function MAX_BATCH_ROLE() view returns(uint256)
func (_Authority *AuthorityCallerSession) MAXBATCHROLE() (*big.Int, error) {
	return _Authority.Contract.MAXBATCHROLE(&_Authority.CallOpts)
}

// Config is a free data retrieval call binding the contract method 0x79502c55.
//
// Solidity: function config() view returns(address)
func (_Authority *AuthorityCaller) Config(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Authority.contract.Call(opts, &out, "config")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Config is a free data retrieval call binding the contract method 0x79502c55.
//
// Solidity: function config() view returns(address)
func (_Authority *AuthoritySession) Config() (common.Address, error) {
	return _Authority.Contract.Config(&_Authority.CallOpts)
}

// Config is a free data retrieval call binding the contract method 0x79502c55.
//
// Solidity: function config() view returns(address)
func (_Authority *AuthorityCallerSession) Config() (common.Address, error) {
	return _Authority.Contract.Config(&_Authority.CallOpts)
}

// HasRoleOrAbove is a free data retrieval call binding the contract method 0x6e24d2f1.
//
// Solidity: function hasRoleOrAbove(address user, uint8 minimumRole) view returns(bool)
func (_Authority *AuthorityCaller) HasRoleOrAbove(opts *bind.CallOpts, user common.Address, minimumRole uint8) (bool, error) {
	var out []interface{}
	err := _Authority.contract.Call(opts, &out, "hasRoleOrAbove", user, minimumRole)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// HasRoleOrAbove is a free data retrieval call binding the contract method 0x6e24d2f1.
//
// Solidity: function hasRoleOrAbove(address user, uint8 minimumRole) view returns(bool)
func (_Authority *AuthoritySession) HasRoleOrAbove(user common.Address, minimumRole uint8) (bool, error) {
	return _Authority.Contract.HasRoleOrAbove(&_Authority.CallOpts, user, minimumRole)
}

// HasRoleOrAbove is a free data retrieval call binding the contract method 0x6e24d2f1.
//
// Solidity: function hasRoleOrAbove(address user, uint8 minimumRole) view returns(bool)
func (_Authority *AuthorityCallerSession) HasRoleOrAbove(user common.Address, minimumRole uint8) (bool, error) {
	return _Authority.Contract.HasRoleOrAbove(&_Authority.CallOpts, user, minimumRole)
}

// PaginateUsers is a free data retrieval call binding the contract method 0xe940665f.
//
// Solidity: function paginateUsers(uint256 offset, uint256 limit) view returns(address[])
func (_Authority *AuthorityCaller) PaginateUsers(opts *bind.CallOpts, offset *big.Int, limit *big.Int) ([]common.Address, error) {
	var out []interface{}
	err := _Authority.contract.Call(opts, &out, "paginateUsers", offset, limit)

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// PaginateUsers is a free data retrieval call binding the contract method 0xe940665f.
//
// Solidity: function paginateUsers(uint256 offset, uint256 limit) view returns(address[])
func (_Authority *AuthoritySession) PaginateUsers(offset *big.Int, limit *big.Int) ([]common.Address, error) {
	return _Authority.Contract.PaginateUsers(&_Authority.CallOpts, offset, limit)
}

// PaginateUsers is a free data retrieval call binding the contract method 0xe940665f.
//
// Solidity: function paginateUsers(uint256 offset, uint256 limit) view returns(address[])
func (_Authority *AuthorityCallerSession) PaginateUsers(offset *big.Int, limit *big.Int) ([]common.Address, error) {
	return _Authority.Contract.PaginateUsers(&_Authority.CallOpts, offset, limit)
}

// UserToIndex is a free data retrieval call binding the contract method 0xca5da7af.
//
// Solidity: function userToIndex(address ) view returns(uint256)
func (_Authority *AuthorityCaller) UserToIndex(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Authority.contract.Call(opts, &out, "userToIndex", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// UserToIndex is a free data retrieval call binding the contract method 0xca5da7af.
//
// Solidity: function userToIndex(address ) view returns(uint256)
func (_Authority *AuthoritySession) UserToIndex(arg0 common.Address) (*big.Int, error) {
	return _Authority.Contract.UserToIndex(&_Authority.CallOpts, arg0)
}

// UserToIndex is a free data retrieval call binding the contract method 0xca5da7af.
//
// Solidity: function userToIndex(address ) view returns(uint256)
func (_Authority *AuthorityCallerSession) UserToIndex(arg0 common.Address) (*big.Int, error) {
	return _Authority.Contract.UserToIndex(&_Authority.CallOpts, arg0)
}

// UserToNonce is a free data retrieval call binding the contract method 0xd87a794f.
//
// Solidity: function userToNonce(address ) view returns(uint256)
func (_Authority *AuthorityCaller) UserToNonce(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Authority.contract.Call(opts, &out, "userToNonce", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// UserToNonce is a free data retrieval call binding the contract method 0xd87a794f.
//
// Solidity: function userToNonce(address ) view returns(uint256)
func (_Authority *AuthoritySession) UserToNonce(arg0 common.Address) (*big.Int, error) {
	return _Authority.Contract.UserToNonce(&_Authority.CallOpts, arg0)
}

// UserToNonce is a free data retrieval call binding the contract method 0xd87a794f.
//
// Solidity: function userToNonce(address ) view returns(uint256)
func (_Authority *AuthorityCallerSession) UserToNonce(arg0 common.Address) (*big.Int, error) {
	return _Authority.Contract.UserToNonce(&_Authority.CallOpts, arg0)
}

// UserToRole is a free data retrieval call binding the contract method 0x0cd12a2f.
//
// Solidity: function userToRole(address ) view returns(uint8)
func (_Authority *AuthorityCaller) UserToRole(opts *bind.CallOpts, arg0 common.Address) (uint8, error) {
	var out []interface{}
	err := _Authority.contract.Call(opts, &out, "userToRole", arg0)

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// UserToRole is a free data retrieval call binding the contract method 0x0cd12a2f.
//
// Solidity: function userToRole(address ) view returns(uint8)
func (_Authority *AuthoritySession) UserToRole(arg0 common.Address) (uint8, error) {
	return _Authority.Contract.UserToRole(&_Authority.CallOpts, arg0)
}

// UserToRole is a free data retrieval call binding the contract method 0x0cd12a2f.
//
// Solidity: function userToRole(address ) view returns(uint8)
func (_Authority *AuthorityCallerSession) UserToRole(arg0 common.Address) (uint8, error) {
	return _Authority.Contract.UserToRole(&_Authority.CallOpts, arg0)
}

// Users is a free data retrieval call binding the contract method 0x365b98b2.
//
// Solidity: function users(uint256 ) view returns(address)
func (_Authority *AuthorityCaller) Users(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _Authority.contract.Call(opts, &out, "users", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Users is a free data retrieval call binding the contract method 0x365b98b2.
//
// Solidity: function users(uint256 ) view returns(address)
func (_Authority *AuthoritySession) Users(arg0 *big.Int) (common.Address, error) {
	return _Authority.Contract.Users(&_Authority.CallOpts, arg0)
}

// Users is a free data retrieval call binding the contract method 0x365b98b2.
//
// Solidity: function users(uint256 ) view returns(address)
func (_Authority *AuthorityCallerSession) Users(arg0 *big.Int) (common.Address, error) {
	return _Authority.Contract.Users(&_Authority.CallOpts, arg0)
}

// BatchUpdateUserRoleWithSignature is a paid mutator transaction binding the contract method 0x1bfdfa6b.
//
// Solidity: function batchUpdateUserRoleWithSignature((address,(address,uint8)[],uint256,bytes) params) returns()
func (_Authority *AuthorityTransactor) BatchUpdateUserRoleWithSignature(opts *bind.TransactOpts, params CredentialAuthorityBatchUpdateUserRoleWithSignatureParams) (*types.Transaction, error) {
	return _Authority.contract.Transact(opts, "batchUpdateUserRoleWithSignature", params)
}

// BatchUpdateUserRoleWithSignature is a paid mutator transaction binding the contract method 0x1bfdfa6b.
//
// Solidity: function batchUpdateUserRoleWithSignature((address,(address,uint8)[],uint256,bytes) params) returns()
func (_Authority *AuthoritySession) BatchUpdateUserRoleWithSignature(params CredentialAuthorityBatchUpdateUserRoleWithSignatureParams) (*types.Transaction, error) {
	return _Authority.Contract.BatchUpdateUserRoleWithSignature(&_Authority.TransactOpts, params)
}

// BatchUpdateUserRoleWithSignature is a paid mutator transaction binding the contract method 0x1bfdfa6b.
//
// Solidity: function batchUpdateUserRoleWithSignature((address,(address,uint8)[],uint256,bytes) params) returns()
func (_Authority *AuthorityTransactorSession) BatchUpdateUserRoleWithSignature(params CredentialAuthorityBatchUpdateUserRoleWithSignatureParams) (*types.Transaction, error) {
	return _Authority.Contract.BatchUpdateUserRoleWithSignature(&_Authority.TransactOpts, params)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address superAdminUser, address _config) returns()
func (_Authority *AuthorityTransactor) Initialize(opts *bind.TransactOpts, superAdminUser common.Address, _config common.Address) (*types.Transaction, error) {
	return _Authority.contract.Transact(opts, "initialize", superAdminUser, _config)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address superAdminUser, address _config) returns()
func (_Authority *AuthoritySession) Initialize(superAdminUser common.Address, _config common.Address) (*types.Transaction, error) {
	return _Authority.Contract.Initialize(&_Authority.TransactOpts, superAdminUser, _config)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address superAdminUser, address _config) returns()
func (_Authority *AuthorityTransactorSession) Initialize(superAdminUser common.Address, _config common.Address) (*types.Transaction, error) {
	return _Authority.Contract.Initialize(&_Authority.TransactOpts, superAdminUser, _config)
}

// TransferSuperAdminWithSignature is a paid mutator transaction binding the contract method 0x86dc7a91.
//
// Solidity: function transferSuperAdminWithSignature((address,address,uint256,bytes) params) returns()
func (_Authority *AuthorityTransactor) TransferSuperAdminWithSignature(opts *bind.TransactOpts, params CredentialAuthorityTransferSuperAdminWithSignatureParams) (*types.Transaction, error) {
	return _Authority.contract.Transact(opts, "transferSuperAdminWithSignature", params)
}

// TransferSuperAdminWithSignature is a paid mutator transaction binding the contract method 0x86dc7a91.
//
// Solidity: function transferSuperAdminWithSignature((address,address,uint256,bytes) params) returns()
func (_Authority *AuthoritySession) TransferSuperAdminWithSignature(params CredentialAuthorityTransferSuperAdminWithSignatureParams) (*types.Transaction, error) {
	return _Authority.Contract.TransferSuperAdminWithSignature(&_Authority.TransactOpts, params)
}

// TransferSuperAdminWithSignature is a paid mutator transaction binding the contract method 0x86dc7a91.
//
// Solidity: function transferSuperAdminWithSignature((address,address,uint256,bytes) params) returns()
func (_Authority *AuthorityTransactorSession) TransferSuperAdminWithSignature(params CredentialAuthorityTransferSuperAdminWithSignatureParams) (*types.Transaction, error) {
	return _Authority.Contract.TransferSuperAdminWithSignature(&_Authority.TransactOpts, params)
}

// AuthorityInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the Authority contract.
type AuthorityInitializedIterator struct {
	Event *AuthorityInitialized // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AuthorityInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AuthorityInitialized)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AuthorityInitialized)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AuthorityInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AuthorityInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AuthorityInitialized represents a Initialized event raised by the Authority contract.
type AuthorityInitialized struct {
	Version uint64
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_Authority *AuthorityFilterer) FilterInitialized(opts *bind.FilterOpts) (*AuthorityInitializedIterator, error) {

	logs, sub, err := _Authority.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &AuthorityInitializedIterator{contract: _Authority.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_Authority *AuthorityFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *AuthorityInitialized) (event.Subscription, error) {

	logs, sub, err := _Authority.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AuthorityInitialized)
				if err := _Authority.contract.UnpackLog(event, "Initialized", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseInitialized is a log parse operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_Authority *AuthorityFilterer) ParseInitialized(log types.Log) (*AuthorityInitialized, error) {
	event := new(AuthorityInitialized)
	if err := _Authority.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AuthoritySuperAdminTransferredIterator is returned from FilterSuperAdminTransferred and is used to iterate over the raw logs and unpacked data for SuperAdminTransferred events raised by the Authority contract.
type AuthoritySuperAdminTransferredIterator struct {
	Event *AuthoritySuperAdminTransferred // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AuthoritySuperAdminTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AuthoritySuperAdminTransferred)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AuthoritySuperAdminTransferred)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AuthoritySuperAdminTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AuthoritySuperAdminTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AuthoritySuperAdminTransferred represents a SuperAdminTransferred event raised by the Authority contract.
type AuthoritySuperAdminTransferred struct {
	OldSuperAdmin common.Address
	NewSuperAdmin common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterSuperAdminTransferred is a free log retrieval operation binding the contract event 0x0f62530a074f4e1e883a8c916fa7f8639d52598edb7f9b5aa3148d991db5610d.
//
// Solidity: event SuperAdminTransferred(address indexed oldSuperAdmin, address indexed newSuperAdmin)
func (_Authority *AuthorityFilterer) FilterSuperAdminTransferred(opts *bind.FilterOpts, oldSuperAdmin []common.Address, newSuperAdmin []common.Address) (*AuthoritySuperAdminTransferredIterator, error) {

	var oldSuperAdminRule []interface{}
	for _, oldSuperAdminItem := range oldSuperAdmin {
		oldSuperAdminRule = append(oldSuperAdminRule, oldSuperAdminItem)
	}
	var newSuperAdminRule []interface{}
	for _, newSuperAdminItem := range newSuperAdmin {
		newSuperAdminRule = append(newSuperAdminRule, newSuperAdminItem)
	}

	logs, sub, err := _Authority.contract.FilterLogs(opts, "SuperAdminTransferred", oldSuperAdminRule, newSuperAdminRule)
	if err != nil {
		return nil, err
	}
	return &AuthoritySuperAdminTransferredIterator{contract: _Authority.contract, event: "SuperAdminTransferred", logs: logs, sub: sub}, nil
}

// WatchSuperAdminTransferred is a free log subscription operation binding the contract event 0x0f62530a074f4e1e883a8c916fa7f8639d52598edb7f9b5aa3148d991db5610d.
//
// Solidity: event SuperAdminTransferred(address indexed oldSuperAdmin, address indexed newSuperAdmin)
func (_Authority *AuthorityFilterer) WatchSuperAdminTransferred(opts *bind.WatchOpts, sink chan<- *AuthoritySuperAdminTransferred, oldSuperAdmin []common.Address, newSuperAdmin []common.Address) (event.Subscription, error) {

	var oldSuperAdminRule []interface{}
	for _, oldSuperAdminItem := range oldSuperAdmin {
		oldSuperAdminRule = append(oldSuperAdminRule, oldSuperAdminItem)
	}
	var newSuperAdminRule []interface{}
	for _, newSuperAdminItem := range newSuperAdmin {
		newSuperAdminRule = append(newSuperAdminRule, newSuperAdminItem)
	}

	logs, sub, err := _Authority.contract.WatchLogs(opts, "SuperAdminTransferred", oldSuperAdminRule, newSuperAdminRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AuthoritySuperAdminTransferred)
				if err := _Authority.contract.UnpackLog(event, "SuperAdminTransferred", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseSuperAdminTransferred is a log parse operation binding the contract event 0x0f62530a074f4e1e883a8c916fa7f8639d52598edb7f9b5aa3148d991db5610d.
//
// Solidity: event SuperAdminTransferred(address indexed oldSuperAdmin, address indexed newSuperAdmin)
func (_Authority *AuthorityFilterer) ParseSuperAdminTransferred(log types.Log) (*AuthoritySuperAdminTransferred, error) {
	event := new(AuthoritySuperAdminTransferred)
	if err := _Authority.contract.UnpackLog(event, "SuperAdminTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AuthorityUserRoleUpdatedIterator is returned from FilterUserRoleUpdated and is used to iterate over the raw logs and unpacked data for UserRoleUpdated events raised by the Authority contract.
type AuthorityUserRoleUpdatedIterator struct {
	Event *AuthorityUserRoleUpdated // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *AuthorityUserRoleUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AuthorityUserRoleUpdated)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(AuthorityUserRoleUpdated)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *AuthorityUserRoleUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AuthorityUserRoleUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AuthorityUserRoleUpdated represents a UserRoleUpdated event raised by the Authority contract.
type AuthorityUserRoleUpdated struct {
	User      common.Address
	OldRole   uint8
	NewRole   uint8
	UpdatedBy common.Address
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterUserRoleUpdated is a free log retrieval operation binding the contract event 0x9aebd01cdf037f450af9b51570e16317b26feaf5fb81a8229143ec20b000e7af.
//
// Solidity: event UserRoleUpdated(address indexed user, uint8 oldRole, uint8 newRole, address indexed updatedBy)
func (_Authority *AuthorityFilterer) FilterUserRoleUpdated(opts *bind.FilterOpts, user []common.Address, updatedBy []common.Address) (*AuthorityUserRoleUpdatedIterator, error) {

	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}

	var updatedByRule []interface{}
	for _, updatedByItem := range updatedBy {
		updatedByRule = append(updatedByRule, updatedByItem)
	}

	logs, sub, err := _Authority.contract.FilterLogs(opts, "UserRoleUpdated", userRule, updatedByRule)
	if err != nil {
		return nil, err
	}
	return &AuthorityUserRoleUpdatedIterator{contract: _Authority.contract, event: "UserRoleUpdated", logs: logs, sub: sub}, nil
}

// WatchUserRoleUpdated is a free log subscription operation binding the contract event 0x9aebd01cdf037f450af9b51570e16317b26feaf5fb81a8229143ec20b000e7af.
//
// Solidity: event UserRoleUpdated(address indexed user, uint8 oldRole, uint8 newRole, address indexed updatedBy)
func (_Authority *AuthorityFilterer) WatchUserRoleUpdated(opts *bind.WatchOpts, sink chan<- *AuthorityUserRoleUpdated, user []common.Address, updatedBy []common.Address) (event.Subscription, error) {

	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}

	var updatedByRule []interface{}
	for _, updatedByItem := range updatedBy {
		updatedByRule = append(updatedByRule, updatedByItem)
	}

	logs, sub, err := _Authority.contract.WatchLogs(opts, "UserRoleUpdated", userRule, updatedByRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AuthorityUserRoleUpdated)
				if err := _Authority.contract.UnpackLog(event, "UserRoleUpdated", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseUserRoleUpdated is a log parse operation binding the contract event 0x9aebd01cdf037f450af9b51570e16317b26feaf5fb81a8229143ec20b000e7af.
//
// Solidity: event UserRoleUpdated(address indexed user, uint8 oldRole, uint8 newRole, address indexed updatedBy)
func (_Authority *AuthorityFilterer) ParseUserRoleUpdated(log types.Log) (*AuthorityUserRoleUpdated, error) {
	event := new(AuthorityUserRoleUpdated)
	if err := _Authority.contract.UnpackLog(event, "UserRoleUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
