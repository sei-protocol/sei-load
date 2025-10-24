package scenarios

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/sei-protocol/sei-load/config"
	"github.com/sei-protocol/sei-load/generator/bindings"
	"github.com/sei-protocol/sei-load/types"
)

const SimpleTarget = "simpletarget"

// SimpleTargetScenario implements the TxGenerator interface for SimpleTarget contract operations
type SimpleTargetScenario struct {
	*ContractScenarioBase[bindings.SimpleTarget]
	contract *bindings.SimpleTarget
}

// NewSimpleTargetScenario creates a new SimpleTarget scenario
func NewSimpleTargetScenario(cfg config.Scenario) TxGenerator {
	scenario := &SimpleTargetScenario{}
	scenario.ContractScenarioBase = NewContractScenarioBase[bindings.SimpleTarget](scenario, cfg)
	return scenario
}

// Name returns the name of the scenario.
func (s *SimpleTargetScenario) Name() string {
	return SimpleTarget
}

// DeployContract implements ContractDeployer interface - deploys SimpleTarget with specific constructor args
func (s *SimpleTargetScenario) DeployContract(opts *bind.TransactOpts, client *ethclient.Client) (common.Address, *ethtypes.Transaction, error) {
	address, tx, _, err := bindings.DeploySimpleTarget(opts, client)
	return address, tx, err
}

// GetBindFunc implements ContractDeployer interface - returns the binding function
func (s *SimpleTargetScenario) GetBindFunc() ContractBindFunc[bindings.SimpleTarget] {
	return bindings.NewSimpleTarget
}

// SetContract implements ContractDeployer interface - stores the contract instance
func (s *SimpleTargetScenario) SetContract(contract *bindings.SimpleTarget) {
	s.contract = contract
}

// Attach implements TxGenerator interface - attaches to an existing contract
func (s *SimpleTargetScenario) Attach(config *config.LoadConfig, address common.Address) error {
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

	s.contract, err = bindings.NewSimpleTarget(address, client)
	return err
}

// CreateContractTransaction implements ContractDeployer interface - creates SimpleTarget transaction
func (s *SimpleTargetScenario) CreateContractTransaction(auth *bind.TransactOpts, scenario *types.TxScenario) (*ethtypes.Transaction, error) {
	// Call setValue to update the value (simple state change)
	return s.contract.SetValue(auth, bigOne)
}
