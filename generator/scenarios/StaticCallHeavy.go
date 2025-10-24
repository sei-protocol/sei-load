package scenarios

import (
	"log"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/sei-protocol/sei-load/config"
	"github.com/sei-protocol/sei-load/generator/bindings"
	"github.com/sei-protocol/sei-load/generator/utils"
	"github.com/sei-protocol/sei-load/types"
)

const StaticCallHeavy = "staticcallheavy"

// StaticCallHeavyScenario implements the TxGenerator interface for StaticCallHeavy contract operations
type StaticCallHeavyScenario struct {
	*ScenarioBase
	contract *bindings.StaticCallHeavy
}

// NewStaticCallHeavyScenario creates a new StaticCallHeavy scenario
func NewStaticCallHeavyScenario(cfg config.Scenario) TxGenerator {
	scenario := &StaticCallHeavyScenario{}
	scenario.ScenarioBase = NewScenarioBase(scenario, cfg)
	return scenario
}

// Name returns the name of the scenario.
func (s *StaticCallHeavyScenario) Name() string {
	return StaticCallHeavy
}

// DeployScenario implements ScenarioDeployer interface - deploys SimpleTarget first, then StaticCallHeavy, then configures them
func (s *StaticCallHeavyScenario) DeployScenario(config *config.LoadConfig, deployer *types.Account) common.Address {
	// Connect to Ethereum client
	client, err := ethclient.Dial(config.Endpoints[0])
	if err != nil {
		panic("Failed to connect to Ethereum client: " + err.Error())
	}

	// Create deployment options
	opts, err := utils.CreateDeploymentOpts(config.GetChainID(), client, deployer)
	if err != nil {
		panic("Failed to create deployment options: " + err.Error())
	}

	// First deploy SimpleTarget contract
	simpleTargetAddress, simpleTargetTx, _, err := bindings.DeploySimpleTarget(opts, client)
	if err != nil {
		panic("Failed to deploy SimpleTarget: " + err.Error())
	}

	// Wait for SimpleTarget deployment to be mined
	receipt, err := bind.WaitMined(opts.Context, client, simpleTargetTx)
	if err != nil {
		panic("Failed to wait for SimpleTarget deployment: " + err.Error())
	}

	if receipt.Status != ethtypes.ReceiptStatusSuccessful {
		panic("SimpleTarget deployment failed")
	}
	log.Printf("SimpleTarget deployed successfully at %s", simpleTargetAddress.Hex())

	// Create new deployment options for StaticCallHeavy
	opts2, err := utils.CreateDeploymentOpts(config.GetChainID(), client, deployer)
	if err != nil {
		panic("Failed to create deployment options: " + err.Error())
	}

	// Now deploy StaticCallHeavy contract
	staticCallAddress, staticCallTx, _, err := bindings.DeployStaticCallHeavy(opts2, client)
	if err != nil {
		panic("Failed to deploy StaticCallHeavy: " + err.Error())
	}

	// Wait for StaticCallHeavy deployment to be mined
	receipt, err = bind.WaitMined(opts2.Context, client, staticCallTx)
	if err != nil {
		panic("Failed to wait for StaticCallHeavy deployment: " + err.Error())
	}

	if receipt.Status != ethtypes.ReceiptStatusSuccessful {
		panic("StaticCallHeavy deployment failed")
	}
	log.Printf("StaticCallHeavy deployed successfully at %s", staticCallAddress.Hex())

	// Create StaticCallHeavy contract instance
	staticCallContract, err := bindings.NewStaticCallHeavy(staticCallAddress, client)
	if err != nil {
		panic("Failed to create StaticCallHeavy contract instance: " + err.Error())
	}

	// Create new deployment options for configuration
	opts3, err := utils.CreateDeploymentOpts(config.GetChainID(), client, deployer)
	if err != nil {
		panic("Failed to create deployment options: " + err.Error())
	}

	// Configure StaticCallHeavy to call SimpleTarget
	configTx, err := staticCallContract.SetTargetContract(opts3, simpleTargetAddress)
	if err != nil {
		panic("Failed to configure StaticCallHeavy target contract: " + err.Error())
	}

	// Wait for configuration transaction to be mined
	receipt, err = bind.WaitMined(opts3.Context, client, configTx)
	if err != nil {
		panic("Failed to wait for configuration transaction: " + err.Error())
	}

	if receipt.Status != ethtypes.ReceiptStatusSuccessful {
		panic("Configuration transaction failed")
	}
	log.Printf("StaticCallHeavy configured successfully")

	// Store the contract instance
	s.contract = staticCallContract

	// Return the StaticCallHeavy address
	return staticCallAddress
}

// AttachScenario implements ScenarioDeployer interface - attaches to an existing scenario
func (s *StaticCallHeavyScenario) AttachScenario(config *config.LoadConfig, address common.Address) common.Address {
	err := s.Attach(config, address)
	if err != nil {
		panic("Failed to attach scenario: " + err.Error())
	}
	return address
}

// Attach implements TxGenerator interface - attaches to an existing contract
func (s *StaticCallHeavyScenario) Attach(config *config.LoadConfig, address common.Address) error {
	// Call base Attach to set deployed flag and config
	if err := s.ScenarioBase.Attach(config, address); err != nil {
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

	s.contract, err = bindings.NewStaticCallHeavy(address, client)
	if err != nil {
		return err
	}

	return nil
}

// CreateTransaction implements ScenarioDeployer interface - creates StaticCallHeavy transaction
func (s *StaticCallHeavyScenario) CreateTransaction(config *config.LoadConfig, scenario *types.TxScenario) (*ethtypes.Transaction, error) {
	// Create transaction options
	opts := utils.CreateTransactionOpts(config.GetChainID(), scenario)

	// Call performStaticCalls which performs 100 simple static calls
	// This will stress test the snapshot mechanism with actual static calls
	opts.GasLimit = 300000
	tx, err := s.contract.PerformStaticCalls(opts)
	if err != nil {
		return nil, err
	}

	return tx, nil
}
