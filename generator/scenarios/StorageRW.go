package scenarios

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/sei-protocol/sei-load/config"
	"github.com/sei-protocol/sei-load/generator/bindings"
	"github.com/sei-protocol/sei-load/types"
)

const StorageRW = "storagerw"

const (
	// storageRWBaseGas covers the cold-first-touch rmw (cold SLOAD + zero->nonzero
	// SSTORE, ~44k) plus the fixed calldata head, with headroom. The distribution-
	// driven pad's intrinsic cost is added on top per-tx; see package doc.
	storageRWBaseGas = 50000
	// calldataZeroByteGas is the EIP-2028 intrinsic cost of one zero calldata byte.
	// The pad is a zero-filled slice, so each pad byte costs exactly this.
	calldataZeroByteGas = 4
	// storageRWWriteValue is the constant value write() stores; the load contract
	// never asserts on it.
	storageRWWriteValue = 1
)

// storageRWDefaultSlot is the single slot every tx targets when no key
// distribution is configured (the scaffold's 100%-conflict default).
var storageRWDefaultSlot = big.NewInt(0)

// StorageRWScenario implements the TxGenerator interface for StorageRWv1 contract operations
type StorageRWScenario struct {
	*ContractScenarioBase[bindings.StorageRWv1]
	contract *bindings.StorageRWv1
}

// NewStorageRWScenario creates a new StorageRW scenario
func NewStorageRWScenario(cfg config.Scenario) TxGenerator {
	scenario := &StorageRWScenario{}
	scenario.ContractScenarioBase = NewContractScenarioBase[bindings.StorageRWv1](scenario, cfg)
	return scenario
}

// Name returns the name of the scenario.
func (s *StorageRWScenario) Name() string {
	return StorageRW
}

// DeployContract implements ContractDeployer interface - deploys StorageRWv1.
// StorageRWv1 is mapping-backed and takes no constructor arguments; the keyspace
// is generator-side.
func (s *StorageRWScenario) DeployContract(opts *bind.TransactOpts, client *ethclient.Client) (common.Address, *ethtypes.Transaction, error) {
	address, tx, _, err := bindings.DeployStorageRWv1(opts, client)
	return address, tx, err
}

// GetBindFunc implements ContractDeployer interface - returns the binding function
func (s *StorageRWScenario) GetBindFunc() ContractBindFunc[bindings.StorageRWv1] {
	return bindings.NewStorageRWv1
}

// SetContract implements ContractDeployer interface - stores the contract instance
func (s *StorageRWScenario) SetContract(contract *bindings.StorageRWv1) {
	s.contract = contract
}

// Attach implements TxGenerator interface - attaches to an existing contract
func (s *StorageRWScenario) Attach(config *config.LoadConfig, address common.Address) error {
	// Call base Attach to set deployed flag and config
	if err := s.ContractScenarioBase.Attach(config, address); err != nil {
		return err
	}

	var client *ethclient.Client
	var err error
	if !config.MockDeploy {
		client, err = ethclient.Dial(config.Endpoints[0])
		if err != nil {
			return err
		}
	}

	s.contract, err = bindings.NewStorageRWv1(address, client)
	return err
}

// CreateContractTransaction implements ContractDeployer interface - builds one
// StorageRWv1 transaction whose slot (key contention), operation, and calldata
// pad (tx size) are drawn from the configured distributions. With no
// distribution config it reproduces the scaffold's single-slot empty-pad rmw.
// See package doc for the gas rationale.
func (s *StorageRWScenario) CreateContractTransaction(auth *bind.TransactOpts, scenario *types.TxScenario) (*ethtypes.Transaction, error) {
	slot, err := s.pickSlot()
	if err != nil {
		return nil, err
	}
	pad, err := s.pickPad()
	if err != nil {
		return nil, err
	}

	// The pad's intrinsic calldata cost is the only gas the base does not already
	// cover; add it so a large pad cannot underprovision the tx.
	auth.GasLimit = storageRWBaseGas + uint64(len(pad))*calldataZeroByteGas

	switch s.pickOp() {
	case config.OpRead:
		return s.contract.Read(auth, slot, pad)
	case config.OpWrite:
		return s.contract.Write(auth, slot, big.NewInt(storageRWWriteValue), pad)
	default:
		return s.contract.Rmw(auth, slot, pad)
	}
}

// pickSlot draws the storage slot from the key distribution over the configured
// RecordCount keyspace. With no key distribution it returns the fixed default
// slot, preserving the scaffold's 100%-conflict behavior.
func (s *StorageRWScenario) pickSlot() (*big.Int, error) {
	cfg := s.scenarioConfig
	if cfg.KeyDistribution == nil || cfg.RecordCount == 0 {
		return storageRWDefaultSlot, nil
	}
	idx, err := cfg.KeyDistribution.SampleIndex(cfg.RecordCount)
	if err != nil {
		return nil, err
	}
	return new(big.Int).SetUint64(idx), nil
}

// pickPad draws the calldata pad length from the size distribution over the
// configured SizeBuckets histogram, on a sub-stream independent of the key draw.
// With no size distribution it returns an empty pad, preserving the scaffold.
func (s *StorageRWScenario) pickPad() ([]byte, error) {
	cfg := s.scenarioConfig
	if cfg.SizeDistribution == nil || len(cfg.SizeBuckets) == 0 {
		return nil, nil
	}
	bucket, err := cfg.SizeDistribution.SampleIndex(uint64(len(cfg.SizeBuckets)))
	if err != nil {
		return nil, err
	}
	return make([]byte, cfg.SizeBuckets[bucket]), nil
}

// pickOp selects read/write/rmw from the configured mix on its own independent
// sub-stream. With no mix it returns rmw, preserving the scaffold.
func (s *StorageRWScenario) pickOp() config.Operation {
	if s.scenarioConfig.Operations == nil {
		return config.OpRmw
	}
	return s.scenarioConfig.Operations.Select()
}
