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

// StorageRWv1MetaData contains all meta data concerning the StorageRWv1 contract.
var StorageRWv1MetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"VERSION\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"slot\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_pad\",\"type\":\"bytes\"}],\"name\":\"read\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"readAccumulator\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"slot\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_pad\",\"type\":\"bytes\"}],\"name\":\"rmw\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"store\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"slot\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_pad\",\"type\":\"bytes\"}],\"name\":\"write\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b50610256806100206000396000f3fe608060405234801561001057600080fd5b50600436106100625760003560e01c806321988bf31461006757806322746b0714610082578063396e9b3a146100ab5780636057361d146100cd578063829e7369146100ed578063ffa1ad7414610117575b600080fd5b61007060015481565b60405190815260200160405180910390f35b6100a9610090366004610168565b5050600090815260208190526040902080546001019055565b005b6100a96100b93660046101b4565b505060009182526020829052604090912055565b6100706100db366004610207565b60006020819052908152604090205481565b6100a96100fb366004610168565b5050600090815260208190526040902054600180549091019055565b610070600181565b60008083601f84011261013157600080fd5b50813567ffffffffffffffff81111561014957600080fd5b60208301915083602082850101111561016157600080fd5b9250929050565b60008060006040848603121561017d57600080fd5b83359250602084013567ffffffffffffffff81111561019b57600080fd5b6101a78682870161011f565b9497909650939450505050565b600080600080606085870312156101ca57600080fd5b8435935060208501359250604085013567ffffffffffffffff8111156101ef57600080fd5b6101fb8782880161011f565b95989497509550505050565b60006020828403121561021957600080fd5b503591905056fea2646970667358221220d26ea7a4ca87e2bdfc3495f24e75b2694a00ebcca5fdeb15e812eec62089ee1564736f6c63430008130033",
}

// StorageRWv1ABI is the input ABI used to generate the binding from.
// Deprecated: Use StorageRWv1MetaData.ABI instead.
var StorageRWv1ABI = StorageRWv1MetaData.ABI

// StorageRWv1Bin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use StorageRWv1MetaData.Bin instead.
var StorageRWv1Bin = StorageRWv1MetaData.Bin

// DeployStorageRWv1 deploys a new Ethereum contract, binding an instance of StorageRWv1 to it.
func DeployStorageRWv1(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *StorageRWv1, error) {
	parsed, err := StorageRWv1MetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(StorageRWv1Bin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &StorageRWv1{StorageRWv1Caller: StorageRWv1Caller{contract: contract}, StorageRWv1Transactor: StorageRWv1Transactor{contract: contract}, StorageRWv1Filterer: StorageRWv1Filterer{contract: contract}}, nil
}

// StorageRWv1 is an auto generated Go binding around an Ethereum contract.
type StorageRWv1 struct {
	StorageRWv1Caller     // Read-only binding to the contract
	StorageRWv1Transactor // Write-only binding to the contract
	StorageRWv1Filterer   // Log filterer for contract events
}

// StorageRWv1Caller is an auto generated read-only Go binding around an Ethereum contract.
type StorageRWv1Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StorageRWv1Transactor is an auto generated write-only Go binding around an Ethereum contract.
type StorageRWv1Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StorageRWv1Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type StorageRWv1Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StorageRWv1Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type StorageRWv1Session struct {
	Contract     *StorageRWv1      // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// StorageRWv1CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type StorageRWv1CallerSession struct {
	Contract *StorageRWv1Caller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts      // Call options to use throughout this session
}

// StorageRWv1TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type StorageRWv1TransactorSession struct {
	Contract     *StorageRWv1Transactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// StorageRWv1Raw is an auto generated low-level Go binding around an Ethereum contract.
type StorageRWv1Raw struct {
	Contract *StorageRWv1 // Generic contract binding to access the raw methods on
}

// StorageRWv1CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type StorageRWv1CallerRaw struct {
	Contract *StorageRWv1Caller // Generic read-only contract binding to access the raw methods on
}

// StorageRWv1TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type StorageRWv1TransactorRaw struct {
	Contract *StorageRWv1Transactor // Generic write-only contract binding to access the raw methods on
}

