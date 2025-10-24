// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindings

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

// SimpleTargetMetaData contains all meta data concerning the SimpleTarget contract.
var SimpleTargetMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"getValue\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"setValue\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"value\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b50602a5f5560b780601e5f395ff3fe6080604052348015600e575f5ffd5b5060043610603a575f3560e01c80632096525514603e5780633fa4f2451460535780635524107714605a575b5f5ffd5b5f545b60405190815260200160405180910390f35b60415f5481565b60696065366004606b565b5f55565b005b5f60208284031215607a575f5ffd5b503591905056fea264697066735822122026e5c7667b6930bea40710345760ac46161d9baf0b13e477c57eb92b86d5a47f64736f6c634300081e0033",
}

// SimpleTargetABI is the input ABI used to generate the binding from.
// Deprecated: Use SimpleTargetMetaData.ABI instead.
var SimpleTargetABI = SimpleTargetMetaData.ABI

// SimpleTargetBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use SimpleTargetMetaData.Bin instead.
var SimpleTargetBin = SimpleTargetMetaData.Bin

// DeploySimpleTarget deploys a new Ethereum contract, binding an instance of SimpleTarget to it.
func DeploySimpleTarget(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *SimpleTarget, error) {
	parsed, err := SimpleTargetMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(SimpleTargetBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &SimpleTarget{SimpleTargetCaller: SimpleTargetCaller{contract: contract}, SimpleTargetTransactor: SimpleTargetTransactor{contract: contract}, SimpleTargetFilterer: SimpleTargetFilterer{contract: contract}}, nil
}

// SimpleTarget is an auto generated Go binding around an Ethereum contract.
type SimpleTarget struct {
	SimpleTargetCaller     // Read-only binding to the contract
	SimpleTargetTransactor // Write-only binding to the contract
	SimpleTargetFilterer   // Log filterer for contract events
}

// SimpleTargetCaller is an auto generated read-only Go binding around an Ethereum contract.
type SimpleTargetCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SimpleTargetTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SimpleTargetTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SimpleTargetFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SimpleTargetFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SimpleTargetSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SimpleTargetSession struct {
	Contract     *SimpleTarget     // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// SimpleTargetCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SimpleTargetCallerSession struct {
	Contract *SimpleTargetCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts       // Call options to use throughout this session
}

// SimpleTargetTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SimpleTargetTransactorSession struct {
	Contract     *SimpleTargetTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// SimpleTargetRaw is an auto generated low-level Go binding around an Ethereum contract.
type SimpleTargetRaw struct {
	Contract *SimpleTarget // Generic contract binding to access the raw methods on
}

// SimpleTargetCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SimpleTargetCallerRaw struct {
	Contract *SimpleTargetCaller // Generic read-only contract binding to access the raw methods on
}

// SimpleTargetTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SimpleTargetTransactorRaw struct {
	Contract *SimpleTargetTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSimpleTarget creates a new instance of SimpleTarget, bound to a specific deployed contract.
