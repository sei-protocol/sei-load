package generator

import (
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"

	"github.com/sei-protocol/sei-load/config"
	"github.com/sei-protocol/sei-load/generator/scenarios"
	"github.com/sei-protocol/sei-load/types"
)

type scenarioGenerator struct {
	scenario    scenarios.TxGenerator
	accountPool types.AccountPool
	scenarioCfg config.Scenario
	mu          sync.RWMutex
}

func NewScenarioGenerator(accounts types.AccountPool,
	txg scenarios.TxGenerator, scenarioCfg config.Scenario) Generator {
	return &scenarioGenerator{
		scenario:    txg,
		accountPool: accounts,
		scenarioCfg: scenarioCfg,
	}
}

func (g *scenarioGenerator) GenerateN(n int) []*types.LoadTx {
	result := make([]*types.LoadTx, 0, n)
	for i := 0; i < n; i++ {
		if tx, ok := g.Generate(); ok {
			result = append(result, tx)
		} else {
			break // Generator is done
		}
	}
	return result
}

func (g *scenarioGenerator) Generate() (*types.LoadTx, bool) {
	sender := g.accountPool.NextAccount()
	if sender == nil {
		return nil, false
	}
	var receiver common.Address
	if addr := strings.TrimSpace(g.scenarioCfg.FixedReceiver); addr != "" {
		receiver = common.HexToAddress(addr)
	} else {
		rcv := g.accountPool.NextAccount()
		if rcv == nil {
			return nil, false
		}
		receiver = rcv.Address
	}
	return g.scenario.Generate(&types.TxScenario{
		Name:     g.scenario.Name(),
		Sender:   sender,
		Receiver: receiver,
	}), true
}

func (sg *scenarioGenerator) GetAccountPools() []types.AccountPool {
	sg.mu.RLock()
	defer sg.mu.RUnlock()
	return []types.AccountPool{sg.accountPool}
}
