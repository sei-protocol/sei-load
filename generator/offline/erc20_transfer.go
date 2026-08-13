package offline

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"

	"github.com/sei-protocol/sei-load/generator/bindings"
)

const erc20BalancesSlot = uint64(4)

var loadERC20Runtime = sync.OnceValues(buildERC20Runtime)

type erc20TransferScenario struct {
	cfg      Config
	signer   ethtypes.Signer
	contract *abi.ABI
}

func newERC20TransferScenario(cfg Config) (*erc20TransferScenario, error) {
	contract, err := bindings.ERC20MetaData.GetAbi()
	if err != nil {
		return nil, fmt.Errorf("parse ERC20 ABI: %w", err)
	}
	return &erc20TransferScenario{
		cfg:      cfg,
		signer:   ethtypes.LatestSignerForChainID(cfg.ChainID),
		contract: contract,
	}, nil
}

func (s *erc20TransferScenario) SetupGenesis(state GenesisWriter) error {
	code, err := ERC20RuntimeCode()
	if err != nil {
		return err
	}
	state.SetCode(s.cfg.ERC20Contract, code)
	return nil
}

func (s *erc20TransferScenario) SeedSender(state GenesisWriter, sender common.Address) {
	state.SetBalance(sender, s.cfg.SenderBalance)
	state.SetState(s.cfg.ERC20Contract, ERC20BalanceSlot(sender), common.BigToHash(s.cfg.TransferValue))
}

func (s *erc20TransferScenario) BuildTransaction(key *ecdsa.PrivateKey, nonce uint64, recipient common.Address) (*ethtypes.Transaction, error) {
	if key == nil {
		return nil, fmt.Errorf("sender private key is required")
	}
	data, err := s.contract.Pack("transfer", recipient, s.cfg.TransferValue)
	if err != nil {
		return nil, fmt.Errorf("pack ERC20 transfer: %w", err)
	}
	tx := ethtypes.NewTx(&ethtypes.LegacyTx{
		Nonce:    nonce,
		GasPrice: new(big.Int).Set(s.cfg.GasPrice),
		Gas:      s.cfg.GasLimit,
		To:       &s.cfg.ERC20Contract,
		Value:    new(big.Int),
		Data:     data,
	})
	return ethtypes.SignTx(tx, s.signer, key)
}

// ERC20RuntimeCode returns a copy of the runtime produced by the committed
// sei-load ERC20 creation bytecode.
func ERC20RuntimeCode() ([]byte, error) {
	code, err := loadERC20Runtime()
	if err != nil {
		return nil, err
	}
	return append([]byte(nil), code...), nil
}

// ERC20BalanceSlot returns the storage key for _balances[owner] in the
// committed sei-load ERC20 contract.
func ERC20BalanceSlot(owner common.Address) common.Hash {
	var encoded [64]byte
	copy(encoded[12:32], owner.Bytes())
	new(big.Int).SetUint64(erc20BalancesSlot).FillBytes(encoded[32:])
	return crypto.Keccak256Hash(encoded[:])
}

func buildERC20Runtime() ([]byte, error) {
	contract, err := bindings.ERC20MetaData.GetAbi()
	if err != nil {
		return nil, fmt.Errorf("parse ERC20 ABI: %w", err)
	}
	constructor, err := contract.Constructor.Inputs.Pack("LoadToken", "LT")
	if err != nil {
		return nil, fmt.Errorf("pack ERC20 constructor: %w", err)
	}
	initCode := append(common.FromHex(bindings.ERC20Bin), constructor...)
	code, _, _, err := runtime.Create(initCode, &runtime.Config{
		ChainConfig: params.AllEthashProtocolChanges,
		Origin:      common.HexToAddress("0x1"),
		BlockNumber: big.NewInt(1),
		Time:        1_700_000_000,
		GasLimit:    10_000_000,
		GasPrice:    new(big.Int),
		Value:       new(big.Int),
		BaseFee:     new(big.Int),
	})
	if err != nil {
		return nil, fmt.Errorf("execute ERC20 constructor: %w", err)
	}
	if len(code) == 0 {
		return nil, fmt.Errorf("ERC20 constructor returned empty runtime")
	}
	return code, nil
}
