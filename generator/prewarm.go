package generator

import (
	mrand "math/rand/v2"

	"github.com/sei-protocol/sei-load/config"
	"github.com/sei-protocol/sei-load/generator/scenarios"
	"github.com/sei-protocol/sei-load/types"
)

// PrewarmGenerator generates self-transfer transactions to prewarm account nonces
type PrewarmGenerator struct {
	accounts       []*types.Account
	evmScenario    scenarios.TxGenerator
}

// NewPrewarmGenerator creates a new prewarm generator using all account pools from the registry.
func NewPrewarmGenerator(cfg *config.LoadConfig, accounts []*types.Account) *PrewarmGenerator {
	// Create EVMTransfer scenario for prewarming
	evmScenario := scenarios.NewEVMTransferScenario(config.Scenario{})

	// Deploy/initialize the scenario (EVMTransfer doesn't need actual deployment)
	deployer := types.NewAccount()
	evmScenario.Deploy(cfg, deployer)
	return &PrewarmGenerator{
		accounts:       accounts,
		evmScenario:    evmScenario,
	}
}

// Generate generates self-transfer transactions until all known accounts are prewarmed.
func (pg *PrewarmGenerator) Generate(rng *mrand.Rand) (*types.LoadTx, bool) {
	if len(pg.accounts)==0 {
		return nil, false
	}
	account := pg.accounts[0]
	pg.accounts = pg.accounts[1:]
	// Create self-transfer transaction
	scenario := &types.TxScenario{
		Name:     "EVMTransfer",
		Sender:   account,
		Receiver: account.Address, // Send to self
	}
	// Generate the transaction using EVMTransfer scenario
	return pg.evmScenario.Generate(rng, scenario), true
}
