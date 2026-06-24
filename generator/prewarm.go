package generator

import (
	"github.com/sei-protocol/sei-load/config"
	"github.com/sei-protocol/sei-load/generator/scenarios"
	"github.com/sei-protocol/sei-load/types"
)

// PrewarmGenerator generates self-transfer transactions to prewarm account nonces
type PrewarmGenerator struct {
	registry       *types.AccountRegistry
	evmScenario    scenarios.TxGenerator
	currentAccount int
	finished       bool
}

// NewPrewarmGenerator creates a new prewarm generator using all account pools from the registry.
func NewPrewarmGenerator(cfg *config.LoadConfig, registry *types.AccountRegistry) *PrewarmGenerator {
	// Create EVMTransfer scenario for prewarming
	evmScenario := scenarios.NewEVMTransferScenario(config.Scenario{})

	// Deploy/initialize the scenario (EVMTransfer doesn't need actual deployment)
	deployerAccounts := types.GenerateAccounts(1)
	deployer := deployerAccounts[0]
	evmScenario.Deploy(cfg, deployer)

	return &PrewarmGenerator{
		registry:       registry,
		evmScenario:    evmScenario,
		currentAccount: 0,
		finished:       false,
	}
}

// Generate generates self-transfer transactions until all known accounts are prewarmed.
func (pg *PrewarmGenerator) Generate() (*types.LoadTx, bool) {
	accounts := pg.registry.Accounts()

	// Check if we're already finished
	if pg.finished || pg.currentAccount >= len(accounts) {
		return nil, false
	}

	account := accounts[pg.currentAccount]
	if account.Nonce > 0 {
		pg.currentAccount++
		return pg.Generate()
	}
	pg.currentAccount++

	// Create self-transfer transaction
	scenario := &types.TxScenario{
		Name:     "EVMTransfer",
		Sender:   account,
		Receiver: account.Address, // Send to self
	}

	// Generate the transaction using EVMTransfer scenario
	return pg.evmScenario.Generate(scenario), true
}
