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

// StaticCallHeavyMetaData contains all meta data concerning the StaticCallHeavy contract.
var StaticCallHeavyMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"counter\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"performStaticCalls\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_targetContract\",\"type\":\"address\"}],\"name\":\"setTargetContract\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"targetContract\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b505f805561026e8061001f5f395ff3fe608060405234801561000f575f5ffd5b506004361061004a575f3560e01c80632d2f38151461004e57806347fc822f1461005857806361bc221a14610088578063bd90df70146100a3575b5f5ffd5b6100566100ce565b005b6100566100663660046101af565b600180546001600160a01b0319166001600160a01b0392909216919091179055565b6100905f5481565b6040519081526020015b60405180910390f35b6001546100b6906001600160a01b031681565b6040516001600160a01b03909116815260200161009a565b5f5b60648110156101ac576001546001600160a01b0316156101905760015f9054906101000a90046001600160a01b03166001600160a01b031663209652556040518163ffffffff1660e01b8152600401602060405180830381865afa925050508015610158575060408051601f3d908101601f19168201909252610155918101906101dc565b60015b610174575f8054908061016a83610207565b91905055506101a4565b805f5f828254610184919061021f565b909155506101a4915050565b5f8054908061019e83610207565b91905055505b6001016100d0565b50565b5f602082840312156101bf575f5ffd5b81356001600160a01b03811681146101d5575f5ffd5b9392505050565b5f602082840312156101ec575f5ffd5b5051919050565b634e487b7160e01b5f52601160045260245ffd5b5f60018201610218576102186101f3565b5060010190565b80820180821115610232576102326101f3565b9291505056fea2646970667358221220cfdb3f3d2a1761ff21d479367f44a9c9d285353655b57f105cccf5002341519964736f6c634300081e0033",
}

// StaticCallHeavyABI is the input ABI used to generate the binding from.
// Deprecated: Use StaticCallHeavyMetaData.ABI instead.
var StaticCallHeavyABI = StaticCallHeavyMetaData.ABI

// StaticCallHeavyBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use StaticCallHeavyMetaData.Bin instead.
var StaticCallHeavyBin = StaticCallHeavyMetaData.Bin

// DeployStaticCallHeavy deploys a new Ethereum contract, binding an instance of StaticCallHeavy to it.
func DeployStaticCallHeavy(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *StaticCallHeavy, error) {
	parsed, err := StaticCallHeavyMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(StaticCallHeavyBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &StaticCallHeavy{StaticCallHeavyCaller: StaticCallHeavyCaller{contract: contract}, StaticCallHeavyTransactor: StaticCallHeavyTransactor{contract: contract}, StaticCallHeavyFilterer: StaticCallHeavyFilterer{contract: contract}}, nil
}

// StaticCallHeavy is an auto generated Go binding around an Ethereum contract.
type StaticCallHeavy struct {
	StaticCallHeavyCaller     // Read-only binding to the contract
	StaticCallHeavyTransactor // Write-only binding to the contract
	StaticCallHeavyFilterer   // Log filterer for contract events
}

// StaticCallHeavyCaller is an auto generated read-only Go binding around an Ethereum contract.
type StaticCallHeavyCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StaticCallHeavyTransactor is an auto generated write-only Go binding around an Ethereum contract.
type StaticCallHeavyTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StaticCallHeavyFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type StaticCallHeavyFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StaticCallHeavySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type StaticCallHeavySession struct {
	Contract     *StaticCallHeavy  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// StaticCallHeavyCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type StaticCallHeavyCallerSession struct {
	Contract *StaticCallHeavyCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// StaticCallHeavyTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type StaticCallHeavyTransactorSession struct {
	Contract     *StaticCallHeavyTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// StaticCallHeavyRaw is an auto generated low-level Go binding around an Ethereum contract.
type StaticCallHeavyRaw struct {
	Contract *StaticCallHeavy // Generic contract binding to access the raw methods on
}

// StaticCallHeavyCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type StaticCallHeavyCallerRaw struct {
	Contract *StaticCallHeavyCaller // Generic read-only contract binding to access the raw methods on
}

// StaticCallHeavyTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type StaticCallHeavyTransactorRaw struct {
	Contract *StaticCallHeavyTransactor // Generic write-only contract binding to access the raw methods on
}

