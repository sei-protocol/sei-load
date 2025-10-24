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

// ERC20ConflictMetaData contains all meta data concerning the ERC20Conflict contract.
var ERC20ConflictMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"string\",\"name\":\"_name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"_symbol\",\"type\":\"string\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"DEFAULT_BALANCE\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"counter\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561000f575f5ffd5b5060405161094d38038061094d83398101604081905261002e916100f8565b5f61003983826101e1565b50600161004682826101e1565b50506002805460ff191660121790555061029b565b634e487b7160e01b5f52604160045260245ffd5b5f82601f83011261007e575f5ffd5b81516001600160401b038111156100975761009761005b565b604051601f8201601f19908116603f011681016001600160401b03811182821017156100c5576100c561005b565b6040528181528382016020018510156100dc575f5ffd5b8160208501602083015e5f918101602001919091529392505050565b5f5f60408385031215610109575f5ffd5b82516001600160401b0381111561011e575f5ffd5b61012a8582860161006f565b602085015190935090506001600160401b03811115610147575f5ffd5b6101538582860161006f565b9150509250929050565b600181811c9082168061017157607f821691505b60208210810361018f57634e487b7160e01b5f52602260045260245ffd5b50919050565b601f8211156101dc57805f5260205f20601f840160051c810160208510156101ba5750805b601f840160051c820191505b818110156101d9575f81556001016101c6565b50505b505050565b81516001600160401b038111156101fa576101fa61005b565b61020e81610208845461015d565b84610195565b6020601f821160018114610240575f83156102295750848201515b5f19600385901b1c1916600184901b1784556101d9565b5f84815260208120601f198516915b8281101561026f578785015182556020948501946001909201910161024f565b508482101561028c57868401515f19600387901b60f8161c191681555b50505050600190811b01905550565b6106a5806102a85f395ff3fe608060405234801561000f575f5ffd5b50600436106100a6575f3560e01c806361bc221a1161006e57806361bc221a1461013457806370a082311461013d57806370f357351461015057806395d89b4114610160578063a9059cbb14610168578063dd62ed3e1461017b575f5ffd5b806306fdde03146100aa578063095ea7b3146100c857806318160ddd146100eb57806323b872dd14610102578063313ce56714610115575b5f5ffd5b6100b26101b3565b6040516100bf91906104e9565b60405180910390f35b6100db6100d6366004610539565b61023e565b60405190151581526020016100bf565b6100f460035481565b6040519081526020016100bf565b6100db610110366004610561565b6102aa565b6002546101229060ff1681565b60405160ff90911681526020016100bf565b6100f460045481565b6100f461014b36600461059b565b6103bd565b6100f4683635c9adc5dea0000081565b6100b26103f2565b6100db610176366004610539565b6103ff565b6100f46101893660046105b4565b6001600160a01b039182165f90815260066020908152604080832093909416825291909152205490565b5f80546101bf906105e5565b80601f01602080910402602001604051908101604052809291908181526020018280546101eb906105e5565b80156102365780601f1061020d57610100808354040283529160200191610236565b820191905f5260205f20905b81548152906001019060200180831161021957829003601f168201915b505050505081565b335f8181526006602090815260408083206001600160a01b038716808552925280832085905551919290917f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925906102989086815260200190565b60405180910390a35060015b92915050565b6001600160a01b0383165f908152600560205260408120548211156102f5576102dc683635c9adc5dea0000083610631565b6001600160a01b0385165f908152600560205260409020555b60048054905f61030483610644565b90915550506001600160a01b0384165f908152600560205260408120805484929061033090849061065c565b90915550506001600160a01b0383165f90815260056020526040902054610358908390610631565b6001600160a01b038085165f8181526005602052604090819020939093559151908616907fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef906103ab9086815260200190565b60405180910390a35060019392505050565b6001600160a01b0381165f90815260056020526040812054806103e957683635c9adc5dea000006103eb565b805b9392505050565b600180546101bf906105e5565b335f9081526005602052604081205482111561043857610428683635c9adc5dea0000083610631565b335f908152600560205260409020555b335f908152600560205260408120805484929061045690849061065c565b90915550506001600160a01b0383165f9081526005602052604090205461047e908390610631565b6001600160a01b0384165f9081526005602052604081209190915560048054916104a783610644565b90915550506040518281526001600160a01b0384169033907fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef90602001610298565b602081525f82518060208401528060208501604085015e5f604082850101526040601f19601f83011684010191505092915050565b80356001600160a01b0381168114610534575f5ffd5b919050565b5f5f6040838503121561054a575f5ffd5b6105538361051e565b946020939093013593505050565b5f5f5f60608486031215610573575f5ffd5b61057c8461051e565b925061058a6020850161051e565b929592945050506040919091013590565b5f602082840312156105ab575f5ffd5b6103eb8261051e565b5f5f604083850312156105c5575f5ffd5b6105ce8361051e565b91506105dc6020840161051e565b90509250929050565b600181811c908216806105f957607f821691505b60208210810361061757634e487b7160e01b5f52602260045260245ffd5b50919050565b634e487b7160e01b5f52601160045260245ffd5b808201808211156102a4576102a461061d565b5f600182016106555761065561061d565b5060010190565b818103818111156102a4576102a461061d56fea264697066735822122038b7b1c6cab7dacedb114b6dc85cf447dff7a2274dba5f4a2d20b106d16efca164736f6c634300081e0033",
}

