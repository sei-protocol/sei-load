package generator

import (
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

func (g *scenarioGenerator) Generate() (*types.LoadTx, bool) {
	sender := g.accountPool.NextAccount()
	receiver := g.accountPool.NextAccount()
	return g.scenario.Generate(&types.TxScenario{
		Name:     g.scenario.Name(),
		Sender:   sender,
		Receiver: receiver.Address,
	}), true
}