// NewStaticCallHeavy creates a new instance of StaticCallHeavy, bound to a specific deployed contract.
func NewStaticCallHeavy(address common.Address, backend bind.ContractBackend) (*StaticCallHeavy, error) {
	contract, err := bindStaticCallHeavy(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &StaticCallHeavy{StaticCallHeavyCaller: StaticCallHeavyCaller{contract: contract}, StaticCallHeavyTransactor: StaticCallHeavyTransactor{contract: contract}, StaticCallHeavyFilterer: StaticCallHeavyFilterer{contract: contract}}, nil
}

// NewStaticCallHeavyCaller creates a new read-only instance of StaticCallHeavy, bound to a specific deployed contract.
func NewStaticCallHeavyCaller(address common.Address, caller bind.ContractCaller) (*StaticCallHeavyCaller, error) {
	contract, err := bindStaticCallHeavy(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &StaticCallHeavyCaller{contract: contract}, nil
}

// NewStaticCallHeavyTransactor creates a new write-only instance of StaticCallHeavy, bound to a specific deployed contract.
func NewStaticCallHeavyTransactor(address common.Address, transactor bind.ContractTransactor) (*StaticCallHeavyTransactor, error) {
	contract, err := bindStaticCallHeavy(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &StaticCallHeavyTransactor{contract: contract}, nil
}

// NewStaticCallHeavyFilterer creates a new log filterer instance of StaticCallHeavy, bound to a specific deployed contract.
func NewStaticCallHeavyFilterer(address common.Address, filterer bind.ContractFilterer) (*StaticCallHeavyFilterer, error) {
	contract, err := bindStaticCallHeavy(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &StaticCallHeavyFilterer{contract: contract}, nil
}

// bindStaticCallHeavy binds a generic wrapper to an already deployed contract.
func bindStaticCallHeavy(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := StaticCallHeavyMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_StaticCallHeavy *StaticCallHeavyRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _StaticCallHeavy.Contract.StaticCallHeavyCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_StaticCallHeavy *StaticCallHeavyRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _StaticCallHeavy.Contract.StaticCallHeavyTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_StaticCallHeavy *StaticCallHeavyRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _StaticCallHeavy.Contract.StaticCallHeavyTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_StaticCallHeavy *StaticCallHeavyCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _StaticCallHeavy.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_StaticCallHeavy *StaticCallHeavyTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _StaticCallHeavy.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_StaticCallHeavy *StaticCallHeavyTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _StaticCallHeavy.Contract.contract.Transact(opts, method, params...)
}

// Counter is a free data retrieval call binding the contract method 0x61bc221a.
//
// Solidity: function counter() view returns(uint256)
func (_StaticCallHeavy *StaticCallHeavyCaller) Counter(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _StaticCallHeavy.contract.Call(opts, &out, "counter")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Counter is a free data retrieval call binding the contract method 0x61bc221a.
//
// Solidity: function counter() view returns(uint256)
func (_StaticCallHeavy *StaticCallHeavySession) Counter() (*big.Int, error) {
	return _StaticCallHeavy.Contract.Counter(&_StaticCallHeavy.CallOpts)
}

// Counter is a free data retrieval call binding the contract method 0x61bc221a.
//
// Solidity: function counter() view returns(uint256)
func (_StaticCallHeavy *StaticCallHeavyCallerSession) Counter() (*big.Int, error) {
	return _StaticCallHeavy.Contract.Counter(&_StaticCallHeavy.CallOpts)
}

// TargetContract is a free data retrieval call binding the contract method 0xbd90df70.
//
// Solidity: function targetContract() view returns(address)
func (_StaticCallHeavy *StaticCallHeavyCaller) TargetContract(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _StaticCallHeavy.contract.Call(opts, &out, "targetContract")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// TargetContract is a free data retrieval call binding the contract method 0xbd90df70.
//
// Solidity: function targetContract() view returns(address)
func (_StaticCallHeavy *StaticCallHeavySession) TargetContract() (common.Address, error) {
	return _StaticCallHeavy.Contract.TargetContract(&_StaticCallHeavy.CallOpts)
}

// TargetContract is a free data retrieval call binding the contract method 0xbd90df70.
//
// Solidity: function targetContract() view returns(address)
func (_StaticCallHeavy *StaticCallHeavyCallerSession) TargetContract() (common.Address, error) {
	return _StaticCallHeavy.Contract.TargetContract(&_StaticCallHeavy.CallOpts)
}

// PerformStaticCalls is a paid mutator transaction binding the contract method 0x2d2f3815.
//
// Solidity: function performStaticCalls() returns()
func (_StaticCallHeavy *StaticCallHeavyTransactor) PerformStaticCalls(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _StaticCallHeavy.contract.Transact(opts, "performStaticCalls")
}

// PerformStaticCalls is a paid mutator transaction binding the contract method 0x2d2f3815.
//
// Solidity: function performStaticCalls() returns()
func (_StaticCallHeavy *StaticCallHeavySession) PerformStaticCalls() (*types.Transaction, error) {
	return _StaticCallHeavy.Contract.PerformStaticCalls(&_StaticCallHeavy.TransactOpts)
}

// PerformStaticCalls is a paid mutator transaction binding the contract method 0x2d2f3815.
//
// Solidity: function performStaticCalls() returns()
func (_StaticCallHeavy *StaticCallHeavyTransactorSession) PerformStaticCalls() (*types.Transaction, error) {
	return _StaticCallHeavy.Contract.PerformStaticCalls(&_StaticCallHeavy.TransactOpts)
}

// SetTargetContract is a paid mutator transaction binding the contract method 0x47fc822f.
//
// Solidity: function setTargetContract(address _targetContract) returns()
func (_StaticCallHeavy *StaticCallHeavyTransactor) SetTargetContract(opts *bind.TransactOpts, _targetContract common.Address) (*types.Transaction, error) {
	return _StaticCallHeavy.contract.Transact(opts, "setTargetContract", _targetContract)
}

// SetTargetContract is a paid mutator transaction binding the contract method 0x47fc822f.
//
// Solidity: function setTargetContract(address _targetContract) returns()
func (_StaticCallHeavy *StaticCallHeavySession) SetTargetContract(_targetContract common.Address) (*types.Transaction, error) {
	return _StaticCallHeavy.Contract.SetTargetContract(&_StaticCallHeavy.TransactOpts, _targetContract)
}

// SetTargetContract is a paid mutator transaction binding the contract method 0x47fc822f.
//
// Solidity: function setTargetContract(address _targetContract) returns()
func (_StaticCallHeavy *StaticCallHeavyTransactorSession) SetTargetContract(_targetContract common.Address) (*types.Transaction, error) {
	return _StaticCallHeavy.Contract.SetTargetContract(&_StaticCallHeavy.TransactOpts, _targetContract)
}
