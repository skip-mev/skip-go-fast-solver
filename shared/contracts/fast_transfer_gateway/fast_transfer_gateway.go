// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package fast_transfer_gateway

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

// FastTransferOrder is an auto generated low-level Go binding around an user-defined struct.
type FastTransferOrder struct {
	Sender            [32]byte
	Recipient         [32]byte
	AmountIn          *big.Int
	AmountOut         *big.Int
	Nonce             uint32
	SourceDomain      uint32
	DestinationDomain uint32
	TimeoutTimestamp  uint64
	Data              []byte
}

// FastTransferGatewayMetaData contains all meta data concerning the FastTransferGateway contract.
var FastTransferGatewayMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"PERMIT2\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractIPermit2\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"UPGRADE_INTERFACE_VERSION\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"fillOrder\",\"inputs\":[{\"name\":\"filler\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"order\",\"type\":\"tuple\",\"internalType\":\"structFastTransferOrder\",\"components\":[{\"name\":\"sender\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"recipient\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"amountIn\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"amountOut\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"nonce\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"sourceDomain\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"destinationDomain\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"timeoutTimestamp\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"goFastCaller\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractGoFastCaller\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"handle\",\"inputs\":[{\"name\":\"_origin\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"_sender\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_message\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"initialize\",\"inputs\":[{\"name\":\"_localDomain\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"_owner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_token\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_mailbox\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_interchainSecurityModule\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_permit2\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_goFastCaller\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"initiateSettlement\",\"inputs\":[{\"name\":\"repaymentAddress\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"orderIDs\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"initiateTimeout\",\"inputs\":[{\"name\":\"orders\",\"type\":\"tuple[]\",\"internalType\":\"structFastTransferOrder[]\",\"components\":[{\"name\":\"sender\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"recipient\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"amountIn\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"amountOut\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"nonce\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"sourceDomain\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"destinationDomain\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"timeoutTimestamp\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]}],\"outputs\":[],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"interchainSecurityModule\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"localDomain\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"mailbox\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"nonce\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"orderFills\",\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"orderID\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"filler\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"sourceDomain\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"orderStatuses\",\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint8\",\"internalType\":\"enumOrderStatus\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"owner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"proxiableUUID\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"quoteInitiateSettlement\",\"inputs\":[{\"name\":\"sourceDomain\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"repaymentAddress\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"orderIDs\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"quoteInitiateTimeout\",\"inputs\":[{\"name\":\"sourceDomain\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"orders\",\"type\":\"tuple[]\",\"internalType\":\"structFastTransferOrder[]\",\"components\":[{\"name\":\"sender\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"recipient\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"amountIn\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"amountOut\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"nonce\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"sourceDomain\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"destinationDomain\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"timeoutTimestamp\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"remoteDomains\",\"inputs\":[{\"name\":\"\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"renounceOwnership\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setInterchainSecurityModule\",\"inputs\":[{\"name\":\"_interchainSecurityModule\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setMailbox\",\"inputs\":[{\"name\":\"_mailbox\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setRemoteDomain\",\"inputs\":[{\"name\":\"domain\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"remoteContract\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"settlementDetails\",\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"sender\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"nonce\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"destinationDomain\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"submitOrder\",\"inputs\":[{\"name\":\"sender\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"recipient\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"amountIn\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"amountOut\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"destinationDomain\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"timeoutTimestamp\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"submitOrderWithPermit\",\"inputs\":[{\"name\":\"sender\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"recipient\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"amountIn\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"amountOut\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"destinationDomain\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"timeoutTimestamp\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"permitDeadline\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"signature\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"token\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"transferOwnership\",\"inputs\":[{\"name\":\"newOwner\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"upgradeToAndCall\",\"inputs\":[{\"name\":\"newImplementation\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"payable\"},{\"type\":\"event\",\"name\":\"Initialized\",\"inputs\":[{\"name\":\"version\",\"type\":\"uint64\",\"indexed\":false,\"internalType\":\"uint64\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"OrderAlreadySettled\",\"inputs\":[{\"name\":\"orderID\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"OrderRefunded\",\"inputs\":[{\"name\":\"orderID\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"OrderSettled\",\"inputs\":[{\"name\":\"orderID\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"OrderSubmitted\",\"inputs\":[{\"name\":\"orderID\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"order\",\"type\":\"bytes\",\"indexed\":false,\"internalType\":\"bytes\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"OwnershipTransferred\",\"inputs\":[{\"name\":\"previousOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"newOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Upgraded\",\"inputs\":[{\"name\":\"implementation\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"AddressEmptyCode\",\"inputs\":[{\"name\":\"target\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"AddressInsufficientBalance\",\"inputs\":[{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"ERC1967InvalidImplementation\",\"inputs\":[{\"name\":\"implementation\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"ERC1967NonPayable\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"FailedInnerCall\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidInitialization\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"NotInitializing\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"OwnableInvalidOwner\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"OwnableUnauthorizedAccount\",\"inputs\":[{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"ReentrancyGuardReentrantCall\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"SafeERC20FailedOperation\",\"inputs\":[{\"name\":\"token\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"UUPSUnauthorizedCallContext\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"UUPSUnsupportedProxiableUUID\",\"inputs\":[{\"name\":\"slot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]}]",
}

// FastTransferGatewayABI is the input ABI used to generate the binding from.
// Deprecated: Use FastTransferGatewayMetaData.ABI instead.
var FastTransferGatewayABI = FastTransferGatewayMetaData.ABI

// FastTransferGateway is an auto generated Go binding around an Ethereum contract.
type FastTransferGateway struct {
	FastTransferGatewayCaller     // Read-only binding to the contract
	FastTransferGatewayTransactor // Write-only binding to the contract
	FastTransferGatewayFilterer   // Log filterer for contract events
}

// FastTransferGatewayCaller is an auto generated read-only Go binding around an Ethereum contract.
type FastTransferGatewayCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// FastTransferGatewayTransactor is an auto generated write-only Go binding around an Ethereum contract.
type FastTransferGatewayTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// FastTransferGatewayFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type FastTransferGatewayFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// FastTransferGatewaySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type FastTransferGatewaySession struct {
	Contract     *FastTransferGateway // Generic contract binding to set the session for
	CallOpts     bind.CallOpts        // Call options to use throughout this session
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// FastTransferGatewayCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type FastTransferGatewayCallerSession struct {
	Contract *FastTransferGatewayCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts              // Call options to use throughout this session
}

// FastTransferGatewayTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type FastTransferGatewayTransactorSession struct {
	Contract     *FastTransferGatewayTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts              // Transaction auth options to use throughout this session
}

// FastTransferGatewayRaw is an auto generated low-level Go binding around an Ethereum contract.
type FastTransferGatewayRaw struct {
	Contract *FastTransferGateway // Generic contract binding to access the raw methods on
}

// FastTransferGatewayCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type FastTransferGatewayCallerRaw struct {
	Contract *FastTransferGatewayCaller // Generic read-only contract binding to access the raw methods on
}

// FastTransferGatewayTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type FastTransferGatewayTransactorRaw struct {
	Contract *FastTransferGatewayTransactor // Generic write-only contract binding to access the raw methods on
}

