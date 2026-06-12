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

// storageRWSlot is the fixed storage slot every transaction targets in this
// scaffold. The per-tx slot/value/pad distribution arrives in PLT-465.
var storageRWSlot = big.NewInt(0)

// storageRWPad is the fixed calldata pad for this scaffold (empty). The
// distribution-driven pad sizing arrives in PLT-465.
var storageRWPad = []byte{}

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

// CreateContractTransaction implements ContractDeployer interface - creates a
// StorageRWv1 transaction. This scaffold issues a fixed read-modify-write against
// a single hardcoded slot with an empty pad to prove the deploy/send path; the
// per-tx slot/value/pad distribution arrives in PLT-465.
func (s *StorageRWScenario) CreateContractTransaction(auth *bind.TransactOpts, scenario *types.TxScenario) (*ethtypes.Transaction, error) {
	return s.contract.Rmw(auth, storageRWSlot, storageRWPad)
}