// NewStorageRWv1 creates a new instance of StorageRWv1, bound to a specific deployed contract.
func NewStorageRWv1(address common.Address, backend bind.ContractBackend) (*StorageRWv1, error) {
	contract, err := bindStorageRWv1(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &StorageRWv1{StorageRWv1Caller: StorageRWv1Caller{contract: contract}, StorageRWv1Transactor: StorageRWv1Transactor{contract: contract}, StorageRWv1Filterer: StorageRWv1Filterer{contract: contract}}, nil
}

// NewStorageRWv1Caller creates a new read-only instance of StorageRWv1, bound to a specific deployed contract.
func NewStorageRWv1Caller(address common.Address, caller bind.ContractCaller) (*StorageRWv1Caller, error) {
	contract, err := bindStorageRWv1(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &StorageRWv1Caller{contract: contract}, nil
}

// NewStorageRWv1Transactor creates a new write-only instance of StorageRWv1, bound to a specific deployed contract.
func NewStorageRWv1Transactor(address common.Address, transactor bind.ContractTransactor) (*StorageRWv1Transactor, error) {
	contract, err := bindStorageRWv1(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &StorageRWv1Transactor{contract: contract}, nil
}

// NewStorageRWv1Filterer creates a new log filterer instance of StorageRWv1, bound to a specific deployed contract.
func NewStorageRWv1Filterer(address common.Address, filterer bind.ContractFilterer) (*StorageRWv1Filterer, error) {
	contract, err := bindStorageRWv1(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &StorageRWv1Filterer{contract: contract}, nil
}

// bindStorageRWv1 binds a generic wrapper to an already deployed contract.
func bindStorageRWv1(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := StorageRWv1MetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_StorageRWv1 *StorageRWv1Raw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _StorageRWv1.Contract.StorageRWv1Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_StorageRWv1 *StorageRWv1Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _StorageRWv1.Contract.StorageRWv1Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_StorageRWv1 *StorageRWv1Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _StorageRWv1.Contract.StorageRWv1Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_StorageRWv1 *StorageRWv1CallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _StorageRWv1.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_StorageRWv1 *StorageRWv1TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _StorageRWv1.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_StorageRWv1 *StorageRWv1TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _StorageRWv1.Contract.contract.Transact(opts, method, params...)
}

// VERSION is a free data retrieval call binding the contract method 0xffa1ad74.
//
// Solidity: function VERSION() view returns(uint256)
func (_StorageRWv1 *StorageRWv1Caller) VERSION(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _StorageRWv1.contract.Call(opts, &out, "VERSION")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// VERSION is a free data retrieval call binding the contract method 0xffa1ad74.
//
// Solidity: function VERSION() view returns(uint256)
func (_StorageRWv1 *StorageRWv1Session) VERSION() (*big.Int, error) {
	return _StorageRWv1.Contract.VERSION(&_StorageRWv1.CallOpts)
}

// VERSION is a free data retrieval call binding the contract method 0xffa1ad74.
//
// Solidity: function VERSION() view returns(uint256)
func (_StorageRWv1 *StorageRWv1CallerSession) VERSION() (*big.Int, error) {
	return _StorageRWv1.Contract.VERSION(&_StorageRWv1.CallOpts)
}

// ReadAccumulator is a free data retrieval call binding the contract method 0x21988bf3.
//
// Solidity: function readAccumulator() view returns(uint256)
func (_StorageRWv1 *StorageRWv1Caller) ReadAccumulator(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _StorageRWv1.contract.Call(opts, &out, "readAccumulator")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ReadAccumulator is a free data retrieval call binding the contract method 0x21988bf3.
//
// Solidity: function readAccumulator() view returns(uint256)
func (_StorageRWv1 *StorageRWv1Session) ReadAccumulator() (*big.Int, error) {
	return _StorageRWv1.Contract.ReadAccumulator(&_StorageRWv1.CallOpts)
}

// ReadAccumulator is a free data retrieval call binding the contract method 0x21988bf3.
//
// Solidity: function readAccumulator() view returns(uint256)
func (_StorageRWv1 *StorageRWv1CallerSession) ReadAccumulator() (*big.Int, error) {
	return _StorageRWv1.Contract.ReadAccumulator(&_StorageRWv1.CallOpts)
}

// Store is a free data retrieval call binding the contract method 0x6057361d.
//
// Solidity: function store(uint256 ) view returns(uint256)
func (_StorageRWv1 *StorageRWv1Caller) Store(opts *bind.CallOpts, arg0 *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _StorageRWv1.contract.Call(opts, &out, "store", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Store is a free data retrieval call binding the contract method 0x6057361d.
//
// Solidity: function store(uint256 ) view returns(uint256)
func (_StorageRWv1 *StorageRWv1Session) Store(arg0 *big.Int) (*big.Int, error) {
	return _StorageRWv1.Contract.Store(&_StorageRWv1.CallOpts, arg0)
}

// Store is a free data retrieval call binding the contract method 0x6057361d.
//
// Solidity: function store(uint256 ) view returns(uint256)
func (_StorageRWv1 *StorageRWv1CallerSession) Store(arg0 *big.Int) (*big.Int, error) {
	return _StorageRWv1.Contract.Store(&_StorageRWv1.CallOpts, arg0)
}

// Read is a paid mutator transaction binding the contract method 0x829e7369.
//
// Solidity: function read(uint256 slot, bytes _pad) returns()
func (_StorageRWv1 *StorageRWv1Transactor) Read(opts *bind.TransactOpts, slot *big.Int, _pad []byte) (*types.Transaction, error) {
	return _StorageRWv1.contract.Transact(opts, "read", slot, _pad)
}

// Read is a paid mutator transaction binding the contract method 0x829e7369.
//
// Solidity: function read(uint256 slot, bytes _pad) returns()
func (_StorageRWv1 *StorageRWv1Session) Read(slot *big.Int, _pad []byte) (*types.Transaction, error) {
	return _StorageRWv1.Contract.Read(&_StorageRWv1.TransactOpts, slot, _pad)
}

// Read is a paid mutator transaction binding the contract method 0x829e7369.
//
// Solidity: function read(uint256 slot, bytes _pad) returns()
func (_StorageRWv1 *StorageRWv1TransactorSession) Read(slot *big.Int, _pad []byte) (*types.Transaction, error) {
	return _StorageRWv1.Contract.Read(&_StorageRWv1.TransactOpts, slot, _pad)
}

// Rmw is a paid mutator transaction binding the contract method 0x22746b07.
//
// Solidity: function rmw(uint256 slot, bytes _pad) returns()
func (_StorageRWv1 *StorageRWv1Transactor) Rmw(opts *bind.TransactOpts, slot *big.Int, _pad []byte) (*types.Transaction, error) {
	return _StorageRWv1.contract.Transact(opts, "rmw", slot, _pad)
}

// Rmw is a paid mutator transaction binding the contract method 0x22746b07.
//
// Solidity: function rmw(uint256 slot, bytes _pad) returns()
func (_StorageRWv1 *StorageRWv1Session) Rmw(slot *big.Int, _pad []byte) (*types.Transaction, error) {
	return _StorageRWv1.Contract.Rmw(&_StorageRWv1.TransactOpts, slot, _pad)
}

// Rmw is a paid mutator transaction binding the contract method 0x22746b07.
//
// Solidity: function rmw(uint256 slot, bytes _pad) returns()
func (_StorageRWv1 *StorageRWv1TransactorSession) Rmw(slot *big.Int, _pad []byte) (*types.Transaction, error) {
	return _StorageRWv1.Contract.Rmw(&_StorageRWv1.TransactOpts, slot, _pad)
}

// Write is a paid mutator transaction binding the contract method 0x396e9b3a.
//
// Solidity: function write(uint256 slot, uint256 value, bytes _pad) returns()
func (_StorageRWv1 *StorageRWv1Transactor) Write(opts *bind.TransactOpts, slot *big.Int, value *big.Int, _pad []byte) (*types.Transaction, error) {
	return _StorageRWv1.contract.Transact(opts, "write", slot, value, _pad)
}

// Write is a paid mutator transaction binding the contract method 0x396e9b3a.
//
// Solidity: function write(uint256 slot, uint256 value, bytes _pad) returns()
func (_StorageRWv1 *StorageRWv1Session) Write(slot *big.Int, value *big.Int, _pad []byte) (*types.Transaction, error) {
	return _StorageRWv1.Contract.Write(&_StorageRWv1.TransactOpts, slot, value, _pad)
}

// Write is a paid mutator transaction binding the contract method 0x396e9b3a.
//
// Solidity: function write(uint256 slot, uint256 value, bytes _pad) returns()
func (_StorageRWv1 *StorageRWv1TransactorSession) Write(slot *big.Int, value *big.Int, _pad []byte) (*types.Transaction, error) {
	return _StorageRWv1.Contract.Write(&_StorageRWv1.TransactOpts, slot, value, _pad)
}
