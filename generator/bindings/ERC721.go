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

// ERC721MetaData contains all meta data concerning the ERC721 contract.
var ERC721MetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"approved\",\"type\":\"bool\"}],\"name\":\"ApprovalForAll\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"}],\"name\":\"burn\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"}],\"name\":\"getApproved\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"isApprovedForAll\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"}],\"name\":\"mint\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"}],\"name\":\"ownerOf\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"}],\"name\":\"safeTransferFrom\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"safeTransferFrom\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"operator\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"approved\",\"type\":\"bool\"}],\"name\":\"setApprovalForAll\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"interfaceId\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"id\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b50610d9e8061001c5f395ff3fe608060405234801561000f575f5ffd5b50600436106100b1575f3560e01c806342966c681161006e57806342966c68146101565780636352211e1461016957806370a082311461017c578063a22cb4651461019d578063b88d4fde146101b0578063e985e9c5146101c3575f5ffd5b806301ffc9a7146100b5578063081812fc146100dd578063095ea7b31461010857806323b872dd1461011d57806340c10f191461013057806342842e0e14610143575b5f5ffd5b6100c86100c3366004610ae0565b6101f0565b60405190151581526020015b60405180910390f35b6100f06100eb366004610b02565b610226565b6040516001600160a01b0390911681526020016100d4565b61011b610116366004610b2f565b61029f565b005b61011b61012b366004610b57565b61037e565b61011b61013e366004610b2f565b610530565b61011b610151366004610b57565b61053e565b61011b610164366004610b02565b610632565b6100f0610177366004610b02565b61068f565b61018f61018a366004610b91565b6106ee565b6040519081526020016100d4565b61011b6101ab366004610baa565b610757565b61011b6101be366004610be3565b6107c2565b6100c86101d1366004610c78565b600360209081525f928352604080842090915290825290205460ff1681565b5f6001600160e01b031982166380ac58cd60e01b148061022057506001600160e01b031982166301ffc9a760e01b145b92915050565b5f818152602081905260408120546001600160a01b03166102845760405162461bcd60e51b81526020600482015260136024820152721d1bdad95b88191bd95cdb89dd08195e1a5cdd606a1b60448201526064015b60405180910390fd5b505f908152600260205260409020546001600160a01b031690565b5f818152602081905260409020546001600160a01b0316338114806102e657506001600160a01b0381165f90815260036020908152604080832033845290915290205460ff165b6103235760405162461bcd60e51b815260206004820152600e60248201526d1b9bdd08185d5d1a1bdc9a5e995960921b604482015260640161027b565b5f8281526002602052604080822080546001600160a01b0319166001600160a01b0387811691821790925591518593918516917f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b92591a4505050565b5f818152602081905260409020546001600160a01b038481169116146103d65760405162461bcd60e51b815260206004820152600d60248201526c333937b690109e9037bbb732b960991b604482015260640161027b565b6001600160a01b03821661042c5760405162461bcd60e51b815260206004820152601860248201527f7472616e7366657220746f207a65726f20616464726573730000000000000000604482015260640161027b565b6104378333836108a7565b6104745760405162461bcd60e51b815260206004820152600e60248201526d1b9bdd08185d5d1a1bdc9a5e995960921b604482015260640161027b565b6001600160a01b0383165f90815260016020526040812080549161049783610cbd565b90915550506001600160a01b0382165f9081526001602052604081208054916104bf83610cd2565b90915550505f8181526020818152604080832080546001600160a01b038088166001600160a01b031992831681179093556002909452828520805490911690559051849391928716917fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef91a4505050565b61053a8282610917565b5050565b61054983838361037e565b6001600160a01b0382163b15806105ee5750604051630a85bd0160e11b8082523360048301526001600160a01b03858116602484015260448301849052608060648401525f608484015290919084169063150b7a029060a4016020604051808303815f875af11580156105be573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906105e29190610cea565b6001600160e01b031916145b61062d5760405162461bcd60e51b815260206004820152601060248201526f1d5b9cd85999481c9958da5c1a595b9d60821b604482015260640161027b565b505050565b5f818152602081905260409020546001600160a01b031633146106835760405162461bcd60e51b81526020600482015260096024820152683737ba1037bbb732b960b91b604482015260640161027b565b61068c816109e4565b50565b5f818152602081905260409020546001600160a01b0316806106e95760405162461bcd60e51b81526020600482015260136024820152721d1bdad95b88191bd95cdb89dd08195e1a5cdd606a1b604482015260640161027b565b919050565b5f6001600160a01b03821661073c5760405162461bcd60e51b81526020600482015260146024820152736f776e6572203d207a65726f206164647265737360601b604482015260640161027b565b506001600160a01b03165f9081526001602052604090205490565b335f8181526003602090815260408083206001600160a01b03871680855290835292819020805460ff191686151590811790915590519081529192917f17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31910160405180910390a35050565b6107cd85858561037e565b6001600160a01b0384163b15806108615750604051630a85bd0160e11b808252906001600160a01b0386169063150b7a02906108159033908a90899089908990600401610d05565b6020604051808303815f875af1158015610831573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906108559190610cea565b6001600160e01b031916145b6108a05760405162461bcd60e51b815260206004820152601060248201526f1d5b9cd85999481c9958da5c1a595b9d60821b604482015260640161027b565b5050505050565b5f836001600160a01b0316836001600160a01b031614806108ec57506001600160a01b038085165f9081526003602090815260408083209387168352929052205460ff165b8061090f57505f828152600260205260409020546001600160a01b038481169116145b949350505050565b6001600160a01b0382166109645760405162461bcd60e51b81526020600482015260146024820152736d696e7420746f207a65726f206164647265737360601b604482015260640161027b565b6001600160a01b0382165f90815260016020526040812080549161098783610cd2565b90915550505f8181526020819052604080822080546001600160a01b0319166001600160a01b03861690811790915590518392907fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef908290a45050565b5f818152602081905260409020546001600160a01b031680610a355760405162461bcd60e51b815260206004820152600a6024820152691b9bdd081b5a5b9d195960b21b604482015260640161027b565b6001600160a01b0381165f9081526001602081905260408220805491929091610a5f908490610d55565b90915550505f8281526020818152604080832080546001600160a01b031990811690915560029092528083208054909216909155518391906001600160a01b038416907fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef908390a45050565b6001600160e01b03198116811461068c575f5ffd5b5f60208284031215610af0575f5ffd5b8135610afb81610acb565b9392505050565b5f60208284031215610b12575f5ffd5b5035919050565b80356001600160a01b03811681146106e9575f5ffd5b5f5f60408385031215610b40575f5ffd5b610b4983610b19565b946020939093013593505050565b5f5f5f60608486031215610b69575f5ffd5b610b7284610b19565b9250610b8060208501610b19565b929592945050506040919091013590565b5f60208284031215610ba1575f5ffd5b610afb82610b19565b5f5f60408385031215610bbb575f5ffd5b610bc483610b19565b915060208301358015158114610bd8575f5ffd5b809150509250929050565b5f5f5f5f5f60808688031215610bf7575f5ffd5b610c0086610b19565b9450610c0e60208701610b19565b935060408601359250606086013567ffffffffffffffff811115610c30575f5ffd5b8601601f81018813610c40575f5ffd5b803567ffffffffffffffff811115610c56575f5ffd5b886020828401011115610c67575f5ffd5b959894975092955050506020019190565b5f5f60408385031215610c89575f5ffd5b610c9283610b19565b9150610ca060208401610b19565b90509250929050565b634e487b7160e01b5f52601160045260245ffd5b5f81610ccb57610ccb610ca9565b505f190190565b5f60018201610ce357610ce3610ca9565b5060010190565b5f60208284031215610cfa575f5ffd5b8151610afb81610acb565b6001600160a01b03868116825285166020820152604081018490526080606082018190528101829052818360a08301375f81830160a090810191909152601f909201601f19160101949350505050565b8181038181111561022057610220610ca956fea2646970667358221220dae06bc6773cbc1661268f8a0a3047ee3121216e9d30d22c16c0df65afece97064736f6c634300081b0033",
}

