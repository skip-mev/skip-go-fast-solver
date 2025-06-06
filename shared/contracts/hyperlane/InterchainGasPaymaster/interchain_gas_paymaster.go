// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package interchain_gas_paymaster

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

// InterchainGasPaymasterMetaData contains all meta data concerning the InterchainGasPaymaster contract.
var InterchainGasPaymasterMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"messageId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"uint32\",\"name\":\"destinationDomain\",\"type\":\"uint32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"gasAmount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"payment\",\"type\":\"uint256\"}],\"name\":\"GasPayment\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_messageId\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"_destinationDomain\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"_gasAmount\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_refundAddress\",\"type\":\"address\"}],\"name\":\"payForGas\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"_destinationDomain\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"_gasAmount\",\"type\":\"uint256\"}],\"name\":\"quoteGasPayment\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// InterchainGasPaymasterABI is the input ABI used to generate the binding from.
// Deprecated: Use InterchainGasPaymasterMetaData.ABI instead.
var InterchainGasPaymasterABI = InterchainGasPaymasterMetaData.ABI

// InterchainGasPaymaster is an auto generated Go binding around an Ethereum contract.
type InterchainGasPaymaster struct {
	InterchainGasPaymasterCaller     // Read-only binding to the contract
	InterchainGasPaymasterTransactor // Write-only binding to the contract
	InterchainGasPaymasterFilterer   // Log filterer for contract events
}

// InterchainGasPaymasterCaller is an auto generated read-only Go binding around an Ethereum contract.
type InterchainGasPaymasterCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InterchainGasPaymasterTransactor is an auto generated write-only Go binding around an Ethereum contract.
type InterchainGasPaymasterTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InterchainGasPaymasterFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type InterchainGasPaymasterFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InterchainGasPaymasterSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type InterchainGasPaymasterSession struct {
	Contract     *InterchainGasPaymaster // Generic contract binding to set the session for
	CallOpts     bind.CallOpts           // Call options to use throughout this session
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// InterchainGasPaymasterCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type InterchainGasPaymasterCallerSession struct {
	Contract *InterchainGasPaymasterCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                 // Call options to use throughout this session
}

// InterchainGasPaymasterTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type InterchainGasPaymasterTransactorSession struct {
	Contract     *InterchainGasPaymasterTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                 // Transaction auth options to use throughout this session
}

// InterchainGasPaymasterRaw is an auto generated low-level Go binding around an Ethereum contract.
type InterchainGasPaymasterRaw struct {
	Contract *InterchainGasPaymaster // Generic contract binding to access the raw methods on
}

// InterchainGasPaymasterCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type InterchainGasPaymasterCallerRaw struct {
	Contract *InterchainGasPaymasterCaller // Generic read-only contract binding to access the raw methods on
}

// InterchainGasPaymasterTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type InterchainGasPaymasterTransactorRaw struct {
	Contract *InterchainGasPaymasterTransactor // Generic write-only contract binding to access the raw methods on
}

