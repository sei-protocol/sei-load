// Package offline provides backend-neutral load scenarios for executors that
// consume raw Ethereum transactions and an explicitly seeded genesis state.
package offline

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

const (
	Transfer      = "transfer"
	ERC20Transfer = "erc20-transfer"
)

// GenesisWriter is the state surface needed to prepare an offline scenario.
// Implementations may ignore writes that are irrelevant to their backend.
type GenesisWriter interface {
	SetBalance(common.Address, *big.Int)
	SetCode(common.Address, []byte)
	SetState(common.Address, common.Hash, common.Hash)
}

// Config contains transaction and genesis settings shared by offline scenarios.
type Config struct {
	ChainID       *big.Int
	GasPrice      *big.Int
	SenderBalance *big.Int
	TransferValue *big.Int
	GasLimit      uint64
	ERC20Contract common.Address
}

// Scenario builds signed transactions and the genesis state they require.
type Scenario interface {
	SetupGenesis(GenesisWriter) error
	SeedSender(GenesisWriter, common.Address)
	BuildTransaction(*ecdsa.PrivateKey, uint64, common.Address) (*ethtypes.Transaction, error)
}

// NewScenario constructs a backend-neutral scenario.
func NewScenario(kind string, cfg Config) (Scenario, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	cfg = cloneConfig(cfg)
	switch kind {
	case Transfer:
		return newTransferScenario(cfg), nil
	case ERC20Transfer:
		if cfg.ERC20Contract == (common.Address{}) {
			return nil, fmt.Errorf("erc20 contract must be non-zero")
		}
		return newERC20TransferScenario(cfg)
	default:
		return nil, fmt.Errorf("unsupported offline scenario %q", kind)
	}
}

func validateConfig(cfg Config) error {
	switch {
	case cfg.ChainID == nil || cfg.ChainID.Sign() <= 0:
		return fmt.Errorf("chain ID must be positive")
	case cfg.GasPrice == nil || cfg.GasPrice.Sign() < 0:
		return fmt.Errorf("gas price must be non-negative")
	case cfg.GasPrice.BitLen() > 256:
		return fmt.Errorf("gas price must fit in 256 bits")
	case cfg.SenderBalance == nil || cfg.SenderBalance.Sign() < 0:
		return fmt.Errorf("sender balance must be non-negative")
	case cfg.SenderBalance.BitLen() > 256:
		return fmt.Errorf("sender balance must fit in 256 bits")
	case cfg.TransferValue == nil || cfg.TransferValue.Sign() < 0:
		return fmt.Errorf("transfer value must be non-negative")
	case cfg.TransferValue.BitLen() > 256:
		return fmt.Errorf("transfer value must fit in 256 bits")
	case cfg.GasLimit == 0:
		return fmt.Errorf("gas limit must be positive")
	default:
		return nil
	}
}

func cloneConfig(cfg Config) Config {
	cfg.ChainID = new(big.Int).Set(cfg.ChainID)
	cfg.GasPrice = new(big.Int).Set(cfg.GasPrice)
	cfg.SenderBalance = new(big.Int).Set(cfg.SenderBalance)
	cfg.TransferValue = new(big.Int).Set(cfg.TransferValue)
	return cfg
}