// ERC721ABI is the input ABI used to generate the binding from.
// Deprecated: Use ERC721MetaData.ABI instead.
var ERC721ABI = ERC721MetaData.ABI

// ERC721Bin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ERC721MetaData.Bin instead.
var ERC721Bin = ERC721MetaData.Bin

// DeployERC721 deploys a new Ethereum contract, binding an instance of ERC721 to it.
func DeployERC721(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *ERC721, error) {
	parsed, err := ERC721MetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ERC721Bin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ERC721{ERC721Caller: ERC721Caller{contract: contract}, ERC721Transactor: ERC721Transactor{contract: contract}, ERC721Filterer: ERC721Filterer{contract: contract}}, nil
}

// ERC721 is an auto generated Go binding around an Ethereum contract.
type ERC721 struct {
	ERC721Caller     // Read-only binding to the contract
	ERC721Transactor // Write-only binding to the contract
	ERC721Filterer   // Log filterer for contract events
}

// ERC721Caller is an auto generated read-only Go binding around an Ethereum contract.
type ERC721Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ERC721Transactor is an auto generated write-only Go binding around an Ethereum contract.
type ERC721Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ERC721Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ERC721Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ERC721Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ERC721Session struct {
	Contract     *ERC721           // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ERC721CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ERC721CallerSession struct {
	Contract *ERC721Caller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// ERC721TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ERC721TransactorSession struct {
	Contract     *ERC721Transactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ERC721Raw is an auto generated low-level Go binding around an Ethereum contract.
type ERC721Raw struct {
	Contract *ERC721 // Generic contract binding to access the raw methods on
}

// ERC721CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ERC721CallerRaw struct {
	Contract *ERC721Caller // Generic read-only contract binding to access the raw methods on
}