// NewInterchainGasPaymaster creates a new instance of InterchainGasPaymaster, bound to a specific deployed contract.
func NewInterchainGasPaymaster(address common.Address, backend bind.ContractBackend) (*InterchainGasPaymaster, error) {
	contract, err := bindInterchainGasPaymaster(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &InterchainGasPaymaster{InterchainGasPaymasterCaller: InterchainGasPaymasterCaller{contract: contract}, InterchainGasPaymasterTransactor: InterchainGasPaymasterTransactor{contract: contract}, InterchainGasPaymasterFilterer: InterchainGasPaymasterFilterer{contract: contract}}, nil
}

// NewInterchainGasPaymasterCaller creates a new read-only instance of InterchainGasPaymaster, bound to a specific deployed contract.
func NewInterchainGasPaymasterCaller(address common.Address, caller bind.ContractCaller) (*InterchainGasPaymasterCaller, error) {
	contract, err := bindInterchainGasPaymaster(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &InterchainGasPaymasterCaller{contract: contract}, nil
}

// NewInterchainGasPaymasterTransactor creates a new write-only instance of InterchainGasPaymaster, bound to a specific deployed contract.
func NewInterchainGasPaymasterTransactor(address common.Address, transactor bind.ContractTransactor) (*InterchainGasPaymasterTransactor, error) {
	contract, err := bindInterchainGasPaymaster(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &InterchainGasPaymasterTransactor{contract: contract}, nil
}

// NewInterchainGasPaymasterFilterer creates a new log filterer instance of InterchainGasPaymaster, bound to a specific deployed contract.
func NewInterchainGasPaymasterFilterer(address common.Address, filterer bind.ContractFilterer) (*InterchainGasPaymasterFilterer, error) {
	contract, err := bindInterchainGasPaymaster(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &InterchainGasPaymasterFilterer{contract: contract}, nil
}

// bindInterchainGasPaymaster binds a generic wrapper to an already deployed contract.
func bindInterchainGasPaymaster(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := InterchainGasPaymasterMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_InterchainGasPaymaster *InterchainGasPaymasterRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _InterchainGasPaymaster.Contract.InterchainGasPaymasterCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_InterchainGasPaymaster *InterchainGasPaymasterRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _InterchainGasPaymaster.Contract.InterchainGasPaymasterTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_InterchainGasPaymaster *InterchainGasPaymasterRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _InterchainGasPaymaster.Contract.InterchainGasPaymasterTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_InterchainGasPaymaster *InterchainGasPaymasterCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _InterchainGasPaymaster.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_InterchainGasPaymaster *InterchainGasPaymasterTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _InterchainGasPaymaster.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_InterchainGasPaymaster *InterchainGasPaymasterTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _InterchainGasPaymaster.Contract.contract.Transact(opts, method, params...)
}

// QuoteGasPayment is a free data retrieval call binding the contract method 0xa6929793.
//
// Solidity: function quoteGasPayment(uint32 _destinationDomain, uint256 _gasAmount) view returns(uint256)
func (_InterchainGasPaymaster *InterchainGasPaymasterCaller) QuoteGasPayment(opts *bind.CallOpts, _destinationDomain uint32, _gasAmount *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _InterchainGasPaymaster.contract.Call(opts, &out, "quoteGasPayment", _destinationDomain, _gasAmount)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// QuoteGasPayment is a free data retrieval call binding the contract method 0xa6929793.
//
// Solidity: function quoteGasPayment(uint32 _destinationDomain, uint256 _gasAmount) view returns(uint256)
func (_InterchainGasPaymaster *InterchainGasPaymasterSession) QuoteGasPayment(_destinationDomain uint32, _gasAmount *big.Int) (*big.Int, error) {
	return _InterchainGasPaymaster.Contract.QuoteGasPayment(&_InterchainGasPaymaster.CallOpts, _destinationDomain, _gasAmount)
}

// QuoteGasPayment is a free data retrieval call binding the contract method 0xa6929793.
//
// Solidity: function quoteGasPayment(uint32 _destinationDomain, uint256 _gasAmount) view returns(uint256)
func (_InterchainGasPaymaster *InterchainGasPaymasterCallerSession) QuoteGasPayment(_destinationDomain uint32, _gasAmount *big.Int) (*big.Int, error) {
	return _InterchainGasPaymaster.Contract.QuoteGasPayment(&_InterchainGasPaymaster.CallOpts, _destinationDomain, _gasAmount)
}

// PayForGas is a paid mutator transaction binding the contract method 0x11bf2c18.
//
// Solidity: function payForGas(bytes32 _messageId, uint32 _destinationDomain, uint256 _gasAmount, address _refundAddress) payable returns()
func (_InterchainGasPaymaster *InterchainGasPaymasterTransactor) PayForGas(opts *bind.TransactOpts, _messageId [32]byte, _destinationDomain uint32, _gasAmount *big.Int, _refundAddress common.Address) (*types.Transaction, error) {
	return _InterchainGasPaymaster.contract.Transact(opts, "payForGas", _messageId, _destinationDomain, _gasAmount, _refundAddress)
}

// PayForGas is a paid mutator transaction binding the contract method 0x11bf2c18.
//
// Solidity: function payForGas(bytes32 _messageId, uint32 _destinationDomain, uint256 _gasAmount, address _refundAddress) payable returns()
func (_InterchainGasPaymaster *InterchainGasPaymasterSession) PayForGas(_messageId [32]byte, _destinationDomain uint32, _gasAmount *big.Int, _refundAddress common.Address) (*types.Transaction, error) {
	return _InterchainGasPaymaster.Contract.PayForGas(&_InterchainGasPaymaster.TransactOpts, _messageId, _destinationDomain, _gasAmount, _refundAddress)
}

// PayForGas is a paid mutator transaction binding the contract method 0x11bf2c18.
//
// Solidity: function payForGas(bytes32 _messageId, uint32 _destinationDomain, uint256 _gasAmount, address _refundAddress) payable returns()
func (_InterchainGasPaymaster *InterchainGasPaymasterTransactorSession) PayForGas(_messageId [32]byte, _destinationDomain uint32, _gasAmount *big.Int, _refundAddress common.Address) (*types.Transaction, error) {
	return _InterchainGasPaymaster.Contract.PayForGas(&_InterchainGasPaymaster.TransactOpts, _messageId, _destinationDomain, _gasAmount, _refundAddress)
}

// InterchainGasPaymasterGasPaymentIterator is returned from FilterGasPayment and is used to iterate over the raw logs and unpacked data for GasPayment events raised by the InterchainGasPaymaster contract.
type InterchainGasPaymasterGasPaymentIterator struct {
	Event *InterchainGasPaymasterGasPayment // Event containing the contract specifics and raw log

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
func (it *InterchainGasPaymasterGasPaymentIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(InterchainGasPaymasterGasPayment)
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
		it.Event = new(InterchainGasPaymasterGasPayment)
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
func (it *InterchainGasPaymasterGasPaymentIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *InterchainGasPaymasterGasPaymentIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// InterchainGasPaymasterGasPayment represents a GasPayment event raised by the InterchainGasPaymaster contract.
type InterchainGasPaymasterGasPayment struct {
	MessageId         [32]byte
	DestinationDomain uint32
	GasAmount         *big.Int
	Payment           *big.Int
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterGasPayment is a free log retrieval operation binding the contract event 0x65695c3748edae85a24cc2c60b299b31f463050bc259150d2e5802ec8d11720a.
//
// Solidity: event GasPayment(bytes32 indexed messageId, uint32 indexed destinationDomain, uint256 gasAmount, uint256 payment)
func (_InterchainGasPaymaster *InterchainGasPaymasterFilterer) FilterGasPayment(opts *bind.FilterOpts, messageId [][32]byte, destinationDomain []uint32) (*InterchainGasPaymasterGasPaymentIterator, error) {

	var messageIdRule []interface{}
	for _, messageIdItem := range messageId {
		messageIdRule = append(messageIdRule, messageIdItem)
	}
	var destinationDomainRule []interface{}
	for _, destinationDomainItem := range destinationDomain {
		destinationDomainRule = append(destinationDomainRule, destinationDomainItem)
	}

	logs, sub, err := _InterchainGasPaymaster.contract.FilterLogs(opts, "GasPayment", messageIdRule, destinationDomainRule)
	if err != nil {
		return nil, err
	}
	return &InterchainGasPaymasterGasPaymentIterator{contract: _InterchainGasPaymaster.contract, event: "GasPayment", logs: logs, sub: sub}, nil
}

// WatchGasPayment is a free log subscription operation binding the contract event 0x65695c3748edae85a24cc2c60b299b31f463050bc259150d2e5802ec8d11720a.
//
// Solidity: event GasPayment(bytes32 indexed messageId, uint32 indexed destinationDomain, uint256 gasAmount, uint256 payment)
func (_InterchainGasPaymaster *InterchainGasPaymasterFilterer) WatchGasPayment(opts *bind.WatchOpts, sink chan<- *InterchainGasPaymasterGasPayment, messageId [][32]byte, destinationDomain []uint32) (event.Subscription, error) {

	var messageIdRule []interface{}
	for _, messageIdItem := range messageId {
		messageIdRule = append(messageIdRule, messageIdItem)
	}
	var destinationDomainRule []interface{}
	for _, destinationDomainItem := range destinationDomain {
		destinationDomainRule = append(destinationDomainRule, destinationDomainItem)
	}

	logs, sub, err := _InterchainGasPaymaster.contract.WatchLogs(opts, "GasPayment", messageIdRule, destinationDomainRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(InterchainGasPaymasterGasPayment)
				if err := _InterchainGasPaymaster.contract.UnpackLog(event, "GasPayment", log); err != nil {
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

// ParseGasPayment is a log parse operation binding the contract event 0x65695c3748edae85a24cc2c60b299b31f463050bc259150d2e5802ec8d11720a.
//
// Solidity: event GasPayment(bytes32 indexed messageId, uint32 indexed destinationDomain, uint256 gasAmount, uint256 payment)
func (_InterchainGasPaymaster *InterchainGasPaymasterFilterer) ParseGasPayment(log types.Log) (*InterchainGasPaymasterGasPayment, error) {
	event := new(InterchainGasPaymasterGasPayment)
	if err := _InterchainGasPaymaster.contract.UnpackLog(event, "GasPayment", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
