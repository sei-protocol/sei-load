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

// DisperseMetaData contains all meta data concerning the Disperse contract.
var DisperseMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"etherAmount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"tokenAmount\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"recipients\",\"type\":\"address[]\"},{\"internalType\":\"uint256[]\",\"name\":\"values\",\"type\":\"uint256[]\"}],\"name\":\"disperseEther\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"recipients\",\"type\":\"address[]\"}],\"name\":\"disperseEtherFixed\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"token\",\"type\":\"address\"},{\"internalType\":\"address[]\",\"name\":\"recipients\",\"type\":\"address[]\"},{\"internalType\":\"uint256[]\",\"name\":\"values\",\"type\":\"uint256[]\"}],\"name\":\"disperseToken\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"token\",\"type\":\"address\"},{\"internalType\":\"address[]\",\"name\":\"recipients\",\"type\":\"address[]\"}],\"name\":\"disperseTokenFixed\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"token\",\"type\":\"address\"},{\"internalType\":\"address[]\",\"name\":\"recipients\",\"type\":\"address[]\"},{\"internalType\":\"uint256[]\",\"name\":\"values\",\"type\":\"uint256[]\"}],\"name\":\"disperseTokenSimple\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"fixedEtherAmount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"fixedTokenAmount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"setFixedEtherAmount\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"setFixedTokenAmount\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b50604051610ae5380380610ae5833981016040819052602b916049565b5f80546001600160a01b03191633179055600191909155600255606a565b5f5f604083850312156059575f5ffd5b505080516020909101519092909150565b610a6e806100775f395ff3fe60806040526004361061009a575f3560e01c80638da5cb5b116100625780638da5cb5b14610138578063c73a2d601461016e578063e63d38ed1461018d578063e904a243146101a0578063f2fde38b146101b5578063fb169f36146101d4575f5ffd5b8063094864231461009e578063107520fe146100c65780631839bcf3146100e757806351ba162c146100fa5780635fcc7d1514610119575b5f5ffd5b3480156100a9575f5ffd5b506100b360015481565b6040519081526020015b60405180910390f35b3480156100d1575f5ffd5b506100e56100e03660046107ab565b6101f3565b005b6100e56100f536600461080a565b61020d565b348015610105575f5ffd5b506100e5610114366004610860565b6102cf565b348015610124575f5ffd5b506100e56101333660046107ab565b6103b7565b348015610143575f5ffd5b505f54610156906001600160a01b031681565b6040516001600160a01b0390911681526020016100bd565b348015610179575f5ffd5b506100e5610188366004610860565b6103d1565b6100e561019b3660046108e2565b610567565b3480156101ab575f5ffd5b506100b360025481565b3480156101c0575f5ffd5b506100e56101cf36600461094e565b610627565b3480156101df575f5ffd5b506100e56101ee366004610970565b61065d565b5f546001600160a01b03163314610208575f5ffd5b600255565b60015461021b9082906109d5565b3414610225575f5ffd5b5f5b8181101561029657828282818110610241576102416109f2565b9050602002016020810190610256919061094e565b6001600160a01b03166108fc60015490811502906040515f60405180830381858888f1935050505015801561028d573d5f5f3e3d5ffd5b50600101610227565b504780156102ca57604051339082156108fc029083905f818181858888f193505050501580156102c8573d5f5f3e3d5ffd5b505b505050565b5f5b838110156103af57856001600160a01b03166323b872dd338787858181106102fb576102fb6109f2565b9050602002016020810190610310919061094e565b868686818110610322576103226109f2565b6040516001600160e01b031960e088901b1681526001600160a01b039586166004820152949093166024850152506020909102013560448201526064016020604051808303815f875af115801561037b573d5f5f3e3d5ffd5b505050506040513d601f19601f8201168201806040525081019061039f9190610a06565b6103a7575f5ffd5b6001016102d1565b505050505050565b5f546001600160a01b031633146103cc575f5ffd5b600155565b5f805b8481101561040a578383828181106103ee576103ee6109f2565b90506020020135826104009190610a25565b91506001016103d4565b506040516323b872dd60e01b8152336004820152306024820152604481018290526001600160a01b038716906323b872dd906064016020604051808303815f875af115801561045b573d5f5f3e3d5ffd5b505050506040513d601f19601f8201168201806040525081019061047f9190610a06565b610487575f5ffd5b5f5b8481101561055e57866001600160a01b031663a9059cbb8787848181106104b2576104b26109f2565b90506020020160208101906104c7919061094e565b8686858181106104d9576104d96109f2565b6040516001600160e01b031960e087901b1681526001600160a01b03909416600485015260200291909101356024830152506044016020604051808303815f875af115801561052a573d5f5f3e3d5ffd5b505050506040513d601f19601f8201168201806040525081019061054e9190610a06565b610556575f5ffd5b600101610489565b50505050505050565b5f5b838110156105ee57848482818110610583576105836109f2565b9050602002016020810190610598919061094e565b6001600160a01b03166108fc8484848181106105b6576105b66109f2565b9050602002013590811502906040515f60405180830381858888f193505050501580156105e5573d5f5f3e3d5ffd5b50600101610569565b5047801561062057604051339082156108fc029083905f818181858888f193505050501580156103af573d5f5f3e3d5ffd5b5050505050565b5f546001600160a01b0316331461063c575f5ffd5b5f80546001600160a01b0319166001600160a01b0392909216919091179055565b6002545f9061066d9083906109d5565b6040516323b872dd60e01b8152336004820152306024820152604481018290529091506001600160a01b038516906323b872dd906064016020604051808303815f875af11580156106c0573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906106e49190610a06565b6106ec575f5ffd5b5f5b8281101561062057846001600160a01b031663a9059cbb858584818110610717576107176109f2565b905060200201602081019061072c919061094e565b6002546040516001600160e01b031960e085901b1681526001600160a01b03909216600483015260248201526044016020604051808303815f875af1158015610777573d5f5f3e3d5ffd5b505050506040513d601f19601f8201168201806040525081019061079b9190610a06565b6107a3575f5ffd5b6001016106ee565b5f602082840312156107bb575f5ffd5b5035919050565b5f5f83601f8401126107d2575f5ffd5b50813567ffffffffffffffff8111156107e9575f5ffd5b6020830191508360208260051b8501011115610803575f5ffd5b9250929050565b5f5f6020838503121561081b575f5ffd5b823567ffffffffffffffff811115610831575f5ffd5b61083d858286016107c2565b90969095509350505050565b6001600160a01b038116811461085d575f5ffd5b50565b5f5f5f5f5f60608688031215610874575f5ffd5b853561087f81610849565b9450602086013567ffffffffffffffff81111561089a575f5ffd5b6108a6888289016107c2565b909550935050604086013567ffffffffffffffff8111156108c5575f5ffd5b6108d1888289016107c2565b969995985093965092949392505050565b5f5f5f5f604085870312156108f5575f5ffd5b843567ffffffffffffffff81111561090b575f5ffd5b610917878288016107c2565b909550935050602085013567ffffffffffffffff811115610936575f5ffd5b610942878288016107c2565b95989497509550505050565b5f6020828403121561095e575f5ffd5b813561096981610849565b9392505050565b5f5f5f60408486031215610982575f5ffd5b833561098d81610849565b9250602084013567ffffffffffffffff8111156109a8575f5ffd5b6109b4868287016107c2565b9497909650939450505050565b634e487b7160e01b5f52601160045260245ffd5b80820281158282048414176109ec576109ec6109c1565b92915050565b634e487b7160e01b5f52603260045260245ffd5b5f60208284031215610a16575f5ffd5b81518015158114610969575f5ffd5b808201808211156109ec576109ec6109c156fea2646970667358221220f1dee86157b5dbc23726804537f4aac7e801a5811a55887970b83f951eabc4ee64736f6c634300081e0033",
}