// NewFastTransferGateway creates a new instance of FastTransferGateway, bound to a specific deployed contract.
func NewFastTransferGateway(address common.Address, backend bind.ContractBackend) (*FastTransferGateway, error) {
	contract, err := bindFastTransferGateway(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &FastTransferGateway{FastTransferGatewayCaller: FastTransferGatewayCaller{contract: contract}, FastTransferGatewayTransactor: FastTransferGatewayTransactor{contract: contract}, FastTransferGatewayFilterer: FastTransferGatewayFilterer{contract: contract}}, nil
}

// NewFastTransferGatewayCaller creates a new read-only instance of FastTransferGateway, bound to a specific deployed contract.
func NewFastTransferGatewayCaller(address common.Address, caller bind.ContractCaller) (*FastTransferGatewayCaller, error) {
	contract, err := bindFastTransferGateway(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &FastTransferGatewayCaller{contract: contract}, nil
}

// NewFastTransferGatewayTransactor creates a new write-only instance of FastTransferGateway, bound to a specific deployed contract.
func NewFastTransferGatewayTransactor(address common.Address, transactor bind.ContractTransactor) (*FastTransferGatewayTransactor, error) {
	contract, err := bindFastTransferGateway(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &FastTransferGatewayTransactor{contract: contract}, nil
}

// NewFastTransferGatewayFilterer creates a new log filterer instance of FastTransferGateway, bound to a specific deployed contract.
func NewFastTransferGatewayFilterer(address common.Address, filterer bind.ContractFilterer) (*FastTransferGatewayFilterer, error) {
	contract, err := bindFastTransferGateway(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &FastTransferGatewayFilterer{contract: contract}, nil
}

// bindFastTransferGateway binds a generic wrapper to an already deployed contract.
func bindFastTransferGateway(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := FastTransferGatewayMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_FastTransferGateway *FastTransferGatewayRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _FastTransferGateway.Contract.FastTransferGatewayCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_FastTransferGateway *FastTransferGatewayRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _FastTransferGateway.Contract.FastTransferGatewayTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_FastTransferGateway *FastTransferGatewayRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _FastTransferGateway.Contract.FastTransferGatewayTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_FastTransferGateway *FastTransferGatewayCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _FastTransferGateway.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_FastTransferGateway *FastTransferGatewayTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _FastTransferGateway.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_FastTransferGateway *FastTransferGatewayTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _FastTransferGateway.Contract.contract.Transact(opts, method, params...)
}

// PERMIT2 is a free data retrieval call binding the contract method 0x6afdd850.
//
// Solidity: function PERMIT2() view returns(address)
func (_FastTransferGateway *FastTransferGatewayCaller) PERMIT2(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _FastTransferGateway.contract.Call(opts, &out, "PERMIT2")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// PERMIT2 is a free data retrieval call binding the contract method 0x6afdd850.
//
// Solidity: function PERMIT2() view returns(address)
func (_FastTransferGateway *FastTransferGatewaySession) PERMIT2() (common.Address, error) {
	return _FastTransferGateway.Contract.PERMIT2(&_FastTransferGateway.CallOpts)
}

// PERMIT2 is a free data retrieval call binding the contract method 0x6afdd850.
//
// Solidity: function PERMIT2() view returns(address)
func (_FastTransferGateway *FastTransferGatewayCallerSession) PERMIT2() (common.Address, error) {
	return _FastTransferGateway.Contract.PERMIT2(&_FastTransferGateway.CallOpts)
}

// UPGRADEINTERFACEVERSION is a free data retrieval call binding the contract method 0xad3cb1cc.
//
// Solidity: function UPGRADE_INTERFACE_VERSION() view returns(string)
func (_FastTransferGateway *FastTransferGatewayCaller) UPGRADEINTERFACEVERSION(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _FastTransferGateway.contract.Call(opts, &out, "UPGRADE_INTERFACE_VERSION")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// UPGRADEINTERFACEVERSION is a free data retrieval call binding the contract method 0xad3cb1cc.
//
// Solidity: function UPGRADE_INTERFACE_VERSION() view returns(string)
func (_FastTransferGateway *FastTransferGatewaySession) UPGRADEINTERFACEVERSION() (string, error) {
	return _FastTransferGateway.Contract.UPGRADEINTERFACEVERSION(&_FastTransferGateway.CallOpts)
}

// UPGRADEINTERFACEVERSION is a free data retrieval call binding the contract method 0xad3cb1cc.
//
// Solidity: function UPGRADE_INTERFACE_VERSION() view returns(string)
func (_FastTransferGateway *FastTransferGatewayCallerSession) UPGRADEINTERFACEVERSION() (string, error) {
	return _FastTransferGateway.Contract.UPGRADEINTERFACEVERSION(&_FastTransferGateway.CallOpts)
}

// GoFastCaller is a free data retrieval call binding the contract method 0xc87d1240.
//
// Solidity: function goFastCaller() view returns(address)
func (_FastTransferGateway *FastTransferGatewayCaller) GoFastCaller(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _FastTransferGateway.contract.Call(opts, &out, "goFastCaller")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GoFastCaller is a free data retrieval call binding the contract method 0xc87d1240.
//
// Solidity: function goFastCaller() view returns(address)
func (_FastTransferGateway *FastTransferGatewaySession) GoFastCaller() (common.Address, error) {
	return _FastTransferGateway.Contract.GoFastCaller(&_FastTransferGateway.CallOpts)
}

// GoFastCaller is a free data retrieval call binding the contract method 0xc87d1240.
//
// Solidity: function goFastCaller() view returns(address)
func (_FastTransferGateway *FastTransferGatewayCallerSession) GoFastCaller() (common.Address, error) {
	return _FastTransferGateway.Contract.GoFastCaller(&_FastTransferGateway.CallOpts)
}

// InterchainSecurityModule is a free data retrieval call binding the contract method 0xde523cf3.
//
// Solidity: function interchainSecurityModule() view returns(address)
func (_FastTransferGateway *FastTransferGatewayCaller) InterchainSecurityModule(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _FastTransferGateway.contract.Call(opts, &out, "interchainSecurityModule")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// InterchainSecurityModule is a free data retrieval call binding the contract method 0xde523cf3.
//
// Solidity: function interchainSecurityModule() view returns(address)
func (_FastTransferGateway *FastTransferGatewaySession) InterchainSecurityModule() (common.Address, error) {
	return _FastTransferGateway.Contract.InterchainSecurityModule(&_FastTransferGateway.CallOpts)
}

// InterchainSecurityModule is a free data retrieval call binding the contract method 0xde523cf3.
//
// Solidity: function interchainSecurityModule() view returns(address)
func (_FastTransferGateway *FastTransferGatewayCallerSession) InterchainSecurityModule() (common.Address, error) {
	return _FastTransferGateway.Contract.InterchainSecurityModule(&_FastTransferGateway.CallOpts)
}

// LocalDomain is a free data retrieval call binding the contract method 0x8d3638f4.
//
// Solidity: function localDomain() view returns(uint32)
func (_FastTransferGateway *FastTransferGatewayCaller) LocalDomain(opts *bind.CallOpts) (uint32, error) {
	var out []interface{}
	err := _FastTransferGateway.contract.Call(opts, &out, "localDomain")

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// LocalDomain is a free data retrieval call binding the contract method 0x8d3638f4.
//
// Solidity: function localDomain() view returns(uint32)
func (_FastTransferGateway *FastTransferGatewaySession) LocalDomain() (uint32, error) {
	return _FastTransferGateway.Contract.LocalDomain(&_FastTransferGateway.CallOpts)
}

// LocalDomain is a free data retrieval call binding the contract method 0x8d3638f4.
//
// Solidity: function localDomain() view returns(uint32)
func (_FastTransferGateway *FastTransferGatewayCallerSession) LocalDomain() (uint32, error) {
	return _FastTransferGateway.Contract.LocalDomain(&_FastTransferGateway.CallOpts)
}

// Mailbox is a free data retrieval call binding the contract method 0xd5438eae.
//
// Solidity: function mailbox() view returns(address)
func (_FastTransferGateway *FastTransferGatewayCaller) Mailbox(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _FastTransferGateway.contract.Call(opts, &out, "mailbox")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Mailbox is a free data retrieval call binding the contract method 0xd5438eae.
//
// Solidity: function mailbox() view returns(address)
func (_FastTransferGateway *FastTransferGatewaySession) Mailbox() (common.Address, error) {
	return _FastTransferGateway.Contract.Mailbox(&_FastTransferGateway.CallOpts)
}

// Mailbox is a free data retrieval call binding the contract method 0xd5438eae.
//
// Solidity: function mailbox() view returns(address)
func (_FastTransferGateway *FastTransferGatewayCallerSession) Mailbox() (common.Address, error) {
	return _FastTransferGateway.Contract.Mailbox(&_FastTransferGateway.CallOpts)
}

// Nonce is a free data retrieval call binding the contract method 0xaffed0e0.
//
// Solidity: function nonce() view returns(uint32)
func (_FastTransferGateway *FastTransferGatewayCaller) Nonce(opts *bind.CallOpts) (uint32, error) {
	var out []interface{}
	err := _FastTransferGateway.contract.Call(opts, &out, "nonce")

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// Nonce is a free data retrieval call binding the contract method 0xaffed0e0.
//
// Solidity: function nonce() view returns(uint32)
func (_FastTransferGateway *FastTransferGatewaySession) Nonce() (uint32, error) {
	return _FastTransferGateway.Contract.Nonce(&_FastTransferGateway.CallOpts)
}

// Nonce is a free data retrieval call binding the contract method 0xaffed0e0.
//
// Solidity: function nonce() view returns(uint32)
func (_FastTransferGateway *FastTransferGatewayCallerSession) Nonce() (uint32, error) {
	return _FastTransferGateway.Contract.Nonce(&_FastTransferGateway.CallOpts)
}

// OrderFills is a free data retrieval call binding the contract method 0xf7213db6.
//
// Solidity: function orderFills(bytes32 ) view returns(bytes32 orderID, address filler, uint32 sourceDomain)
func (_FastTransferGateway *FastTransferGatewayCaller) OrderFills(opts *bind.CallOpts, arg0 [32]byte) (struct {
	OrderID      [32]byte
	Filler       common.Address
	SourceDomain uint32
}, error) {
	var out []interface{}
	err := _FastTransferGateway.contract.Call(opts, &out, "orderFills", arg0)

	outstruct := new(struct {
		OrderID      [32]byte
		Filler       common.Address
		SourceDomain uint32
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.OrderID = *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)
	outstruct.Filler = *abi.ConvertType(out[1], new(common.Address)).(*common.Address)
	outstruct.SourceDomain = *abi.ConvertType(out[2], new(uint32)).(*uint32)

	return *outstruct, err

}

// OrderFills is a free data retrieval call binding the contract method 0xf7213db6.
//
// Solidity: function orderFills(bytes32 ) view returns(bytes32 orderID, address filler, uint32 sourceDomain)
func (_FastTransferGateway *FastTransferGatewaySession) OrderFills(arg0 [32]byte) (struct {
	OrderID      [32]byte
	Filler       common.Address
	SourceDomain uint32
}, error) {
	return _FastTransferGateway.Contract.OrderFills(&_FastTransferGateway.CallOpts, arg0)
}

// OrderFills is a free data retrieval call binding the contract method 0xf7213db6.
//
// Solidity: function orderFills(bytes32 ) view returns(bytes32 orderID, address filler, uint32 sourceDomain)
func (_FastTransferGateway *FastTransferGatewayCallerSession) OrderFills(arg0 [32]byte) (struct {
	OrderID      [32]byte
	Filler       common.Address
	SourceDomain uint32
}, error) {
	return _FastTransferGateway.Contract.OrderFills(&_FastTransferGateway.CallOpts, arg0)
}

// OrderStatuses is a free data retrieval call binding the contract method 0x7f665ee5.
//
// Solidity: function orderStatuses(bytes32 ) view returns(uint8)
func (_FastTransferGateway *FastTransferGatewayCaller) OrderStatuses(opts *bind.CallOpts, arg0 [32]byte) (uint8, error) {
	var out []interface{}
	err := _FastTransferGateway.contract.Call(opts, &out, "orderStatuses", arg0)

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// OrderStatuses is a free data retrieval call binding the contract method 0x7f665ee5.
//
// Solidity: function orderStatuses(bytes32 ) view returns(uint8)
func (_FastTransferGateway *FastTransferGatewaySession) OrderStatuses(arg0 [32]byte) (uint8, error) {
	return _FastTransferGateway.Contract.OrderStatuses(&_FastTransferGateway.CallOpts, arg0)
}

// OrderStatuses is a free data retrieval call binding the contract method 0x7f665ee5.
//
// Solidity: function orderStatuses(bytes32 ) view returns(uint8)
func (_FastTransferGateway *FastTransferGatewayCallerSession) OrderStatuses(arg0 [32]byte) (uint8, error) {
	return _FastTransferGateway.Contract.OrderStatuses(&_FastTransferGateway.CallOpts, arg0)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_FastTransferGateway *FastTransferGatewayCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _FastTransferGateway.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_FastTransferGateway *FastTransferGatewaySession) Owner() (common.Address, error) {
	return _FastTransferGateway.Contract.Owner(&_FastTransferGateway.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_FastTransferGateway *FastTransferGatewayCallerSession) Owner() (common.Address, error) {
	return _FastTransferGateway.Contract.Owner(&_FastTransferGateway.CallOpts)
}

// ProxiableUUID is a free data retrieval call binding the contract method 0x52d1902d.
//
// Solidity: function proxiableUUID() view returns(bytes32)
func (_FastTransferGateway *FastTransferGatewayCaller) ProxiableUUID(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _FastTransferGateway.contract.Call(opts, &out, "proxiableUUID")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// ProxiableUUID is a free data retrieval call binding the contract method 0x52d1902d.
//
// Solidity: function proxiableUUID() view returns(bytes32)
func (_FastTransferGateway *FastTransferGatewaySession) ProxiableUUID() ([32]byte, error) {
	return _FastTransferGateway.Contract.ProxiableUUID(&_FastTransferGateway.CallOpts)
}

// ProxiableUUID is a free data retrieval call binding the contract method 0x52d1902d.
//
// Solidity: function proxiableUUID() view returns(bytes32)
func (_FastTransferGateway *FastTransferGatewayCallerSession) ProxiableUUID() ([32]byte, error) {
	return _FastTransferGateway.Contract.ProxiableUUID(&_FastTransferGateway.CallOpts)
}

// QuoteInitiateSettlement is a free data retrieval call binding the contract method 0xe88787c2.
//
// Solidity: function quoteInitiateSettlement(uint32 sourceDomain, bytes32 repaymentAddress, bytes orderIDs) view returns(uint256)
func (_FastTransferGateway *FastTransferGatewayCaller) QuoteInitiateSettlement(opts *bind.CallOpts, sourceDomain uint32, repaymentAddress [32]byte, orderIDs []byte) (*big.Int, error) {
	var out []interface{}
	err := _FastTransferGateway.contract.Call(opts, &out, "quoteInitiateSettlement", sourceDomain, repaymentAddress, orderIDs)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// QuoteInitiateSettlement is a free data retrieval call binding the contract method 0xe88787c2.
//
// Solidity: function quoteInitiateSettlement(uint32 sourceDomain, bytes32 repaymentAddress, bytes orderIDs) view returns(uint256)
func (_FastTransferGateway *FastTransferGatewaySession) QuoteInitiateSettlement(sourceDomain uint32, repaymentAddress [32]byte, orderIDs []byte) (*big.Int, error) {
	return _FastTransferGateway.Contract.QuoteInitiateSettlement(&_FastTransferGateway.CallOpts, sourceDomain, repaymentAddress, orderIDs)
}

// QuoteInitiateSettlement is a free data retrieval call binding the contract method 0xe88787c2.
//
// Solidity: function quoteInitiateSettlement(uint32 sourceDomain, bytes32 repaymentAddress, bytes orderIDs) view returns(uint256)
func (_FastTransferGateway *FastTransferGatewayCallerSession) QuoteInitiateSettlement(sourceDomain uint32, repaymentAddress [32]byte, orderIDs []byte) (*big.Int, error) {
	return _FastTransferGateway.Contract.QuoteInitiateSettlement(&_FastTransferGateway.CallOpts, sourceDomain, repaymentAddress, orderIDs)
}

// QuoteInitiateTimeout is a free data retrieval call binding the contract method 0x4eae2607.
//
// Solidity: function quoteInitiateTimeout(uint32 sourceDomain, (bytes32,bytes32,uint256,uint256,uint32,uint32,uint32,uint64,bytes)[] orders) view returns(uint256)
func (_FastTransferGateway *FastTransferGatewayCaller) QuoteInitiateTimeout(opts *bind.CallOpts, sourceDomain uint32, orders []FastTransferOrder) (*big.Int, error) {
	var out []interface{}
	err := _FastTransferGateway.contract.Call(opts, &out, "quoteInitiateTimeout", sourceDomain, orders)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// QuoteInitiateTimeout is a free data retrieval call binding the contract method 0x4eae2607.
//
// Solidity: function quoteInitiateTimeout(uint32 sourceDomain, (bytes32,bytes32,uint256,uint256,uint32,uint32,uint32,uint64,bytes)[] orders) view returns(uint256)
func (_FastTransferGateway *FastTransferGatewaySession) QuoteInitiateTimeout(sourceDomain uint32, orders []FastTransferOrder) (*big.Int, error) {
	return _FastTransferGateway.Contract.QuoteInitiateTimeout(&_FastTransferGateway.CallOpts, sourceDomain, orders)
}

// QuoteInitiateTimeout is a free data retrieval call binding the contract method 0x4eae2607.
//
// Solidity: function quoteInitiateTimeout(uint32 sourceDomain, (bytes32,bytes32,uint256,uint256,uint32,uint32,uint32,uint64,bytes)[] orders) view returns(uint256)
func (_FastTransferGateway *FastTransferGatewayCallerSession) QuoteInitiateTimeout(sourceDomain uint32, orders []FastTransferOrder) (*big.Int, error) {
	return _FastTransferGateway.Contract.QuoteInitiateTimeout(&_FastTransferGateway.CallOpts, sourceDomain, orders)
}

// RemoteDomains is a free data retrieval call binding the contract method 0x1ea9e2e3.
//
// Solidity: function remoteDomains(uint32 ) view returns(bytes32)
func (_FastTransferGateway *FastTransferGatewayCaller) RemoteDomains(opts *bind.CallOpts, arg0 uint32) ([32]byte, error) {
	var out []interface{}
	err := _FastTransferGateway.contract.Call(opts, &out, "remoteDomains", arg0)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// RemoteDomains is a free data retrieval call binding the contract method 0x1ea9e2e3.
//
// Solidity: function remoteDomains(uint32 ) view returns(bytes32)
func (_FastTransferGateway *FastTransferGatewaySession) RemoteDomains(arg0 uint32) ([32]byte, error) {
	return _FastTransferGateway.Contract.RemoteDomains(&_FastTransferGateway.CallOpts, arg0)
}

// RemoteDomains is a free data retrieval call binding the contract method 0x1ea9e2e3.
//
// Solidity: function remoteDomains(uint32 ) view returns(bytes32)
func (_FastTransferGateway *FastTransferGatewayCallerSession) RemoteDomains(arg0 uint32) ([32]byte, error) {
	return _FastTransferGateway.Contract.RemoteDomains(&_FastTransferGateway.CallOpts, arg0)
}

// SettlementDetails is a free data retrieval call binding the contract method 0x85cf3f93.
//
// Solidity: function settlementDetails(bytes32 ) view returns(bytes32 sender, uint256 nonce, uint32 destinationDomain, uint256 amount)
func (_FastTransferGateway *FastTransferGatewayCaller) SettlementDetails(opts *bind.CallOpts, arg0 [32]byte) (struct {
	Sender            [32]byte
	Nonce             *big.Int
	DestinationDomain uint32
	Amount            *big.Int
}, error) {
	var out []interface{}
	err := _FastTransferGateway.contract.Call(opts, &out, "settlementDetails", arg0)

	outstruct := new(struct {
		Sender            [32]byte
		Nonce             *big.Int
		DestinationDomain uint32
		Amount            *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Sender = *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)
	outstruct.Nonce = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.DestinationDomain = *abi.ConvertType(out[2], new(uint32)).(*uint32)
	outstruct.Amount = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// SettlementDetails is a free data retrieval call binding the contract method 0x85cf3f93.
//
// Solidity: function settlementDetails(bytes32 ) view returns(bytes32 sender, uint256 nonce, uint32 destinationDomain, uint256 amount)
func (_FastTransferGateway *FastTransferGatewaySession) SettlementDetails(arg0 [32]byte) (struct {
	Sender            [32]byte
	Nonce             *big.Int
	DestinationDomain uint32
	Amount            *big.Int
}, error) {
	return _FastTransferGateway.Contract.SettlementDetails(&_FastTransferGateway.CallOpts, arg0)
}

// SettlementDetails is a free data retrieval call binding the contract method 0x85cf3f93.
//
// Solidity: function settlementDetails(bytes32 ) view returns(bytes32 sender, uint256 nonce, uint32 destinationDomain, uint256 amount)
func (_FastTransferGateway *FastTransferGatewayCallerSession) SettlementDetails(arg0 [32]byte) (struct {
	Sender            [32]byte
	Nonce             *big.Int
	DestinationDomain uint32
	Amount            *big.Int
}, error) {
	return _FastTransferGateway.Contract.SettlementDetails(&_FastTransferGateway.CallOpts, arg0)
}

// Token is a free data retrieval call binding the contract method 0xfc0c546a.
//
// Solidity: function token() view returns(address)
func (_FastTransferGateway *FastTransferGatewayCaller) Token(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _FastTransferGateway.contract.Call(opts, &out, "token")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Token is a free data retrieval call binding the contract method 0xfc0c546a.
//
// Solidity: function token() view returns(address)
func (_FastTransferGateway *FastTransferGatewaySession) Token() (common.Address, error) {
	return _FastTransferGateway.Contract.Token(&_FastTransferGateway.CallOpts)
}

// Token is a free data retrieval call binding the contract method 0xfc0c546a.
//
// Solidity: function token() view returns(address)
func (_FastTransferGateway *FastTransferGatewayCallerSession) Token() (common.Address, error) {
	return _FastTransferGateway.Contract.Token(&_FastTransferGateway.CallOpts)
}

// FillOrder is a paid mutator transaction binding the contract method 0xb549117c.
//
// Solidity: function fillOrder(address filler, (bytes32,bytes32,uint256,uint256,uint32,uint32,uint32,uint64,bytes) order) returns()
func (_FastTransferGateway *FastTransferGatewayTransactor) FillOrder(opts *bind.TransactOpts, filler common.Address, order FastTransferOrder) (*types.Transaction, error) {
	return _FastTransferGateway.contract.Transact(opts, "fillOrder", filler, order)
}

// FillOrder is a paid mutator transaction binding the contract method 0xb549117c.
//
// Solidity: function fillOrder(address filler, (bytes32,bytes32,uint256,uint256,uint32,uint32,uint32,uint64,bytes) order) returns()
func (_FastTransferGateway *FastTransferGatewaySession) FillOrder(filler common.Address, order FastTransferOrder) (*types.Transaction, error) {
	return _FastTransferGateway.Contract.FillOrder(&_FastTransferGateway.TransactOpts, filler, order)
}

// FillOrder is a paid mutator transaction binding the contract method 0xb549117c.
//
// Solidity: function fillOrder(address filler, (bytes32,bytes32,uint256,uint256,uint32,uint32,uint32,uint64,bytes) order) returns()
func (_FastTransferGateway *FastTransferGatewayTransactorSession) FillOrder(filler common.Address, order FastTransferOrder) (*types.Transaction, error) {
	return _FastTransferGateway.Contract.FillOrder(&_FastTransferGateway.TransactOpts, filler, order)
}

// Handle is a paid mutator transaction binding the contract method 0x56d5d475.
//
// Solidity: function handle(uint32 _origin, bytes32 _sender, bytes _message) payable returns()
func (_FastTransferGateway *FastTransferGatewayTransactor) Handle(opts *bind.TransactOpts, _origin uint32, _sender [32]byte, _message []byte) (*types.Transaction, error) {
	return _FastTransferGateway.contract.Transact(opts, "handle", _origin, _sender, _message)
}

// Handle is a paid mutator transaction binding the contract method 0x56d5d475.
//
// Solidity: function handle(uint32 _origin, bytes32 _sender, bytes _message) payable returns()
func (_FastTransferGateway *FastTransferGatewaySession) Handle(_origin uint32, _sender [32]byte, _message []byte) (*types.Transaction, error) {
	return _FastTransferGateway.Contract.Handle(&_FastTransferGateway.TransactOpts, _origin, _sender, _message)
}

// Handle is a paid mutator transaction binding the contract method 0x56d5d475.
//
// Solidity: function handle(uint32 _origin, bytes32 _sender, bytes _message) payable returns()
func (_FastTransferGateway *FastTransferGatewayTransactorSession) Handle(_origin uint32, _sender [32]byte, _message []byte) (*types.Transaction, error) {
	return _FastTransferGateway.Contract.Handle(&_FastTransferGateway.TransactOpts, _origin, _sender, _message)
}

// Initialize is a paid mutator transaction binding the contract method 0x0b5e0b68.
//
// Solidity: function initialize(uint32 _localDomain, address _owner, address _token, address _mailbox, address _interchainSecurityModule, address _permit2, address _goFastCaller) returns()
func (_FastTransferGateway *FastTransferGatewayTransactor) Initialize(opts *bind.TransactOpts, _localDomain uint32, _owner common.Address, _token common.Address, _mailbox common.Address, _interchainSecurityModule common.Address, _permit2 common.Address, _goFastCaller common.Address) (*types.Transaction, error) {
	return _FastTransferGateway.contract.Transact(opts, "initialize", _localDomain, _owner, _token, _mailbox, _interchainSecurityModule, _permit2, _goFastCaller)
}

// Initialize is a paid mutator transaction binding the contract method 0x0b5e0b68.
//
// Solidity: function initialize(uint32 _localDomain, address _owner, address _token, address _mailbox, address _interchainSecurityModule, address _permit2, address _goFastCaller) returns()
func (_FastTransferGateway *FastTransferGatewaySession) Initialize(_localDomain uint32, _owner common.Address, _token common.Address, _mailbox common.Address, _interchainSecurityModule common.Address, _permit2 common.Address, _goFastCaller common.Address) (*types.Transaction, error) {
	return _FastTransferGateway.Contract.Initialize(&_FastTransferGateway.TransactOpts, _localDomain, _owner, _token, _mailbox, _interchainSecurityModule, _permit2, _goFastCaller)
}

// Initialize is a paid mutator transaction binding the contract method 0x0b5e0b68.
//
// Solidity: function initialize(uint32 _localDomain, address _owner, address _token, address _mailbox, address _interchainSecurityModule, address _permit2, address _goFastCaller) returns()
func (_FastTransferGateway *FastTransferGatewayTransactorSession) Initialize(_localDomain uint32, _owner common.Address, _token common.Address, _mailbox common.Address, _interchainSecurityModule common.Address, _permit2 common.Address, _goFastCaller common.Address) (*types.Transaction, error) {
	return _FastTransferGateway.Contract.Initialize(&_FastTransferGateway.TransactOpts, _localDomain, _owner, _token, _mailbox, _interchainSecurityModule, _permit2, _goFastCaller)
}

// InitiateSettlement is a paid mutator transaction binding the contract method 0x30c5b926.
//
// Solidity: function initiateSettlement(bytes32 repaymentAddress, bytes orderIDs) payable returns()
func (_FastTransferGateway *FastTransferGatewayTransactor) InitiateSettlement(opts *bind.TransactOpts, repaymentAddress [32]byte, orderIDs []byte) (*types.Transaction, error) {
	return _FastTransferGateway.contract.Transact(opts, "initiateSettlement", repaymentAddress, orderIDs)
}

// InitiateSettlement is a paid mutator transaction binding the contract method 0x30c5b926.
//
// Solidity: function initiateSettlement(bytes32 repaymentAddress, bytes orderIDs) payable returns()
func (_FastTransferGateway *FastTransferGatewaySession) InitiateSettlement(repaymentAddress [32]byte, orderIDs []byte) (*types.Transaction, error) {
	return _FastTransferGateway.Contract.InitiateSettlement(&_FastTransferGateway.TransactOpts, repaymentAddress, orderIDs)
}

// InitiateSettlement is a paid mutator transaction binding the contract method 0x30c5b926.
//
// Solidity: function initiateSettlement(bytes32 repaymentAddress, bytes orderIDs) payable returns()
func (_FastTransferGateway *FastTransferGatewayTransactorSession) InitiateSettlement(repaymentAddress [32]byte, orderIDs []byte) (*types.Transaction, error) {
	return _FastTransferGateway.Contract.InitiateSettlement(&_FastTransferGateway.TransactOpts, repaymentAddress, orderIDs)
}

// InitiateTimeout is a paid mutator transaction binding the contract method 0x88efb875.
//
// Solidity: function initiateTimeout((bytes32,bytes32,uint256,uint256,uint32,uint32,uint32,uint64,bytes)[] orders) payable returns()
func (_FastTransferGateway *FastTransferGatewayTransactor) InitiateTimeout(opts *bind.TransactOpts, orders []FastTransferOrder) (*types.Transaction, error) {
	return _FastTransferGateway.contract.Transact(opts, "initiateTimeout", orders)
}

// InitiateTimeout is a paid mutator transaction binding the contract method 0x88efb875.
//
// Solidity: function initiateTimeout((bytes32,bytes32,uint256,uint256,uint32,uint32,uint32,uint64,bytes)[] orders) payable returns()
func (_FastTransferGateway *FastTransferGatewaySession) InitiateTimeout(orders []FastTransferOrder) (*types.Transaction, error) {
	return _FastTransferGateway.Contract.InitiateTimeout(&_FastTransferGateway.TransactOpts, orders)
}

// InitiateTimeout is a paid mutator transaction binding the contract method 0x88efb875.
//
// Solidity: function initiateTimeout((bytes32,bytes32,uint256,uint256,uint32,uint32,uint32,uint64,bytes)[] orders) payable returns()
func (_FastTransferGateway *FastTransferGatewayTransactorSession) InitiateTimeout(orders []FastTransferOrder) (*types.Transaction, error) {
	return _FastTransferGateway.Contract.InitiateTimeout(&_FastTransferGateway.TransactOpts, orders)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_FastTransferGateway *FastTransferGatewayTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _FastTransferGateway.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_FastTransferGateway *FastTransferGatewaySession) RenounceOwnership() (*types.Transaction, error) {
	return _FastTransferGateway.Contract.RenounceOwnership(&_FastTransferGateway.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_FastTransferGateway *FastTransferGatewayTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _FastTransferGateway.Contract.RenounceOwnership(&_FastTransferGateway.TransactOpts)
}

// SetInterchainSecurityModule is a paid mutator transaction binding the contract method 0x0e72cc06.
//
// Solidity: function setInterchainSecurityModule(address _interchainSecurityModule) returns()
func (_FastTransferGateway *FastTransferGatewayTransactor) SetInterchainSecurityModule(opts *bind.TransactOpts, _interchainSecurityModule common.Address) (*types.Transaction, error) {
	return _FastTransferGateway.contract.Transact(opts, "setInterchainSecurityModule", _interchainSecurityModule)
}

// SetInterchainSecurityModule is a paid mutator transaction binding the contract method 0x0e72cc06.
//
// Solidity: function setInterchainSecurityModule(address _interchainSecurityModule) returns()
func (_FastTransferGateway *FastTransferGatewaySession) SetInterchainSecurityModule(_interchainSecurityModule common.Address) (*types.Transaction, error) {
	return _FastTransferGateway.Contract.SetInterchainSecurityModule(&_FastTransferGateway.TransactOpts, _interchainSecurityModule)
}

// SetInterchainSecurityModule is a paid mutator transaction binding the contract method 0x0e72cc06.
//
// Solidity: function setInterchainSecurityModule(address _interchainSecurityModule) returns()
func (_FastTransferGateway *FastTransferGatewayTransactorSession) SetInterchainSecurityModule(_interchainSecurityModule common.Address) (*types.Transaction, error) {
	return _FastTransferGateway.Contract.SetInterchainSecurityModule(&_FastTransferGateway.TransactOpts, _interchainSecurityModule)
}

// SetMailbox is a paid mutator transaction binding the contract method 0xf3c61d6b.
//
// Solidity: function setMailbox(address _mailbox) returns()
func (_FastTransferGateway *FastTransferGatewayTransactor) SetMailbox(opts *bind.TransactOpts, _mailbox common.Address) (*types.Transaction, error) {
	return _FastTransferGateway.contract.Transact(opts, "setMailbox", _mailbox)
}

// SetMailbox is a paid mutator transaction binding the contract method 0xf3c61d6b.
//
// Solidity: function setMailbox(address _mailbox) returns()
func (_FastTransferGateway *FastTransferGatewaySession) SetMailbox(_mailbox common.Address) (*types.Transaction, error) {
	return _FastTransferGateway.Contract.SetMailbox(&_FastTransferGateway.TransactOpts, _mailbox)
}

// SetMailbox is a paid mutator transaction binding the contract method 0xf3c61d6b.
//
// Solidity: function setMailbox(address _mailbox) returns()
func (_FastTransferGateway *FastTransferGatewayTransactorSession) SetMailbox(_mailbox common.Address) (*types.Transaction, error) {
	return _FastTransferGateway.Contract.SetMailbox(&_FastTransferGateway.TransactOpts, _mailbox)
}

// SetRemoteDomain is a paid mutator transaction binding the contract method 0xe5dc8496.
//
// Solidity: function setRemoteDomain(uint32 domain, bytes32 remoteContract) returns()
func (_FastTransferGateway *FastTransferGatewayTransactor) SetRemoteDomain(opts *bind.TransactOpts, domain uint32, remoteContract [32]byte) (*types.Transaction, error) {
	return _FastTransferGateway.contract.Transact(opts, "setRemoteDomain", domain, remoteContract)
}

// SetRemoteDomain is a paid mutator transaction binding the contract method 0xe5dc8496.
//
// Solidity: function setRemoteDomain(uint32 domain, bytes32 remoteContract) returns()
func (_FastTransferGateway *FastTransferGatewaySession) SetRemoteDomain(domain uint32, remoteContract [32]byte) (*types.Transaction, error) {
	return _FastTransferGateway.Contract.SetRemoteDomain(&_FastTransferGateway.TransactOpts, domain, remoteContract)
}

// SetRemoteDomain is a paid mutator transaction binding the contract method 0xe5dc8496.
//
// Solidity: function setRemoteDomain(uint32 domain, bytes32 remoteContract) returns()
func (_FastTransferGateway *FastTransferGatewayTransactorSession) SetRemoteDomain(domain uint32, remoteContract [32]byte) (*types.Transaction, error) {
	return _FastTransferGateway.Contract.SetRemoteDomain(&_FastTransferGateway.TransactOpts, domain, remoteContract)
}

// SubmitOrder is a paid mutator transaction binding the contract method 0x6ad1b6ac.
//
// Solidity: function submitOrder(bytes32 sender, bytes32 recipient, uint256 amountIn, uint256 amountOut, uint32 destinationDomain, uint64 timeoutTimestamp, bytes data) returns(bytes32)
func (_FastTransferGateway *FastTransferGatewayTransactor) SubmitOrder(opts *bind.TransactOpts, sender [32]byte, recipient [32]byte, amountIn *big.Int, amountOut *big.Int, destinationDomain uint32, timeoutTimestamp uint64, data []byte) (*types.Transaction, error) {
	return _FastTransferGateway.contract.Transact(opts, "submitOrder", sender, recipient, amountIn, amountOut, destinationDomain, timeoutTimestamp, data)
}

// SubmitOrder is a paid mutator transaction binding the contract method 0x6ad1b6ac.
//
// Solidity: function submitOrder(bytes32 sender, bytes32 recipient, uint256 amountIn, uint256 amountOut, uint32 destinationDomain, uint64 timeoutTimestamp, bytes data) returns(bytes32)
func (_FastTransferGateway *FastTransferGatewaySession) SubmitOrder(sender [32]byte, recipient [32]byte, amountIn *big.Int, amountOut *big.Int, destinationDomain uint32, timeoutTimestamp uint64, data []byte) (*types.Transaction, error) {
	return _FastTransferGateway.Contract.SubmitOrder(&_FastTransferGateway.TransactOpts, sender, recipient, amountIn, amountOut, destinationDomain, timeoutTimestamp, data)
}

// SubmitOrder is a paid mutator transaction binding the contract method 0x6ad1b6ac.
//
// Solidity: function submitOrder(bytes32 sender, bytes32 recipient, uint256 amountIn, uint256 amountOut, uint32 destinationDomain, uint64 timeoutTimestamp, bytes data) returns(bytes32)
func (_FastTransferGateway *FastTransferGatewayTransactorSession) SubmitOrder(sender [32]byte, recipient [32]byte, amountIn *big.Int, amountOut *big.Int, destinationDomain uint32, timeoutTimestamp uint64, data []byte) (*types.Transaction, error) {
	return _FastTransferGateway.Contract.SubmitOrder(&_FastTransferGateway.TransactOpts, sender, recipient, amountIn, amountOut, destinationDomain, timeoutTimestamp, data)
}

// SubmitOrderWithPermit is a paid mutator transaction binding the contract method 0x061b32c9.
//
// Solidity: function submitOrderWithPermit(bytes32 sender, bytes32 recipient, uint256 amountIn, uint256 amountOut, uint32 destinationDomain, uint64 timeoutTimestamp, uint256 permitDeadline, bytes data, bytes signature) returns(bytes32)
func (_FastTransferGateway *FastTransferGatewayTransactor) SubmitOrderWithPermit(opts *bind.TransactOpts, sender [32]byte, recipient [32]byte, amountIn *big.Int, amountOut *big.Int, destinationDomain uint32, timeoutTimestamp uint64, permitDeadline *big.Int, data []byte, signature []byte) (*types.Transaction, error) {
	return _FastTransferGateway.contract.Transact(opts, "submitOrderWithPermit", sender, recipient, amountIn, amountOut, destinationDomain, timeoutTimestamp, permitDeadline, data, signature)
}

// SubmitOrderWithPermit is a paid mutator transaction binding the contract method 0x061b32c9.
//
// Solidity: function submitOrderWithPermit(bytes32 sender, bytes32 recipient, uint256 amountIn, uint256 amountOut, uint32 destinationDomain, uint64 timeoutTimestamp, uint256 permitDeadline, bytes data, bytes signature) returns(bytes32)
func (_FastTransferGateway *FastTransferGatewaySession) SubmitOrderWithPermit(sender [32]byte, recipient [32]byte, amountIn *big.Int, amountOut *big.Int, destinationDomain uint32, timeoutTimestamp uint64, permitDeadline *big.Int, data []byte, signature []byte) (*types.Transaction, error) {
	return _FastTransferGateway.Contract.SubmitOrderWithPermit(&_FastTransferGateway.TransactOpts, sender, recipient, amountIn, amountOut, destinationDomain, timeoutTimestamp, permitDeadline, data, signature)
}

// SubmitOrderWithPermit is a paid mutator transaction binding the contract method 0x061b32c9.
//
// Solidity: function submitOrderWithPermit(bytes32 sender, bytes32 recipient, uint256 amountIn, uint256 amountOut, uint32 destinationDomain, uint64 timeoutTimestamp, uint256 permitDeadline, bytes data, bytes signature) returns(bytes32)
func (_FastTransferGateway *FastTransferGatewayTransactorSession) SubmitOrderWithPermit(sender [32]byte, recipient [32]byte, amountIn *big.Int, amountOut *big.Int, destinationDomain uint32, timeoutTimestamp uint64, permitDeadline *big.Int, data []byte, signature []byte) (*types.Transaction, error) {
	return _FastTransferGateway.Contract.SubmitOrderWithPermit(&_FastTransferGateway.TransactOpts, sender, recipient, amountIn, amountOut, destinationDomain, timeoutTimestamp, permitDeadline, data, signature)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_FastTransferGateway *FastTransferGatewayTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _FastTransferGateway.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_FastTransferGateway *FastTransferGatewaySession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _FastTransferGateway.Contract.TransferOwnership(&_FastTransferGateway.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_FastTransferGateway *FastTransferGatewayTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _FastTransferGateway.Contract.TransferOwnership(&_FastTransferGateway.TransactOpts, newOwner)
}

// UpgradeToAndCall is a paid mutator transaction binding the contract method 0x4f1ef286.
//
// Solidity: function upgradeToAndCall(address newImplementation, bytes data) payable returns()
func (_FastTransferGateway *FastTransferGatewayTransactor) UpgradeToAndCall(opts *bind.TransactOpts, newImplementation common.Address, data []byte) (*types.Transaction, error) {
	return _FastTransferGateway.contract.Transact(opts, "upgradeToAndCall", newImplementation, data)
}

// UpgradeToAndCall is a paid mutator transaction binding the contract method 0x4f1ef286.
//
// Solidity: function upgradeToAndCall(address newImplementation, bytes data) payable returns()
func (_FastTransferGateway *FastTransferGatewaySession) UpgradeToAndCall(newImplementation common.Address, data []byte) (*types.Transaction, error) {
	return _FastTransferGateway.Contract.UpgradeToAndCall(&_FastTransferGateway.TransactOpts, newImplementation, data)
}

// UpgradeToAndCall is a paid mutator transaction binding the contract method 0x4f1ef286.
//
// Solidity: function upgradeToAndCall(address newImplementation, bytes data) payable returns()
func (_FastTransferGateway *FastTransferGatewayTransactorSession) UpgradeToAndCall(newImplementation common.Address, data []byte) (*types.Transaction, error) {
	return _FastTransferGateway.Contract.UpgradeToAndCall(&_FastTransferGateway.TransactOpts, newImplementation, data)
}

// FastTransferGatewayInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the FastTransferGateway contract.
type FastTransferGatewayInitializedIterator struct {
	Event *FastTransferGatewayInitialized // Event containing the contract specifics and raw log

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
func (it *FastTransferGatewayInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FastTransferGatewayInitialized)
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
		it.Event = new(FastTransferGatewayInitialized)
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
func (it *FastTransferGatewayInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FastTransferGatewayInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FastTransferGatewayInitialized represents a Initialized event raised by the FastTransferGateway contract.
type FastTransferGatewayInitialized struct {
	Version uint64
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_FastTransferGateway *FastTransferGatewayFilterer) FilterInitialized(opts *bind.FilterOpts) (*FastTransferGatewayInitializedIterator, error) {

	logs, sub, err := _FastTransferGateway.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &FastTransferGatewayInitializedIterator{contract: _FastTransferGateway.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_FastTransferGateway *FastTransferGatewayFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *FastTransferGatewayInitialized) (event.Subscription, error) {

	logs, sub, err := _FastTransferGateway.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FastTransferGatewayInitialized)
				if err := _FastTransferGateway.contract.UnpackLog(event, "Initialized", log); err != nil {
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
func (_FastTransferGateway *FastTransferGatewayFilterer) ParseInitialized(log types.Log) (*FastTransferGatewayInitialized, error) {
	event := new(FastTransferGatewayInitialized)
	if err := _FastTransferGateway.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FastTransferGatewayOrderAlreadySettledIterator is returned from FilterOrderAlreadySettled and is used to iterate over the raw logs and unpacked data for OrderAlreadySettled events raised by the FastTransferGateway contract.
type FastTransferGatewayOrderAlreadySettledIterator struct {
	Event *FastTransferGatewayOrderAlreadySettled // Event containing the contract specifics and raw log

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
func (it *FastTransferGatewayOrderAlreadySettledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FastTransferGatewayOrderAlreadySettled)
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
		it.Event = new(FastTransferGatewayOrderAlreadySettled)
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
func (it *FastTransferGatewayOrderAlreadySettledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FastTransferGatewayOrderAlreadySettledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FastTransferGatewayOrderAlreadySettled represents a OrderAlreadySettled event raised by the FastTransferGateway contract.
type FastTransferGatewayOrderAlreadySettled struct {
	OrderID [32]byte
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterOrderAlreadySettled is a free log retrieval operation binding the contract event 0x0349d9fa752b33cd4d30f97058afcf8e7b9d5c3c7a20056699a8947fedf73138.
//
// Solidity: event OrderAlreadySettled(bytes32 indexed orderID)
func (_FastTransferGateway *FastTransferGatewayFilterer) FilterOrderAlreadySettled(opts *bind.FilterOpts, orderID [][32]byte) (*FastTransferGatewayOrderAlreadySettledIterator, error) {

	var orderIDRule []interface{}
	for _, orderIDItem := range orderID {
		orderIDRule = append(orderIDRule, orderIDItem)
	}

	logs, sub, err := _FastTransferGateway.contract.FilterLogs(opts, "OrderAlreadySettled", orderIDRule)
	if err != nil {
		return nil, err
	}
	return &FastTransferGatewayOrderAlreadySettledIterator{contract: _FastTransferGateway.contract, event: "OrderAlreadySettled", logs: logs, sub: sub}, nil
}

// WatchOrderAlreadySettled is a free log subscription operation binding the contract event 0x0349d9fa752b33cd4d30f97058afcf8e7b9d5c3c7a20056699a8947fedf73138.
//
// Solidity: event OrderAlreadySettled(bytes32 indexed orderID)
func (_FastTransferGateway *FastTransferGatewayFilterer) WatchOrderAlreadySettled(opts *bind.WatchOpts, sink chan<- *FastTransferGatewayOrderAlreadySettled, orderID [][32]byte) (event.Subscription, error) {

	var orderIDRule []interface{}
	for _, orderIDItem := range orderID {
		orderIDRule = append(orderIDRule, orderIDItem)
	}

	logs, sub, err := _FastTransferGateway.contract.WatchLogs(opts, "OrderAlreadySettled", orderIDRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FastTransferGatewayOrderAlreadySettled)
				if err := _FastTransferGateway.contract.UnpackLog(event, "OrderAlreadySettled", log); err != nil {
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

// ParseOrderAlreadySettled is a log parse operation binding the contract event 0x0349d9fa752b33cd4d30f97058afcf8e7b9d5c3c7a20056699a8947fedf73138.
//
// Solidity: event OrderAlreadySettled(bytes32 indexed orderID)
func (_FastTransferGateway *FastTransferGatewayFilterer) ParseOrderAlreadySettled(log types.Log) (*FastTransferGatewayOrderAlreadySettled, error) {
	event := new(FastTransferGatewayOrderAlreadySettled)
	if err := _FastTransferGateway.contract.UnpackLog(event, "OrderAlreadySettled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FastTransferGatewayOrderRefundedIterator is returned from FilterOrderRefunded and is used to iterate over the raw logs and unpacked data for OrderRefunded events raised by the FastTransferGateway contract.
type FastTransferGatewayOrderRefundedIterator struct {
	Event *FastTransferGatewayOrderRefunded // Event containing the contract specifics and raw log

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
func (it *FastTransferGatewayOrderRefundedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FastTransferGatewayOrderRefunded)
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
		it.Event = new(FastTransferGatewayOrderRefunded)
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
func (it *FastTransferGatewayOrderRefundedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FastTransferGatewayOrderRefundedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FastTransferGatewayOrderRefunded represents a OrderRefunded event raised by the FastTransferGateway contract.
type FastTransferGatewayOrderRefunded struct {
	OrderID [32]byte
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterOrderRefunded is a free log retrieval operation binding the contract event 0xa60671d8537ed193e567f86ddf28cf35dc67073b5ad80a2d41359cfa78db0a1e.
//
// Solidity: event OrderRefunded(bytes32 indexed orderID)
func (_FastTransferGateway *FastTransferGatewayFilterer) FilterOrderRefunded(opts *bind.FilterOpts, orderID [][32]byte) (*FastTransferGatewayOrderRefundedIterator, error) {

	var orderIDRule []interface{}
	for _, orderIDItem := range orderID {
		orderIDRule = append(orderIDRule, orderIDItem)
	}

	logs, sub, err := _FastTransferGateway.contract.FilterLogs(opts, "OrderRefunded", orderIDRule)
	if err != nil {
		return nil, err
	}
	return &FastTransferGatewayOrderRefundedIterator{contract: _FastTransferGateway.contract, event: "OrderRefunded", logs: logs, sub: sub}, nil
}

// WatchOrderRefunded is a free log subscription operation binding the contract event 0xa60671d8537ed193e567f86ddf28cf35dc67073b5ad80a2d41359cfa78db0a1e.
//
// Solidity: event OrderRefunded(bytes32 indexed orderID)
func (_FastTransferGateway *FastTransferGatewayFilterer) WatchOrderRefunded(opts *bind.WatchOpts, sink chan<- *FastTransferGatewayOrderRefunded, orderID [][32]byte) (event.Subscription, error) {

	var orderIDRule []interface{}
	for _, orderIDItem := range orderID {
		orderIDRule = append(orderIDRule, orderIDItem)
	}

	logs, sub, err := _FastTransferGateway.contract.WatchLogs(opts, "OrderRefunded", orderIDRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FastTransferGatewayOrderRefunded)
				if err := _FastTransferGateway.contract.UnpackLog(event, "OrderRefunded", log); err != nil {
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

// ParseOrderRefunded is a log parse operation binding the contract event 0xa60671d8537ed193e567f86ddf28cf35dc67073b5ad80a2d41359cfa78db0a1e.
//
// Solidity: event OrderRefunded(bytes32 indexed orderID)
func (_FastTransferGateway *FastTransferGatewayFilterer) ParseOrderRefunded(log types.Log) (*FastTransferGatewayOrderRefunded, error) {
	event := new(FastTransferGatewayOrderRefunded)
	if err := _FastTransferGateway.contract.UnpackLog(event, "OrderRefunded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FastTransferGatewayOrderSettledIterator is returned from FilterOrderSettled and is used to iterate over the raw logs and unpacked data for OrderSettled events raised by the FastTransferGateway contract.
type FastTransferGatewayOrderSettledIterator struct {
	Event *FastTransferGatewayOrderSettled // Event containing the contract specifics and raw log

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
func (it *FastTransferGatewayOrderSettledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FastTransferGatewayOrderSettled)
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
		it.Event = new(FastTransferGatewayOrderSettled)
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
func (it *FastTransferGatewayOrderSettledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FastTransferGatewayOrderSettledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FastTransferGatewayOrderSettled represents a OrderSettled event raised by the FastTransferGateway contract.
type FastTransferGatewayOrderSettled struct {
	OrderID [32]byte
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterOrderSettled is a free log retrieval operation binding the contract event 0xd4250d6114a611e75d68b1c6f14c61e967863d8ac20bc8ebfa4e5f28f6647366.
//
// Solidity: event OrderSettled(bytes32 indexed orderID)
func (_FastTransferGateway *FastTransferGatewayFilterer) FilterOrderSettled(opts *bind.FilterOpts, orderID [][32]byte) (*FastTransferGatewayOrderSettledIterator, error) {

	var orderIDRule []interface{}
	for _, orderIDItem := range orderID {
		orderIDRule = append(orderIDRule, orderIDItem)
	}

	logs, sub, err := _FastTransferGateway.contract.FilterLogs(opts, "OrderSettled", orderIDRule)
	if err != nil {
		return nil, err
	}
	return &FastTransferGatewayOrderSettledIterator{contract: _FastTransferGateway.contract, event: "OrderSettled", logs: logs, sub: sub}, nil
}

// WatchOrderSettled is a free log subscription operation binding the contract event 0xd4250d6114a611e75d68b1c6f14c61e967863d8ac20bc8ebfa4e5f28f6647366.
//
// Solidity: event OrderSettled(bytes32 indexed orderID)
func (_FastTransferGateway *FastTransferGatewayFilterer) WatchOrderSettled(opts *bind.WatchOpts, sink chan<- *FastTransferGatewayOrderSettled, orderID [][32]byte) (event.Subscription, error) {

	var orderIDRule []interface{}
	for _, orderIDItem := range orderID {
		orderIDRule = append(orderIDRule, orderIDItem)
	}

	logs, sub, err := _FastTransferGateway.contract.WatchLogs(opts, "OrderSettled", orderIDRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FastTransferGatewayOrderSettled)
				if err := _FastTransferGateway.contract.UnpackLog(event, "OrderSettled", log); err != nil {
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

// ParseOrderSettled is a log parse operation binding the contract event 0xd4250d6114a611e75d68b1c6f14c61e967863d8ac20bc8ebfa4e5f28f6647366.
//
// Solidity: event OrderSettled(bytes32 indexed orderID)
func (_FastTransferGateway *FastTransferGatewayFilterer) ParseOrderSettled(log types.Log) (*FastTransferGatewayOrderSettled, error) {
	event := new(FastTransferGatewayOrderSettled)
	if err := _FastTransferGateway.contract.UnpackLog(event, "OrderSettled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FastTransferGatewayOrderSubmittedIterator is returned from FilterOrderSubmitted and is used to iterate over the raw logs and unpacked data for OrderSubmitted events raised by the FastTransferGateway contract.
type FastTransferGatewayOrderSubmittedIterator struct {
	Event *FastTransferGatewayOrderSubmitted // Event containing the contract specifics and raw log

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
func (it *FastTransferGatewayOrderSubmittedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FastTransferGatewayOrderSubmitted)
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
		it.Event = new(FastTransferGatewayOrderSubmitted)
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
func (it *FastTransferGatewayOrderSubmittedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FastTransferGatewayOrderSubmittedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FastTransferGatewayOrderSubmitted represents a OrderSubmitted event raised by the FastTransferGateway contract.
type FastTransferGatewayOrderSubmitted struct {
	OrderID [32]byte
	Order   []byte
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterOrderSubmitted is a free log retrieval operation binding the contract event 0x59f858504f8d8ad967dd7453df850e265270474e364b7e2fbd3333e06efdbfc0.
//
// Solidity: event OrderSubmitted(bytes32 indexed orderID, bytes order)
func (_FastTransferGateway *FastTransferGatewayFilterer) FilterOrderSubmitted(opts *bind.FilterOpts, orderID [][32]byte) (*FastTransferGatewayOrderSubmittedIterator, error) {

	var orderIDRule []interface{}
	for _, orderIDItem := range orderID {
		orderIDRule = append(orderIDRule, orderIDItem)
	}

	logs, sub, err := _FastTransferGateway.contract.FilterLogs(opts, "OrderSubmitted", orderIDRule)
	if err != nil {
		return nil, err
	}
	return &FastTransferGatewayOrderSubmittedIterator{contract: _FastTransferGateway.contract, event: "OrderSubmitted", logs: logs, sub: sub}, nil
}

// WatchOrderSubmitted is a free log subscription operation binding the contract event 0x59f858504f8d8ad967dd7453df850e265270474e364b7e2fbd3333e06efdbfc0.
//
// Solidity: event OrderSubmitted(bytes32 indexed orderID, bytes order)
func (_FastTransferGateway *FastTransferGatewayFilterer) WatchOrderSubmitted(opts *bind.WatchOpts, sink chan<- *FastTransferGatewayOrderSubmitted, orderID [][32]byte) (event.Subscription, error) {

	var orderIDRule []interface{}
	for _, orderIDItem := range orderID {
		orderIDRule = append(orderIDRule, orderIDItem)
	}

	logs, sub, err := _FastTransferGateway.contract.WatchLogs(opts, "OrderSubmitted", orderIDRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FastTransferGatewayOrderSubmitted)
				if err := _FastTransferGateway.contract.UnpackLog(event, "OrderSubmitted", log); err != nil {
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

// ParseOrderSubmitted is a log parse operation binding the contract event 0x59f858504f8d8ad967dd7453df850e265270474e364b7e2fbd3333e06efdbfc0.
//
// Solidity: event OrderSubmitted(bytes32 indexed orderID, bytes order)
func (_FastTransferGateway *FastTransferGatewayFilterer) ParseOrderSubmitted(log types.Log) (*FastTransferGatewayOrderSubmitted, error) {
	event := new(FastTransferGatewayOrderSubmitted)
	if err := _FastTransferGateway.contract.UnpackLog(event, "OrderSubmitted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FastTransferGatewayOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the FastTransferGateway contract.
type FastTransferGatewayOwnershipTransferredIterator struct {
	Event *FastTransferGatewayOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *FastTransferGatewayOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FastTransferGatewayOwnershipTransferred)
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
		it.Event = new(FastTransferGatewayOwnershipTransferred)
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
func (it *FastTransferGatewayOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FastTransferGatewayOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FastTransferGatewayOwnershipTransferred represents a OwnershipTransferred event raised by the FastTransferGateway contract.
type FastTransferGatewayOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_FastTransferGateway *FastTransferGatewayFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*FastTransferGatewayOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _FastTransferGateway.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &FastTransferGatewayOwnershipTransferredIterator{contract: _FastTransferGateway.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_FastTransferGateway *FastTransferGatewayFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *FastTransferGatewayOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _FastTransferGateway.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FastTransferGatewayOwnershipTransferred)
				if err := _FastTransferGateway.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_FastTransferGateway *FastTransferGatewayFilterer) ParseOwnershipTransferred(log types.Log) (*FastTransferGatewayOwnershipTransferred, error) {
	event := new(FastTransferGatewayOwnershipTransferred)
	if err := _FastTransferGateway.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FastTransferGatewayUpgradedIterator is returned from FilterUpgraded and is used to iterate over the raw logs and unpacked data for Upgraded events raised by the FastTransferGateway contract.
type FastTransferGatewayUpgradedIterator struct {
	Event *FastTransferGatewayUpgraded // Event containing the contract specifics and raw log

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
func (it *FastTransferGatewayUpgradedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FastTransferGatewayUpgraded)
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
		it.Event = new(FastTransferGatewayUpgraded)
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
func (it *FastTransferGatewayUpgradedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FastTransferGatewayUpgradedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FastTransferGatewayUpgraded represents a Upgraded event raised by the FastTransferGateway contract.
type FastTransferGatewayUpgraded struct {
	Implementation common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterUpgraded is a free log retrieval operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_FastTransferGateway *FastTransferGatewayFilterer) FilterUpgraded(opts *bind.FilterOpts, implementation []common.Address) (*FastTransferGatewayUpgradedIterator, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _FastTransferGateway.contract.FilterLogs(opts, "Upgraded", implementationRule)
	if err != nil {
		return nil, err
	}
	return &FastTransferGatewayUpgradedIterator{contract: _FastTransferGateway.contract, event: "Upgraded", logs: logs, sub: sub}, nil
}

// WatchUpgraded is a free log subscription operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_FastTransferGateway *FastTransferGatewayFilterer) WatchUpgraded(opts *bind.WatchOpts, sink chan<- *FastTransferGatewayUpgraded, implementation []common.Address) (event.Subscription, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _FastTransferGateway.contract.WatchLogs(opts, "Upgraded", implementationRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FastTransferGatewayUpgraded)
				if err := _FastTransferGateway.contract.UnpackLog(event, "Upgraded", log); err != nil {
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

// ParseUpgraded is a log parse operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_FastTransferGateway *FastTransferGatewayFilterer) ParseUpgraded(log types.Log) (*FastTransferGatewayUpgraded, error) {
	event := new(FastTransferGatewayUpgraded)
	if err := _FastTransferGateway.contract.UnpackLog(event, "Upgraded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