// ERC721TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ERC721TransactorRaw struct {
	Contract *ERC721Transactor // Generic write-only contract binding to access the raw methods on
}

// NewERC721 creates a new instance of ERC721, bound to a specific deployed contract.
func NewERC721(address common.Address, backend bind.ContractBackend) (*ERC721, error) {
	contract, err := bindERC721(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ERC721{ERC721Caller: ERC721Caller{contract: contract}, ERC721Transactor: ERC721Transactor{contract: contract}, ERC721Filterer: ERC721Filterer{contract: contract}}, nil
}

// NewERC721Caller creates a new read-only instance of ERC721, bound to a specific deployed contract.
func NewERC721Caller(address common.Address, caller bind.ContractCaller) (*ERC721Caller, error) {
	contract, err := bindERC721(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ERC721Caller{contract: contract}, nil
}

// NewERC721Transactor creates a new write-only instance of ERC721, bound to a specific deployed contract.
func NewERC721Transactor(address common.Address, transactor bind.ContractTransactor) (*ERC721Transactor, error) {
	contract, err := bindERC721(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ERC721Transactor{contract: contract}, nil
}

// NewERC721Filterer creates a new log filterer instance of ERC721, bound to a specific deployed contract.
func NewERC721Filterer(address common.Address, filterer bind.ContractFilterer) (*ERC721Filterer, error) {
	contract, err := bindERC721(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ERC721Filterer{contract: contract}, nil
}

// bindERC721 binds a generic wrapper to an already deployed contract.
func bindERC721(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ERC721MetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ERC721 *ERC721Raw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ERC721.Contract.ERC721Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ERC721 *ERC721Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ERC721.Contract.ERC721Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ERC721 *ERC721Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ERC721.Contract.ERC721Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ERC721 *ERC721CallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ERC721.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ERC721 *ERC721TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ERC721.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ERC721 *ERC721TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ERC721.Contract.contract.Transact(opts, method, params...)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address owner) view returns(uint256)
func (_ERC721 *ERC721Caller) BalanceOf(opts *bind.CallOpts, owner common.Address) (*big.Int, error) {
	var out []interface{}
	err := _ERC721.contract.Call(opts, &out, "balanceOf", owner)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address owner) view returns(uint256)
func (_ERC721 *ERC721Session) BalanceOf(owner common.Address) (*big.Int, error) {
	return _ERC721.Contract.BalanceOf(&_ERC721.CallOpts, owner)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address owner) view returns(uint256)
func (_ERC721 *ERC721CallerSession) BalanceOf(owner common.Address) (*big.Int, error) {
	return _ERC721.Contract.BalanceOf(&_ERC721.CallOpts, owner)
}

// GetApproved is a free data retrieval call binding the contract method 0x081812fc.
//
// Solidity: function getApproved(uint256 id) view returns(address)
func (_ERC721 *ERC721Caller) GetApproved(opts *bind.CallOpts, id *big.Int) (common.Address, error) {
	var out []interface{}
	err := _ERC721.contract.Call(opts, &out, "getApproved", id)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetApproved is a free data retrieval call binding the contract method 0x081812fc.
//
// Solidity: function getApproved(uint256 id) view returns(address)
func (_ERC721 *ERC721Session) GetApproved(id *big.Int) (common.Address, error) {
	return _ERC721.Contract.GetApproved(&_ERC721.CallOpts, id)
}

// GetApproved is a free data retrieval call binding the contract method 0x081812fc.
//
// Solidity: function getApproved(uint256 id) view returns(address)
func (_ERC721 *ERC721CallerSession) GetApproved(id *big.Int) (common.Address, error) {
	return _ERC721.Contract.GetApproved(&_ERC721.CallOpts, id)
}

// IsApprovedForAll is a free data retrieval call binding the contract method 0xe985e9c5.
//
// Solidity: function isApprovedForAll(address , address ) view returns(bool)
func (_ERC721 *ERC721Caller) IsApprovedForAll(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address) (bool, error) {
	var out []interface{}
	err := _ERC721.contract.Call(opts, &out, "isApprovedForAll", arg0, arg1)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsApprovedForAll is a free data retrieval call binding the contract method 0xe985e9c5.
//
// Solidity: function isApprovedForAll(address , address ) view returns(bool)
func (_ERC721 *ERC721Session) IsApprovedForAll(arg0 common.Address, arg1 common.Address) (bool, error) {
	return _ERC721.Contract.IsApprovedForAll(&_ERC721.CallOpts, arg0, arg1)
}

// IsApprovedForAll is a free data retrieval call binding the contract method 0xe985e9c5.
//
// Solidity: function isApprovedForAll(address , address ) view returns(bool)
func (_ERC721 *ERC721CallerSession) IsApprovedForAll(arg0 common.Address, arg1 common.Address) (bool, error) {
	return _ERC721.Contract.IsApprovedForAll(&_ERC721.CallOpts, arg0, arg1)
}

// OwnerOf is a free data retrieval call binding the contract method 0x6352211e.
//
// Solidity: function ownerOf(uint256 id) view returns(address owner)
func (_ERC721 *ERC721Caller) OwnerOf(opts *bind.CallOpts, id *big.Int) (common.Address, error) {
	var out []interface{}
	err := _ERC721.contract.Call(opts, &out, "ownerOf", id)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OwnerOf is a free data retrieval call binding the contract method 0x6352211e.
//
// Solidity: function ownerOf(uint256 id) view returns(address owner)
func (_ERC721 *ERC721Session) OwnerOf(id *big.Int) (common.Address, error) {
	return _ERC721.Contract.OwnerOf(&_ERC721.CallOpts, id)
}

// OwnerOf is a free data retrieval call binding the contract method 0x6352211e.
//
// Solidity: function ownerOf(uint256 id) view returns(address owner)
func (_ERC721 *ERC721CallerSession) OwnerOf(id *big.Int) (common.Address, error) {
	return _ERC721.Contract.OwnerOf(&_ERC721.CallOpts, id)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) pure returns(bool)
func (_ERC721 *ERC721Caller) SupportsInterface(opts *bind.CallOpts, interfaceId [4]byte) (bool, error) {
	var out []interface{}
	err := _ERC721.contract.Call(opts, &out, "supportsInterface", interfaceId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) pure returns(bool)
func (_ERC721 *ERC721Session) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _ERC721.Contract.SupportsInterface(&_ERC721.CallOpts, interfaceId)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) pure returns(bool)
func (_ERC721 *ERC721CallerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _ERC721.Contract.SupportsInterface(&_ERC721.CallOpts, interfaceId)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 id) returns()
func (_ERC721 *ERC721Transactor) Approve(opts *bind.TransactOpts, spender common.Address, id *big.Int) (*types.Transaction, error) {
	return _ERC721.contract.Transact(opts, "approve", spender, id)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 id) returns()
func (_ERC721 *ERC721Session) Approve(spender common.Address, id *big.Int) (*types.Transaction, error) {
	return _ERC721.Contract.Approve(&_ERC721.TransactOpts, spender, id)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 id) returns()
func (_ERC721 *ERC721TransactorSession) Approve(spender common.Address, id *big.Int) (*types.Transaction, error) {
	return _ERC721.Contract.Approve(&_ERC721.TransactOpts, spender, id)
}

// Burn is a paid mutator transaction binding the contract method 0x42966c68.
//
// Solidity: function burn(uint256 id) returns()
func (_ERC721 *ERC721Transactor) Burn(opts *bind.TransactOpts, id *big.Int) (*types.Transaction, error) {
	return _ERC721.contract.Transact(opts, "burn", id)
}

// Burn is a paid mutator transaction binding the contract method 0x42966c68.
//
// Solidity: function burn(uint256 id) returns()
func (_ERC721 *ERC721Session) Burn(id *big.Int) (*types.Transaction, error) {
	return _ERC721.Contract.Burn(&_ERC721.TransactOpts, id)
}

// Burn is a paid mutator transaction binding the contract method 0x42966c68.
//
// Solidity: function burn(uint256 id) returns()
func (_ERC721 *ERC721TransactorSession) Burn(id *big.Int) (*types.Transaction, error) {
	return _ERC721.Contract.Burn(&_ERC721.TransactOpts, id)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address to, uint256 id) returns()
func (_ERC721 *ERC721Transactor) Mint(opts *bind.TransactOpts, to common.Address, id *big.Int) (*types.Transaction, error) {
	return _ERC721.contract.Transact(opts, "mint", to, id)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address to, uint256 id) returns()
func (_ERC721 *ERC721Session) Mint(to common.Address, id *big.Int) (*types.Transaction, error) {
	return _ERC721.Contract.Mint(&_ERC721.TransactOpts, to, id)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address to, uint256 id) returns()
func (_ERC721 *ERC721TransactorSession) Mint(to common.Address, id *big.Int) (*types.Transaction, error) {
	return _ERC721.Contract.Mint(&_ERC721.TransactOpts, to, id)
}

// SafeTransferFrom is a paid mutator transaction binding the contract method 0x42842e0e.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 id) returns()
func (_ERC721 *ERC721Transactor) SafeTransferFrom(opts *bind.TransactOpts, from common.Address, to common.Address, id *big.Int) (*types.Transaction, error) {
	return _ERC721.contract.Transact(opts, "safeTransferFrom", from, to, id)
}

// SafeTransferFrom is a paid mutator transaction binding the contract method 0x42842e0e.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 id) returns()
func (_ERC721 *ERC721Session) SafeTransferFrom(from common.Address, to common.Address, id *big.Int) (*types.Transaction, error) {
	return _ERC721.Contract.SafeTransferFrom(&_ERC721.TransactOpts, from, to, id)
}

// SafeTransferFrom is a paid mutator transaction binding the contract method 0x42842e0e.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 id) returns()
func (_ERC721 *ERC721TransactorSession) SafeTransferFrom(from common.Address, to common.Address, id *big.Int) (*types.Transaction, error) {
	return _ERC721.Contract.SafeTransferFrom(&_ERC721.TransactOpts, from, to, id)
}

// SafeTransferFrom0 is a paid mutator transaction binding the contract method 0xb88d4fde.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 id, bytes data) returns()
func (_ERC721 *ERC721Transactor) SafeTransferFrom0(opts *bind.TransactOpts, from common.Address, to common.Address, id *big.Int, data []byte) (*types.Transaction, error) {
	return _ERC721.contract.Transact(opts, "safeTransferFrom0", from, to, id, data)
}

// SafeTransferFrom0 is a paid mutator transaction binding the contract method 0xb88d4fde.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 id, bytes data) returns()
func (_ERC721 *ERC721Session) SafeTransferFrom0(from common.Address, to common.Address, id *big.Int, data []byte) (*types.Transaction, error) {
	return _ERC721.Contract.SafeTransferFrom0(&_ERC721.TransactOpts, from, to, id, data)
}

// SafeTransferFrom0 is a paid mutator transaction binding the contract method 0xb88d4fde.
//
// Solidity: function safeTransferFrom(address from, address to, uint256 id, bytes data) returns()
func (_ERC721 *ERC721TransactorSession) SafeTransferFrom0(from common.Address, to common.Address, id *big.Int, data []byte) (*types.Transaction, error) {
	return _ERC721.Contract.SafeTransferFrom0(&_ERC721.TransactOpts, from, to, id, data)
}

// SetApprovalForAll is a paid mutator transaction binding the contract method 0xa22cb465.
//
// Solidity: function setApprovalForAll(address operator, bool approved) returns()
func (_ERC721 *ERC721Transactor) SetApprovalForAll(opts *bind.TransactOpts, operator common.Address, approved bool) (*types.Transaction, error) {
	return _ERC721.contract.Transact(opts, "setApprovalForAll", operator, approved)
}

// SetApprovalForAll is a paid mutator transaction binding the contract method 0xa22cb465.
//
// Solidity: function setApprovalForAll(address operator, bool approved) returns()
func (_ERC721 *ERC721Session) SetApprovalForAll(operator common.Address, approved bool) (*types.Transaction, error) {
	return _ERC721.Contract.SetApprovalForAll(&_ERC721.TransactOpts, operator, approved)
}

// SetApprovalForAll is a paid mutator transaction binding the contract method 0xa22cb465.
//
// Solidity: function setApprovalForAll(address operator, bool approved) returns()
func (_ERC721 *ERC721TransactorSession) SetApprovalForAll(operator common.Address, approved bool) (*types.Transaction, error) {
	return _ERC721.Contract.SetApprovalForAll(&_ERC721.TransactOpts, operator, approved)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 id) returns()
func (_ERC721 *ERC721Transactor) TransferFrom(opts *bind.TransactOpts, from common.Address, to common.Address, id *big.Int) (*types.Transaction, error) {
	return _ERC721.contract.Transact(opts, "transferFrom", from, to, id)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 id) returns()
func (_ERC721 *ERC721Session) TransferFrom(from common.Address, to common.Address, id *big.Int) (*types.Transaction, error) {
	return _ERC721.Contract.TransferFrom(&_ERC721.TransactOpts, from, to, id)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 id) returns()
func (_ERC721 *ERC721TransactorSession) TransferFrom(from common.Address, to common.Address, id *big.Int) (*types.Transaction, error) {
	return _ERC721.Contract.TransferFrom(&_ERC721.TransactOpts, from, to, id)
}

// ERC721ApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the ERC721 contract.
type ERC721ApprovalIterator struct {
	Event *ERC721Approval // Event containing the contract specifics and raw log

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
func (it *ERC721ApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ERC721Approval)
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
		it.Event = new(ERC721Approval)
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
func (it *ERC721ApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ERC721ApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ERC721Approval represents a Approval event raised by the ERC721 contract.
type ERC721Approval struct {
	Owner   common.Address
	Spender common.Address
	Id      *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 indexed id)
func (_ERC721 *ERC721Filterer) FilterApproval(opts *bind.FilterOpts, owner []common.Address, spender []common.Address, id []*big.Int) (*ERC721ApprovalIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}
	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _ERC721.contract.FilterLogs(opts, "Approval", ownerRule, spenderRule, idRule)
	if err != nil {
		return nil, err
	}
	return &ERC721ApprovalIterator{contract: _ERC721.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 indexed id)
func (_ERC721 *ERC721Filterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *ERC721Approval, owner []common.Address, spender []common.Address, id []*big.Int) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}
	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _ERC721.contract.WatchLogs(opts, "Approval", ownerRule, spenderRule, idRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ERC721Approval)
				if err := _ERC721.contract.UnpackLog(event, "Approval", log); err != nil {
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
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 indexed id)
func (_ERC721 *ERC721Filterer) ParseApproval(log types.Log) (*ERC721Approval, error) {
	event := new(ERC721Approval)
	if err := _ERC721.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ERC721ApprovalForAllIterator is returned from FilterApprovalForAll and is used to iterate over the raw logs and unpacked data for ApprovalForAll events raised by the ERC721 contract.
type ERC721ApprovalForAllIterator struct {
	Event *ERC721ApprovalForAll // Event containing the contract specifics and raw log

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
func (it *ERC721ApprovalForAllIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ERC721ApprovalForAll)
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
		it.Event = new(ERC721ApprovalForAll)
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
func (it *ERC721ApprovalForAllIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ERC721ApprovalForAllIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ERC721ApprovalForAll represents a ApprovalForAll event raised by the ERC721 contract.
type ERC721ApprovalForAll struct {
	Owner    common.Address
	Operator common.Address
	Approved bool
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterApprovalForAll is a free log retrieval operation binding the contract event 0x17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31.
//
// Solidity: event ApprovalForAll(address indexed owner, address indexed operator, bool approved)
func (_ERC721 *ERC721Filterer) FilterApprovalForAll(opts *bind.FilterOpts, owner []common.Address, operator []common.Address) (*ERC721ApprovalForAllIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}

	logs, sub, err := _ERC721.contract.FilterLogs(opts, "ApprovalForAll", ownerRule, operatorRule)
	if err != nil {
		return nil, err
	}
	return &ERC721ApprovalForAllIterator{contract: _ERC721.contract, event: "ApprovalForAll", logs: logs, sub: sub}, nil
}

// WatchApprovalForAll is a free log subscription operation binding the contract event 0x17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31.
//
// Solidity: event ApprovalForAll(address indexed owner, address indexed operator, bool approved)
func (_ERC721 *ERC721Filterer) WatchApprovalForAll(opts *bind.WatchOpts, sink chan<- *ERC721ApprovalForAll, owner []common.Address, operator []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}

	logs, sub, err := _ERC721.contract.WatchLogs(opts, "ApprovalForAll", ownerRule, operatorRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ERC721ApprovalForAll)
				if err := _ERC721.contract.UnpackLog(event, "ApprovalForAll", log); err != nil {
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

// ParseApprovalForAll is a log parse operation binding the contract event 0x17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31.
//
// Solidity: event ApprovalForAll(address indexed owner, address indexed operator, bool approved)
func (_ERC721 *ERC721Filterer) ParseApprovalForAll(log types.Log) (*ERC721ApprovalForAll, error) {
	event := new(ERC721ApprovalForAll)
	if err := _ERC721.contract.UnpackLog(event, "ApprovalForAll", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ERC721TransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the ERC721 contract.
type ERC721TransferIterator struct {
	Event *ERC721Transfer // Event containing the contract specifics and raw log

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
func (it *ERC721TransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ERC721Transfer)
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
		it.Event = new(ERC721Transfer)
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
func (it *ERC721TransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ERC721TransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ERC721Transfer represents a Transfer event raised by the ERC721 contract.
type ERC721Transfer struct {
	From common.Address
	To   common.Address
	Id   *big.Int
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 indexed id)
func (_ERC721 *ERC721Filterer) FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address, id []*big.Int) (*ERC721TransferIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}
	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _ERC721.contract.FilterLogs(opts, "Transfer", fromRule, toRule, idRule)
	if err != nil {
		return nil, err
	}
	return &ERC721TransferIterator{contract: _ERC721.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 indexed id)
func (_ERC721 *ERC721Filterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *ERC721Transfer, from []common.Address, to []common.Address, id []*big.Int) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}
	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _ERC721.contract.WatchLogs(opts, "Transfer", fromRule, toRule, idRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ERC721Transfer)
				if err := _ERC721.contract.UnpackLog(event, "Transfer", log); err != nil {
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
// Solidity: event Transfer(address indexed from, address indexed to, uint256 indexed id)
func (_ERC721 *ERC721Filterer) ParseTransfer(log types.Log) (*ERC721Transfer, error) {
	event := new(ERC721Transfer)
	if err := _ERC721.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
