package scenarios

import (
	"fmt"
	"math/big"
	mrand "math/rand/v2"

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
	// storageRWBaseGas covers execution plus the fixed calldata head. Measured
	// worst case is a read that first writes readAccumulator, at 46,269 including
	// intrinsic; rmw and write cold-first-touch sit near 44k. 50k clears all
	// three, but only by ~3.7k — and SSTORE_SET is a Sei governance parameter
	// (SeiSstoreSetGasEip2200, default 20,000), so a raise past ~23.7k would put
	// read out of gas. See package doc for why the limit is kept tight anyway.
	storageRWBaseGas = 50000
	// storageRWWriteValue is the constant value write stores. The load contract
	// never asserts on it.
	storageRWWriteValue = 1

	// abiWord is the 32-byte unit the ABI right-pads a dynamic argument up to,
	// so the pad reaches the wire as a whole number of words.
	abiWord = 32
	// calldataFloorGasPerByte is what a zero calldata byte costs under EIP-7623,
	// which is live on Sei (PragueTime is 0). The floor is 21000 + 10 per token
	// and a zero byte is one token, so charging 10 per padded pad byte on top of
	// the base always clears it: the base exceeds 21000 by more than the head's
	// worst-case token cost.
	//
	// The pre-Prague rate of 4 would be short above roughly 4.5 KiB of pad, and
	// Sei's ante checks only the intrinsic cost, not the floor — so such a tx is
	// admitted, reserves its full declared limit, then fails in execution with
	// GasUsed equal to the limit. It lands in a block as an included failure and
	// inflates the very gas-used metric the run reports.
	calldataFloorGasPerByte = 10
)

// storageRWDefaultSlot is the single slot every tx targets when no key
// distribution is configured — the 100%-conflict default.
var storageRWDefaultSlot = big.NewInt(0)

// StorageRWScenario implements the TxGenerator interface for StorageRWv1 contract operations
type StorageRWScenario struct {
	*ContractScenarioBase[bindings.StorageRWv1]
	contract   *bindings.StorageRWv1
	operations *config.OperationPicker
}

// NewStorageRWScenario creates a new StorageRW scenario
func NewStorageRWScenario(cfg config.Scenario) TxGenerator {
	scenario := &StorageRWScenario{
		operations: config.StorageRWOperations.Picker(cfg.Operations),
	}
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
// StorageRWv1 transaction whose slot (key contention), calldata pad (tx size),
// and operation are drawn from the scenario config. With none of the three
// configured it falls back to a single-slot empty-pad rmw and draws no
// randomness. See package doc for the gas rationale.
//
// The draws run in a fixed order: slot, then pad, then operation. That order
// must stay stable — all three share the run's single PRNG, so reordering them
// shifts every subsequent draw and diverges a replay at the same seed.
func (s *StorageRWScenario) CreateContractTransaction(rng *mrand.Rand, auth *bind.TransactOpts, scenario *types.TxScenario) (*ethtypes.Transaction, error) {
	slot, err := s.pickSlot(rng)
	if err != nil {
		return nil, err
	}
	pad, err := s.pickPad(rng)
	if err != nil {
		return nil, err
	}

	// Charge the pad at the EIP-7623 floor rate over its on-wire length, which
	// the ABI rounds up to a whole word.
	paddedPad := (uint64(len(pad)) + abiWord - 1) / abiWord * abiWord
	auth.GasLimit = storageRWBaseGas + paddedPad*calldataFloorGasPerByte

	// Record the draw on the scenario so the send path and the inclusion tracker
	// can attribute their samples to the operation that produced them.
	op := s.operations.Select(rng)
	scenario.Operation = op

	switch op {
	case config.OpRmw:
		return s.contract.Rmw(auth, slot, pad)
	case config.OpRead:
		return s.contract.Read(auth, slot, pad)
	case config.OpWrite:
		return s.contract.Write(auth, slot, big.NewInt(storageRWWriteValue), pad)
	default:
		return nil, fmt.Errorf("storagerw: no contract method for operation %q", op)
	}
}

// pickSlot draws the storage slot from the key distribution over the configured
// RecordCount keyspace. With no key distribution it returns the fixed default
// slot and consumes no randomness.
func (s *StorageRWScenario) pickSlot(rng *mrand.Rand) (*big.Int, error) {
	cfg := s.scenarioConfig
	if cfg.KeyDistribution == nil || cfg.RecordCount == 0 {
		return storageRWDefaultSlot, nil
	}
	idx, err := cfg.KeyDistribution.SampleIndex(rng, cfg.RecordCount)
	if err != nil {
		return nil, err
	}
	return new(big.Int).SetUint64(idx), nil
}

// pickPad draws the calldata pad length from the size distribution over the
// configured SizeBuckets histogram. With no size distribution it returns an
// empty pad and consumes no randomness.
func (s *StorageRWScenario) pickPad(rng *mrand.Rand) ([]byte, error) {
	cfg := s.scenarioConfig
	if cfg.SizeDistribution == nil || len(cfg.SizeBuckets) == 0 {
		return nil, nil
	}
	bucket, err := cfg.SizeDistribution.SampleIndex(rng, uint64(len(cfg.SizeBuckets)))
	if err != nil {
		return nil, err
	}
	return make([]byte, cfg.SizeBuckets[bucket]), nil
}