func NewSimpleTarget(address common.Address, backend bind.ContractBackend) (*SimpleTarget, error) {
	contract, err := bindSimpleTarget(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SimpleTarget{SimpleTargetCaller: SimpleTargetCaller{contract: contract}, SimpleTargetTransactor: SimpleTargetTransactor{contract: contract}, SimpleTargetFilterer: SimpleTargetFilterer{contract: contract}}, nil
}

// NewSimpleTargetCaller creates a new read-only instance of SimpleTarget, bound to a specific deployed contract.
func NewSimpleTargetCaller(address common.Address, caller bind.ContractCaller) (*SimpleTargetCaller, error) {
	contract, err := bindSimpleTarget(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SimpleTargetCaller{contract: contract}, nil
}

// NewSimpleTargetTransactor creates a new write-only instance of SimpleTarget, bound to a specific deployed contract.
func NewSimpleTargetTransactor(address common.Address, transactor bind.ContractTransactor) (*SimpleTargetTransactor, error) {
	contract, err := bindSimpleTarget(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SimpleTargetTransactor{contract: contract}, nil
}

// NewSimpleTargetFilterer creates a new log filterer instance of SimpleTarget, bound to a specific deployed contract.
func NewSimpleTargetFilterer(address common.Address, filterer bind.ContractFilterer) (*SimpleTargetFilterer, error) {
	contract, err := bindSimpleTarget(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SimpleTargetFilterer{contract: contract}, nil
}

// bindSimpleTarget binds a generic wrapper to an already deployed contract.
func bindSimpleTarget(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := SimpleTargetMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SimpleTarget *SimpleTargetRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SimpleTarget.Contract.SimpleTargetCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SimpleTarget *SimpleTargetRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SimpleTarget.Contract.SimpleTargetTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SimpleTarget *SimpleTargetRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SimpleTarget.Contract.SimpleTargetTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SimpleTarget *SimpleTargetCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SimpleTarget.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SimpleTarget *SimpleTargetTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SimpleTarget.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SimpleTarget *SimpleTargetTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SimpleTarget.Contract.contract.Transact(opts, method, params...)
}

// GetValue is a free data retrieval call binding the contract method 0x20965255.
//
// Solidity: function getValue() view returns(uint256)
func (_SimpleTarget *SimpleTargetCaller) GetValue(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _SimpleTarget.contract.Call(opts, &out, "getValue")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetValue is a free data retrieval call binding the contract method 0x20965255.
//
// Solidity: function getValue() view returns(uint256)
func (_SimpleTarget *SimpleTargetSession) GetValue() (*big.Int, error) {
	return _SimpleTarget.Contract.GetValue(&_SimpleTarget.CallOpts)
}

// GetValue is a free data retrieval call binding the contract method 0x20965255.
//
// Solidity: function getValue() view returns(uint256)
func (_SimpleTarget *SimpleTargetCallerSession) GetValue() (*big.Int, error) {
	return _SimpleTarget.Contract.GetValue(&_SimpleTarget.CallOpts)
}

// Value is a free data retrieval call binding the contract method 0x3fa4f245.
//
// Solidity: function value() view returns(uint256)
func (_SimpleTarget *SimpleTargetCaller) Value(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _SimpleTarget.contract.Call(opts, &out, "value")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Value is a free data retrieval call binding the contract method 0x3fa4f245.
//
// Solidity: function value() view returns(uint256)
func (_SimpleTarget *SimpleTargetSession) Value() (*big.Int, error) {
	return _SimpleTarget.Contract.Value(&_SimpleTarget.CallOpts)
}

// Value is a free data retrieval call binding the contract method 0x3fa4f245.
//
// Solidity: function value() view returns(uint256)
func (_SimpleTarget *SimpleTargetCallerSession) Value() (*big.Int, error) {
	return _SimpleTarget.Contract.Value(&_SimpleTarget.CallOpts)
}

// SetValue is a paid mutator transaction binding the contract method 0x55241077.
//
// Solidity: function setValue(uint256 _value) returns()
func (_SimpleTarget *SimpleTargetTransactor) SetValue(opts *bind.TransactOpts, _value *big.Int) (*types.Transaction, error) {
	return _SimpleTarget.contract.Transact(opts, "setValue", _value)
}

// SetValue is a paid mutator transaction binding the contract method 0x55241077.
//
// Solidity: function setValue(uint256 _value) returns()
func (_SimpleTarget *SimpleTargetSession) SetValue(_value *big.Int) (*types.Transaction, error) {
	return _SimpleTarget.Contract.SetValue(&_SimpleTarget.TransactOpts, _value)
}

// SetValue is a paid mutator transaction binding the contract method 0x55241077.
//
// Solidity: function setValue(uint256 _value) returns()
func (_SimpleTarget *SimpleTargetTransactorSession) SetValue(_value *big.Int) (*types.Transaction, error) {
	return _SimpleTarget.Contract.SetValue(&_SimpleTarget.TransactOpts, _value)
}
