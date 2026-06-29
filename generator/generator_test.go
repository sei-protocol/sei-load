package generator_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/sei-protocol/sei-load/config"
	"github.com/sei-protocol/sei-load/generator"
	"github.com/sei-protocol/sei-load/generator/scenarios"
	"github.com/sei-protocol/sei-load/types"
	"github.com/sei-protocol/sei-load/utils"
	"github.com/sei-protocol/sei-load/utils/require"
	rngutil "github.com/sei-protocol/sei-load/utils/rng"
)

var errStopGeneration = errors.New("stop generation")

type inner struct {
	txs    []*types.LoadTx
	nonces map[common.Address]uint64
}

type collectingSender struct {
	limit int
	inner utils.Mutex[*inner]
}

func newCollectingSender(limit int) *collectingSender {
	return &collectingSender{
		limit: limit,
		inner: utils.NewMutex(&inner{
			nonces: map[common.Address]uint64{},
		}),
	}
}

func (s *collectingSender) Send(_ context.Context, tx *types.LoadTx) error {
	for inner := range s.inner.Lock() {
		inner.txs = append(inner.txs, tx)
		addr := tx.Scenario.Sender.Address
		if tx.Scenario.Nonce != inner.nonces[addr] {
			return fmt.Errorf("bad nonce")
		}
		inner.nonces[addr] += 1
		if len(inner.txs) >= s.limit {
			return errStopGeneration
		}
	}
	return nil
}

func (s *collectingSender) Flush(context.Context) error { return nil }

func (s *collectingSender) Nonce(acc types.Account) uint64 {
	for inner := range s.inner.Lock() {
		return inner.nonces[acc.Address]
	}
	panic("unreachable")
}

func generateN(t *testing.T, src *rngutil.Source, gen *generator.Generator, n int) []*types.LoadTx {
	t.Helper()
	sender := newCollectingSender(n)
	err := gen.Run(t.Context(), src.Rand("generator:test:draws"), sender)
	require.ErrorIs(t, err, errStopGeneration)
	for inner := range sender.inner.Lock() {
		return inner.txs
	}
	panic("unreachable")
}

func TestScenarioWeightsAndAccountDistribution(t *testing.T) {
	cfg := &config.LoadConfig{
		ChainID:    7777,
		MockDeploy: true,
		Endpoints:  []string{"http://localhost:8545"}, // Add endpoints for Attach method
		Scenarios: []config.Scenario{
			{
				Name:   scenarios.ERC20,
				Weight: 2,
				Accounts: &config.AccountConfig{
					Accounts:       10,
					NewAccountRate: 0.0,
				},
			},
			{
				Name:   scenarios.EVMTransfer,
				Weight: 3,
				Accounts: &config.AccountConfig{
					Accounts:       20,
					NewAccountRate: 0.0,
				},
			},
		},
	}

	rngSource := generator.ResolveSeed(cfg)
	gen, err := generator.NewGenerator(rngSource.Rand(rngutil.StreamWeightedShuffle), cfg)
	require.NoError(t, err)
	require.NotNil(t, gen)

	totalTxs := 100
	txs := generateN(t, rngSource, gen, totalTxs)
	require.Len(t, txs, totalTxs)

	// Count occurrences per scenario
	scenarioCounts := make(map[string]int)
	for _, tx := range txs {
		require.NotNil(t, tx.Scenario)
		scenario := tx.Scenario.Name
		scenarioCounts[scenario]++
	}

	// Weight 2:3 → Expect ≈40:60 distribution (±10 allowed)
	require.InDelta(t, 40, float64(scenarioCounts[scenarios.ERC20]), 10)
	require.InDelta(t, 60, float64(scenarioCounts[scenarios.EVMTransfer]), 10)
}
