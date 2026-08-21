package generator_test

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	mrand "math/rand/v2"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/sei-protocol/sei-load/config"
	"github.com/sei-protocol/sei-load/generator"
	"github.com/sei-protocol/sei-load/generator/scenarios"
	"github.com/sei-protocol/sei-load/types"
	"github.com/sei-protocol/sei-load/utils"
	"github.com/sei-protocol/sei-load/utils/require"
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

func (s *collectingSender) collected() []*types.LoadTx {
	for inner := range s.inner.Lock() {
		return inner.txs
	}
	panic("unreachable")
}

func (s *collectingSender) Nonce(acc types.Account) uint64 {
	for inner := range s.inner.Lock() {
		return inner.nonces[acc.Address]
	}
	panic("unreachable")
}

func newTestRng(seed uint64) *mrand.Rand {
	return mrand.New(mrand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
}

func generateN(t *testing.T, rng *mrand.Rand, gen *generator.Generator, n int) []*types.LoadTx {
	t.Helper()
	sender := newCollectingSender(n)
	err := gen.Run(t.Context(), rng, sender)
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

	rng := newTestRng(1)
	gen, err := generator.NewGenerator(t.Context(), rng, cfg, types.NewAccount(false))
	require.NoError(t, err)
	require.NotNil(t, gen)

	totalTxs := 100
	txs := generateN(t, rng, gen, totalTxs)
	signer := ethtypes.LatestSignerForChainID(big.NewInt(cfg.ChainID))
	for _, tx := range txs {
		_, err := ethtypes.Sender(signer, tx.EthTx)
		require.Error(t, err)
	}
	require.Len(t, txs, totalTxs)

	// Count occurrences per scenario. Every tx also carries its operation: the
	// generator records it, and the sender reads it off the LoadTx to label a
	// sample, so a tx that reaches the sender without one loses that sample.
	scenarioCounts := make(map[string]int)
	for _, tx := range txs {
		require.NotNil(t, tx.Scenario)
		require.NotEmpty(t, tx.Scenario.Operation)
		scenario := tx.Scenario.Name
		scenarioCounts[scenario]++
	}

	// Weight 2:3 → Expect ≈40:60 distribution (±10 allowed)
	require.InDelta(t, 40, float64(scenarioCounts[scenarios.ERC20]), 10)
	require.InDelta(t, 60, float64(scenarioCounts[scenarios.EVMTransfer]), 10)
}

// TestPrewarmLabelsEveryTransaction covers the second path that reaches the
// sender. Prewarm does not go through Generator.Run, so the dimensions it
// carries are not covered by the tests above, and both defects this path has had
// were invisible for that reason: it once sent under a different spelling of a
// scenario's name, and it once stamped IntendedSendTime, which defeated the
// inclusion tracker's not-scheduled guard.
func TestPrewarmLabelsEveryTransaction(t *testing.T) {
	cfg := &config.LoadConfig{
		ChainID:   7777,
		Endpoints: []string{"http://localhost:8545"},
		Accounts:  &config.AccountConfig{Accounts: 4},
		Scenarios: []config.Scenario{{Name: scenarios.EVMTransfer, Weight: 1}},
	}
	rng := newTestRng(3)
	gen, err := generator.NewGenerator(t.Context(), rng, cfg, types.NewAccount(false))
	require.NoError(t, err)

	sender := newCollectingSender(1000)
	require.NoError(t, gen.Prewarm(t.Context(), rng, cfg, sender))

	txs := sender.collected()
	require.NotEmpty(t, txs)
	for _, tx := range txs {
		require.Equal(t, scenarios.Prewarm, tx.Scenario.Name)
		require.NotEmpty(t, tx.Scenario.Operation)
		require.True(t, tx.IntendedSendTime.IsZero(),
			"prewarm has no arrival schedule, so the inclusion tracker must skip it")
	}
}