// DisperseABI is the input ABI used to generate the binding from.
// Deprecated: Use DisperseMetaData.ABI instead.
var DisperseABI = DisperseMetaData.ABI

// DisperseBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use DisperseMetaData.Bin instead.
var DisperseBin = DisperseMetaData.Bin

// DeployDisperse deploys a new Ethereum contract, binding an instance of Disperse to it.
func DeployDisperse(auth *bind.TransactOpts, backend bind.ContractBackend, etherAmount *big.Int, tokenAmount *big.Int) (common.Address, *types.Transaction, *Disperse, error) {
	parsed, err := DisperseMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(DisperseBin), backend, etherAmount, tokenAmount)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Disperse{DisperseCaller: DisperseCaller{contract: contract}, DisperseTransactor: DisperseTransactor{contract: contract}, DisperseFilterer: DisperseFilterer{contract: contract}}, nil
}

// Disperse is an auto generated Go binding around an Ethereum contract.
type Disperse struct {
	DisperseCaller     // Read-only binding to the contract
	DisperseTransactor // Write-only binding to the contract
	DisperseFilterer   // Log filterer for contract events
}

// DisperseCaller is an auto generated read-only Go binding around an Ethereum contract.
type DisperseCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DisperseTransactor is an auto generated write-only Go binding around an Ethereum contract.
type DisperseTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DisperseFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type DisperseFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DisperseSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type DisperseSession struct {
	Contract     *Disperse         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// DisperseCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type DisperseCallerSession struct {
	Contract *DisperseCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// DisperseTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type DisperseTransactorSession struct {
	Contract     *DisperseTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// DisperseRaw is an auto generated low-level Go binding around an Ethereum contract.
type DisperseRaw struct {
	Contract *Disperse // Generic contract binding to access the raw methods on
}

// DisperseCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type DisperseCallerRaw struct {
	Contract *DisperseCaller // Generic read-only contract binding to access the raw methods on
}

// DisperseTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type DisperseTransactorRaw struct {
	Contract *DisperseTransactor // Generic write-only contract binding to access the raw methods on
}

// NewDisperse creates a new instance of Disperse, bound to a specific deployed contract.
func NewDisperse(address common.Address, backend bind.ContractBackend) (*Disperse, error) {
	contract, err := bindDisperse(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Disperse{DisperseCaller: DisperseCaller{contract: contract}, DisperseTransactor: DisperseTransactor{contract: contract}, DisperseFilterer: DisperseFilterer{contract: contract}}, nil
}

// NewDisperseCaller creates a new read-only instance of Disperse, bound to a specific deployed contract.
func NewDisperseCaller(address common.Address, caller bind.ContractCaller) (*DisperseCaller, error) {
	contract, err := bindDisperse(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &DisperseCaller{contract: contract}, nil
}

// NewDisperseTransactor creates a new write-only instance of Disperse, bound to a specific deployed contract.
func NewDisperseTransactor(address common.Address, transactor bind.ContractTransactor) (*DisperseTransactor, error) {
	contract, err := bindDisperse(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &DisperseTransactor{contract: contract}, nil
}

// NewDisperseFilterer creates a new log filterer instance of Disperse, bound to a specific deployed contract.
func NewDisperseFilterer(address common.Address, filterer bind.ContractFilterer) (*DisperseFilterer, error) {
	contract, err := bindDisperse(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &DisperseFilterer{contract: contract}, nil
}

// bindDisperse binds a generic wrapper to an already deployed contract.
func bindDisperse(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := DisperseMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Disperse *DisperseRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Disperse.Contract.DisperseCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Disperse *DisperseRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Disperse.Contract.DisperseTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Disperse *DisperseRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Disperse.Contract.DisperseTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Disperse *DisperseCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Disperse.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Disperse *DisperseTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Disperse.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Disperse *DisperseTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Disperse.Contract.contract.Transact(opts, method, params...)
}

// FixedEtherAmount is a free data retrieval call binding the contract method 0x09486423.
//
// Solidity: function fixedEtherAmount() view returns(uint256)
func (_Disperse *DisperseCaller) FixedEtherAmount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Disperse.contract.Call(opts, &out, "fixedEtherAmount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// FixedEtherAmount is a free data retrieval call binding the contract method 0x09486423.
//
// Solidity: function fixedEtherAmount() view returns(uint256)
func (_Disperse *DisperseSession) FixedEtherAmount() (*big.Int, error) {
	return _Disperse.Contract.FixedEtherAmount(&_Disperse.CallOpts)
}

// FixedEtherAmount is a free data retrieval call binding the contract method 0x09486423.
//
// Solidity: function fixedEtherAmount() view returns(uint256)
func (_Disperse *DisperseCallerSession) FixedEtherAmount() (*big.Int, error) {
	return _Disperse.Contract.FixedEtherAmount(&_Disperse.CallOpts)
}

// FixedTokenAmount is a free data retrieval call binding the contract method 0xe904a243.
//
// Solidity: function fixedTokenAmount() view returns(uint256)
func (_Disperse *DisperseCaller) FixedTokenAmount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Disperse.contract.Call(opts, &out, "fixedTokenAmount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// FixedTokenAmount is a free data retrieval call binding the contract method 0xe904a243.
//
// Solidity: function fixedTokenAmount() view returns(uint256)
func (_Disperse *DisperseSession) FixedTokenAmount() (*big.Int, error) {
	return _Disperse.Contract.FixedTokenAmount(&_Disperse.CallOpts)
}

// FixedTokenAmount is a free data retrieval call binding the contract method 0xe904a243.
//
// Solidity: function fixedTokenAmount() view returns(uint256)
func (_Disperse *DisperseCallerSession) FixedTokenAmount() (*big.Int, error) {
	return _Disperse.Contract.FixedTokenAmount(&_Disperse.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Disperse *DisperseCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Disperse.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Disperse *DisperseSession) Owner() (common.Address, error) {
	return _Disperse.Contract.Owner(&_Disperse.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Disperse *DisperseCallerSession) Owner() (common.Address, error) {
	return _Disperse.Contract.Owner(&_Disperse.CallOpts)
}

// DisperseEther is a paid mutator transaction binding the contract method 0xe63d38ed.
//
// Solidity: function disperseEther(address[] recipients, uint256[] values) payable returns()
func (_Disperse *DisperseTransactor) DisperseEther(opts *bind.TransactOpts, recipients []common.Address, values []*big.Int) (*types.Transaction, error) {
	return _Disperse.contract.Transact(opts, "disperseEther", recipients, values)
}

// DisperseEther is a paid mutator transaction binding the contract method 0xe63d38ed.
//
// Solidity: function disperseEther(address[] recipients, uint256[] values) payable returns()
func (_Disperse *DisperseSession) DisperseEther(recipients []common.Address, values []*big.Int) (*types.Transaction, error) {
	return _Disperse.Contract.DisperseEther(&_Disperse.TransactOpts, recipients, values)
}

// DisperseEther is a paid mutator transaction binding the contract method 0xe63d38ed.
//
// Solidity: function disperseEther(address[] recipients, uint256[] values) payable returns()
func (_Disperse *DisperseTransactorSession) DisperseEther(recipients []common.Address, values []*big.Int) (*types.Transaction, error) {
	return _Disperse.Contract.DisperseEther(&_Disperse.TransactOpts, recipients, values)
}

// DisperseEtherFixed is a paid mutator transaction binding the contract method 0x1839bcf3.
//
// Solidity: function disperseEtherFixed(address[] recipients) payable returns()
func (_Disperse *DisperseTransactor) DisperseEtherFixed(opts *bind.TransactOpts, recipients []common.Address) (*types.Transaction, error) {
	return _Disperse.contract.Transact(opts, "disperseEtherFixed", recipients)
}

// DisperseEtherFixed is a paid mutator transaction binding the contract method 0x1839bcf3.
//
// Solidity: function disperseEtherFixed(address[] recipients) payable returns()
func (_Disperse *DisperseSession) DisperseEtherFixed(recipients []common.Address) (*types.Transaction, error) {
	return _Disperse.Contract.DisperseEtherFixed(&_Disperse.TransactOpts, recipients)
}

// DisperseEtherFixed is a paid mutator transaction binding the contract method 0x1839bcf3.
//
// Solidity: function disperseEtherFixed(address[] recipients) payable returns()
func (_Disperse *DisperseTransactorSession) DisperseEtherFixed(recipients []common.Address) (*types.Transaction, error) {
	return _Disperse.Contract.DisperseEtherFixed(&_Disperse.TransactOpts, recipients)
}

// DisperseToken is a paid mutator transaction binding the contract method 0xc73a2d60.
//
// Solidity: function disperseToken(address token, address[] recipients, uint256[] values) returns()
func (_Disperse *DisperseTransactor) DisperseToken(opts *bind.TransactOpts, token common.Address, recipients []common.Address, values []*big.Int) (*types.Transaction, error) {
	return _Disperse.contract.Transact(opts, "disperseToken", token, recipients, values)
}

// DisperseToken is a paid mutator transaction binding the contract method 0xc73a2d60.
//
// Solidity: function disperseToken(address token, address[] recipients, uint256[] values) returns()
func (_Disperse *DisperseSession) DisperseToken(token common.Address, recipients []common.Address, values []*big.Int) (*types.Transaction, error) {
	return _Disperse.Contract.DisperseToken(&_Disperse.TransactOpts, token, recipients, values)
}

// DisperseToken is a paid mutator transaction binding the contract method 0xc73a2d60.
//
// Solidity: function disperseToken(address token, address[] recipients, uint256[] values) returns()
func (_Disperse *DisperseTransactorSession) DisperseToken(token common.Address, recipients []common.Address, values []*big.Int) (*types.Transaction, error) {
	return _Disperse.Contract.DisperseToken(&_Disperse.TransactOpts, token, recipients, values)
}

// DisperseTokenFixed is a paid mutator transaction binding the contract method 0xfb169f36.
//
// Solidity: function disperseTokenFixed(address token, address[] recipients) returns()
func (_Disperse *DisperseTransactor) DisperseTokenFixed(opts *bind.TransactOpts, token common.Address, recipients []common.Address) (*types.Transaction, error) {
	return _Disperse.contract.Transact(opts, "disperseTokenFixed", token, recipients)
}

// DisperseTokenFixed is a paid mutator transaction binding the contract method 0xfb169f36.
//
// Solidity: function disperseTokenFixed(address token, address[] recipients) returns()
func (_Disperse *DisperseSession) DisperseTokenFixed(token common.Address, recipients []common.Address) (*types.Transaction, error) {
	return _Disperse.Contract.DisperseTokenFixed(&_Disperse.TransactOpts, token, recipients)
}

// DisperseTokenFixed is a paid mutator transaction binding the contract method 0xfb169f36.
//
// Solidity: function disperseTokenFixed(address token, address[] recipients) returns()
func (_Disperse *DisperseTransactorSession) DisperseTokenFixed(token common.Address, recipients []common.Address) (*types.Transaction, error) {
	return _Disperse.Contract.DisperseTokenFixed(&_Disperse.TransactOpts, token, recipients)
}

// DisperseTokenSimple is a paid mutator transaction binding the contract method 0x51ba162c.
//
// Solidity: function disperseTokenSimple(address token, address[] recipients, uint256[] values) returns()
func (_Disperse *DisperseTransactor) DisperseTokenSimple(opts *bind.TransactOpts, token common.Address, recipients []common.Address, values []*big.Int) (*types.Transaction, error) {
	return _Disperse.contract.Transact(opts, "disperseTokenSimple", token, recipients, values)
}

// DisperseTokenSimple is a paid mutator transaction binding the contract method 0x51ba162c.
//
// Solidity: function disperseTokenSimple(address token, address[] recipients, uint256[] values) returns()
func (_Disperse *DisperseSession) DisperseTokenSimple(token common.Address, recipients []common.Address, values []*big.Int) (*types.Transaction, error) {
	return _Disperse.Contract.DisperseTokenSimple(&_Disperse.TransactOpts, token, recipients, values)
}

// DisperseTokenSimple is a paid mutator transaction binding the contract method 0x51ba162c.
//
// Solidity: function disperseTokenSimple(address token, address[] recipients, uint256[] values) returns()
func (_Disperse *DisperseTransactorSession) DisperseTokenSimple(token common.Address, recipients []common.Address, values []*big.Int) (*types.Transaction, error) {
	return _Disperse.Contract.DisperseTokenSimple(&_Disperse.TransactOpts, token, recipients, values)
}

// SetFixedEtherAmount is a paid mutator transaction binding the contract method 0x5fcc7d15.
//
// Solidity: function setFixedEtherAmount(uint256 amount) returns()
func (_Disperse *DisperseTransactor) SetFixedEtherAmount(opts *bind.TransactOpts, amount *big.Int) (*types.Transaction, error) {
	return _Disperse.contract.Transact(opts, "setFixedEtherAmount", amount)
}

// SetFixedEtherAmount is a paid mutator transaction binding the contract method 0x5fcc7d15.
//
// Solidity: function setFixedEtherAmount(uint256 amount) returns()
func (_Disperse *DisperseSession) SetFixedEtherAmount(amount *big.Int) (*types.Transaction, error) {
	return _Disperse.Contract.SetFixedEtherAmount(&_Disperse.TransactOpts, amount)
}

// SetFixedEtherAmount is a paid mutator transaction binding the contract method 0x5fcc7d15.
//
// Solidity: function setFixedEtherAmount(uint256 amount) returns()
func (_Disperse *DisperseTransactorSession) SetFixedEtherAmount(amount *big.Int) (*types.Transaction, error) {
	return _Disperse.Contract.SetFixedEtherAmount(&_Disperse.TransactOpts, amount)
}

// SetFixedTokenAmount is a paid mutator transaction binding the contract method 0x107520fe.
//
// Solidity: function setFixedTokenAmount(uint256 amount) returns()
func (_Disperse *DisperseTransactor) SetFixedTokenAmount(opts *bind.TransactOpts, amount *big.Int) (*types.Transaction, error) {
	return _Disperse.contract.Transact(opts, "setFixedTokenAmount", amount)
}

// SetFixedTokenAmount is a paid mutator transaction binding the contract method 0x107520fe.
//
// Solidity: function setFixedTokenAmount(uint256 amount) returns()
func (_Disperse *DisperseSession) SetFixedTokenAmount(amount *big.Int) (*types.Transaction, error) {
	return _Disperse.Contract.SetFixedTokenAmount(&_Disperse.TransactOpts, amount)
}

// SetFixedTokenAmount is a paid mutator transaction binding the contract method 0x107520fe.
//
// Solidity: function setFixedTokenAmount(uint256 amount) returns()
func (_Disperse *DisperseTransactorSession) SetFixedTokenAmount(amount *big.Int) (*types.Transaction, error) {
	return _Disperse.Contract.SetFixedTokenAmount(&_Disperse.TransactOpts, amount)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Disperse *DisperseTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _Disperse.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Disperse *DisperseSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Disperse.Contract.TransferOwnership(&_Disperse.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Disperse *DisperseTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Disperse.Contract.TransferOwnership(&_Disperse.TransactOpts, newOwner)
}