// ERC20ConflictABI is the input ABI used to generate the binding from.
// Deprecated: Use ERC20ConflictMetaData.ABI instead.
var ERC20ConflictABI = ERC20ConflictMetaData.ABI

// ERC20ConflictBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ERC20ConflictMetaData.Bin instead.
var ERC20ConflictBin = ERC20ConflictMetaData.Bin

// DeployERC20Conflict deploys a new Ethereum contract, binding an instance of ERC20Conflict to it.
func DeployERC20Conflict(auth *bind.TransactOpts, backend bind.ContractBackend, _name string, _symbol string) (common.Address, *types.Transaction, *ERC20Conflict, error) {
	parsed, err := ERC20ConflictMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ERC20ConflictBin), backend, _name, _symbol)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ERC20Conflict{ERC20ConflictCaller: ERC20ConflictCaller{contract: contract}, ERC20ConflictTransactor: ERC20ConflictTransactor{contract: contract}, ERC20ConflictFilterer: ERC20ConflictFilterer{contract: contract}}, nil
}

// ERC20Conflict is an auto generated Go binding around an Ethereum contract.
type ERC20Conflict struct {
	ERC20ConflictCaller     // Read-only binding to the contract
	ERC20ConflictTransactor // Write-only binding to the contract
	ERC20ConflictFilterer   // Log filterer for contract events
}

// ERC20ConflictCaller is an auto generated read-only Go binding around an Ethereum contract.
type ERC20ConflictCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ERC20ConflictTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ERC20ConflictTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ERC20ConflictFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ERC20ConflictFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ERC20ConflictSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ERC20ConflictSession struct {
	Contract     *ERC20Conflict    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ERC20ConflictCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ERC20ConflictCallerSession struct {
	Contract *ERC20ConflictCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// ERC20ConflictTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ERC20ConflictTransactorSession struct {
	Contract     *ERC20ConflictTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// ERC20ConflictRaw is an auto generated low-level Go binding around an Ethereum contract.
type ERC20ConflictRaw struct {
	Contract *ERC20Conflict // Generic contract binding to access the raw methods on
}

// ERC20ConflictCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ERC20ConflictCallerRaw struct {
	Contract *ERC20ConflictCaller // Generic read-only contract binding to access the raw methods on
}

// ERC20ConflictTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ERC20ConflictTransactorRaw struct {
	Contract *ERC20ConflictTransactor // Generic write-only contract binding to access the raw methods on
}

