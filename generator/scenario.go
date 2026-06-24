package generator

import (
	mrand "math/rand/v2"

	"github.com/sei-protocol/sei-load/generator/scenarios"
	"github.com/sei-protocol/sei-load/types"
)

type scenarioGenerator struct {
	scenario    scenarios.TxGenerator
	accountPool *types.AccountPool
}

func NewScenarioGenerator(accounts *types.AccountPool, txg scenarios.TxGenerator) Generator {
	return &scenarioGenerator{
		scenario:    txg,
		accountPool: accounts,
	}
}

func (g *scenarioGenerator) Generate(rng *mrand.Rand) (*types.LoadTx, bool) {
	sender := g.accountPool.NextAccount(rng)
	receiver := g.accountPool.NextAccount(rng)
	return g.scenario.Generate(rng, &types.TxScenario{
		Name:     g.scenario.Name(),
		Sender:   sender,
		Receiver: receiver.Address,
	}), true
}