// NewERC20Conflict creates a new instance of ERC20Conflict, bound to a specific deployed contract.
func NewERC20Conflict(address common.Address, backend bind.ContractBackend) (*ERC20Conflict, error) {
	contract, err := bindERC20Conflict(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ERC20Conflict{ERC20ConflictCaller: ERC20ConflictCaller{contract: contract}, ERC20ConflictTransactor: ERC20ConflictTransactor{contract: contract}, ERC20ConflictFilterer: ERC20ConflictFilterer{contract: contract}}, nil
}

// NewERC20ConflictCaller creates a new read-only instance of ERC20Conflict, bound to a specific deployed contract.
func NewERC20ConflictCaller(address common.Address, caller bind.ContractCaller) (*ERC20ConflictCaller, error) {
	contract, err := bindERC20Conflict(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ERC20ConflictCaller{contract: contract}, nil
}

// NewERC20ConflictTransactor creates a new write-only instance of ERC20Conflict, bound to a specific deployed contract.
func NewERC20ConflictTransactor(address common.Address, transactor bind.ContractTransactor) (*ERC20ConflictTransactor, error) {
	contract, err := bindERC20Conflict(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ERC20ConflictTransactor{contract: contract}, nil
}

// NewERC20ConflictFilterer creates a new log filterer instance of ERC20Conflict, bound to a specific deployed contract.
func NewERC20ConflictFilterer(address common.Address, filterer bind.ContractFilterer) (*ERC20ConflictFilterer, error) {
	contract, err := bindERC20Conflict(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ERC20ConflictFilterer{contract: contract}, nil
}

// bindERC20Conflict binds a generic wrapper to an already deployed contract.
func bindERC20Conflict(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ERC20ConflictMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ERC20Conflict *ERC20ConflictRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ERC20Conflict.Contract.ERC20ConflictCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ERC20Conflict *ERC20ConflictRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ERC20Conflict.Contract.ERC20ConflictTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ERC20Conflict *ERC20ConflictRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ERC20Conflict.Contract.ERC20ConflictTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ERC20Conflict *ERC20ConflictCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ERC20Conflict.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ERC20Conflict *ERC20ConflictTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ERC20Conflict.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ERC20Conflict *ERC20ConflictTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ERC20Conflict.Contract.contract.Transact(opts, method, params...)
}

// DEFAULTBALANCE is a free data retrieval call binding the contract method 0x70f35735.
//
// Solidity: function DEFAULT_BALANCE() view returns(uint256)
func (_ERC20Conflict *ERC20ConflictCaller) DEFAULTBALANCE(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ERC20Conflict.contract.Call(opts, &out, "DEFAULT_BALANCE")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// DEFAULTBALANCE is a free data retrieval call binding the contract method 0x70f35735.
//
// Solidity: function DEFAULT_BALANCE() view returns(uint256)
func (_ERC20Conflict *ERC20ConflictSession) DEFAULTBALANCE() (*big.Int, error) {
	return _ERC20Conflict.Contract.DEFAULTBALANCE(&_ERC20Conflict.CallOpts)
}

// DEFAULTBALANCE is a free data retrieval call binding the contract method 0x70f35735.
//
// Solidity: function DEFAULT_BALANCE() view returns(uint256)
func (_ERC20Conflict *ERC20ConflictCallerSession) DEFAULTBALANCE() (*big.Int, error) {
	return _ERC20Conflict.Contract.DEFAULTBALANCE(&_ERC20Conflict.CallOpts)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_ERC20Conflict *ERC20ConflictCaller) Allowance(opts *bind.CallOpts, owner common.Address, spender common.Address) (*big.Int, error) {
	var out []interface{}
	err := _ERC20Conflict.contract.Call(opts, &out, "allowance", owner, spender)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_ERC20Conflict *ERC20ConflictSession) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _ERC20Conflict.Contract.Allowance(&_ERC20Conflict.CallOpts, owner, spender)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_ERC20Conflict *ERC20ConflictCallerSession) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _ERC20Conflict.Contract.Allowance(&_ERC20Conflict.CallOpts, owner, spender)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_ERC20Conflict *ERC20ConflictCaller) BalanceOf(opts *bind.CallOpts, account common.Address) (*big.Int, error) {
	var out []interface{}
	err := _ERC20Conflict.contract.Call(opts, &out, "balanceOf", account)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_ERC20Conflict *ERC20ConflictSession) BalanceOf(account common.Address) (*big.Int, error) {
	return _ERC20Conflict.Contract.BalanceOf(&_ERC20Conflict.CallOpts, account)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_ERC20Conflict *ERC20ConflictCallerSession) BalanceOf(account common.Address) (*big.Int, error) {
	return _ERC20Conflict.Contract.BalanceOf(&_ERC20Conflict.CallOpts, account)
}

// Counter is a free data retrieval call binding the contract method 0x61bc221a.
//
// Solidity: function counter() view returns(uint256)
func (_ERC20Conflict *ERC20ConflictCaller) Counter(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ERC20Conflict.contract.Call(opts, &out, "counter")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Counter is a free data retrieval call binding the contract method 0x61bc221a.
//
// Solidity: function counter() view returns(uint256)
func (_ERC20Conflict *ERC20ConflictSession) Counter() (*big.Int, error) {
	return _ERC20Conflict.Contract.Counter(&_ERC20Conflict.CallOpts)
}

// Counter is a free data retrieval call binding the contract method 0x61bc221a.
//
// Solidity: function counter() view returns(uint256)
func (_ERC20Conflict *ERC20ConflictCallerSession) Counter() (*big.Int, error) {
	return _ERC20Conflict.Contract.Counter(&_ERC20Conflict.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_ERC20Conflict *ERC20ConflictCaller) Decimals(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _ERC20Conflict.contract.Call(opts, &out, "decimals")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_ERC20Conflict *ERC20ConflictSession) Decimals() (uint8, error) {
	return _ERC20Conflict.Contract.Decimals(&_ERC20Conflict.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_ERC20Conflict *ERC20ConflictCallerSession) Decimals() (uint8, error) {
	return _ERC20Conflict.Contract.Decimals(&_ERC20Conflict.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_ERC20Conflict *ERC20ConflictCaller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _ERC20Conflict.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_ERC20Conflict *ERC20ConflictSession) Name() (string, error) {
	return _ERC20Conflict.Contract.Name(&_ERC20Conflict.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_ERC20Conflict *ERC20ConflictCallerSession) Name() (string, error) {
	return _ERC20Conflict.Contract.Name(&_ERC20Conflict.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_ERC20Conflict *ERC20ConflictCaller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _ERC20Conflict.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_ERC20Conflict *ERC20ConflictSession) Symbol() (string, error) {
	return _ERC20Conflict.Contract.Symbol(&_ERC20Conflict.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_ERC20Conflict *ERC20ConflictCallerSession) Symbol() (string, error) {
	return _ERC20Conflict.Contract.Symbol(&_ERC20Conflict.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_ERC20Conflict *ERC20ConflictCaller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ERC20Conflict.contract.Call(opts, &out, "totalSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_ERC20Conflict *ERC20ConflictSession) TotalSupply() (*big.Int, error) {
	return _ERC20Conflict.Contract.TotalSupply(&_ERC20Conflict.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_ERC20Conflict *ERC20ConflictCallerSession) TotalSupply() (*big.Int, error) {
	return _ERC20Conflict.Contract.TotalSupply(&_ERC20Conflict.CallOpts)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_ERC20Conflict *ERC20ConflictTransactor) Approve(opts *bind.TransactOpts, spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _ERC20Conflict.contract.Transact(opts, "approve", spender, amount)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_ERC20Conflict *ERC20ConflictSession) Approve(spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _ERC20Conflict.Contract.Approve(&_ERC20Conflict.TransactOpts, spender, amount)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_ERC20Conflict *ERC20ConflictTransactorSession) Approve(spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _ERC20Conflict.Contract.Approve(&_ERC20Conflict.TransactOpts, spender, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address recipient, uint256 amount) returns(bool)
func (_ERC20Conflict *ERC20ConflictTransactor) Transfer(opts *bind.TransactOpts, recipient common.Address, amount *big.Int) (*types.Transaction, error) {
	return _ERC20Conflict.contract.Transact(opts, "transfer", recipient, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address recipient, uint256 amount) returns(bool)
func (_ERC20Conflict *ERC20ConflictSession) Transfer(recipient common.Address, amount *big.Int) (*types.Transaction, error) {
	return _ERC20Conflict.Contract.Transfer(&_ERC20Conflict.TransactOpts, recipient, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address recipient, uint256 amount) returns(bool)
func (_ERC20Conflict *ERC20ConflictTransactorSession) Transfer(recipient common.Address, amount *big.Int) (*types.Transaction, error) {
	return _ERC20Conflict.Contract.Transfer(&_ERC20Conflict.TransactOpts, recipient, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address sender, address recipient, uint256 amount) returns(bool)
func (_ERC20Conflict *ERC20ConflictTransactor) TransferFrom(opts *bind.TransactOpts, sender common.Address, recipient common.Address, amount *big.Int) (*types.Transaction, error) {
	return _ERC20Conflict.contract.Transact(opts, "transferFrom", sender, recipient, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address sender, address recipient, uint256 amount) returns(bool)
func (_ERC20Conflict *ERC20ConflictSession) TransferFrom(sender common.Address, recipient common.Address, amount *big.Int) (*types.Transaction, error) {
	return _ERC20Conflict.Contract.TransferFrom(&_ERC20Conflict.TransactOpts, sender, recipient, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address sender, address recipient, uint256 amount) returns(bool)
func (_ERC20Conflict *ERC20ConflictTransactorSession) TransferFrom(sender common.Address, recipient common.Address, amount *big.Int) (*types.Transaction, error) {
	return _ERC20Conflict.Contract.TransferFrom(&_ERC20Conflict.TransactOpts, sender, recipient, amount)
}

// ERC20ConflictApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the ERC20Conflict contract.
type ERC20ConflictApprovalIterator struct {
	Event *ERC20ConflictApproval // Event containing the contract specifics and raw log

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
func (it *ERC20ConflictApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ERC20ConflictApproval)
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
		it.Event = new(ERC20ConflictApproval)
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
func (it *ERC20ConflictApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ERC20ConflictApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ERC20ConflictApproval represents a Approval event raised by the ERC20Conflict contract.
type ERC20ConflictApproval struct {
	Owner   common.Address
	Spender common.Address
	Value   *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_ERC20Conflict *ERC20ConflictFilterer) FilterApproval(opts *bind.FilterOpts, owner []common.Address, spender []common.Address) (*ERC20ConflictApprovalIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _ERC20Conflict.contract.FilterLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return &ERC20ConflictApprovalIterator{contract: _ERC20Conflict.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_ERC20Conflict *ERC20ConflictFilterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *ERC20ConflictApproval, owner []common.Address, spender []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _ERC20Conflict.contract.WatchLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ERC20ConflictApproval)
				if err := _ERC20Conflict.contract.UnpackLog(event, "Approval", log); err != nil {
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

// ParseApproval is a log parse operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_ERC20Conflict *ERC20ConflictFilterer) ParseApproval(log types.Log) (*ERC20ConflictApproval, error) {
	event := new(ERC20ConflictApproval)
	if err := _ERC20Conflict.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ERC20ConflictTransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the ERC20Conflict contract.
type ERC20ConflictTransferIterator struct {
	Event *ERC20ConflictTransfer // Event containing the contract specifics and raw log

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
func (it *ERC20ConflictTransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ERC20ConflictTransfer)
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
		it.Event = new(ERC20ConflictTransfer)
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
func (it *ERC20ConflictTransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ERC20ConflictTransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ERC20ConflictTransfer represents a Transfer event raised by the ERC20Conflict contract.
type ERC20ConflictTransfer struct {
	From  common.Address
	To    common.Address
	Value *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_ERC20Conflict *ERC20ConflictFilterer) FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*ERC20ConflictTransferIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _ERC20Conflict.contract.FilterLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &ERC20ConflictTransferIterator{contract: _ERC20Conflict.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_ERC20Conflict *ERC20ConflictFilterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *ERC20ConflictTransfer, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _ERC20Conflict.contract.WatchLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ERC20ConflictTransfer)
				if err := _ERC20Conflict.contract.UnpackLog(event, "Transfer", log); err != nil {
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

// ParseTransfer is a log parse operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_ERC20Conflict *ERC20ConflictFilterer) ParseTransfer(log types.Log) (*ERC20ConflictTransfer, error) {
	event := new(ERC20ConflictTransfer)
	if err := _ERC20Conflict.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
