package generator_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sei-protocol/sei-load/config"
	"github.com/sei-protocol/sei-load/generator"
	"github.com/sei-protocol/sei-load/generator/scenarios"
)

func seededConfig(t *testing.T, seed uint64) *config.LoadConfig {
	t.Helper()
	s := seed
	return &config.LoadConfig{
		ChainID:    7777,
		MockDeploy: true,
		Endpoints:  []string{"http://localhost:8545"},
		Seed:       &s,
		Scenarios: []config.Scenario{
			{
				Name:      scenarios.EVMTransfer,
				Weight:    2,
				Accounts:  &config.AccountConfig{Accounts: 10, NewAccountRate: 0.5},
				GasPicker: randomGasPicker(t, 21000, 90000),
			},
			{
				Name:      scenarios.EVMTransferNoop,
				Weight:    3,
				Accounts:  &config.AccountConfig{Accounts: 20, NewAccountRate: 0.5},
				GasPicker: randomGasPicker(t, 30000, 120000),
			},
		},
	}
}

func randomGasPicker(t *testing.T, min, max uint64) *config.GasPicker {
	t.Helper()
	var gp config.GasPicker
	require.NoError(t, gp.UnmarshalJSON(fmt.Appendf(nil, `{"Name":"random","Min":%d,"Max":%d}`, min, max)))
	return &gp
}

// gasSeq drains n txs from a freshly-built generator and returns the gas value
// of each — the RNG-driven, seed-determined output we replay against.
func gasSeq(t *testing.T, seed uint64, n int) []uint64 {
	t.Helper()
	gen, err := generator.NewConfigBasedGenerator(seededConfig(t, seed))
	require.NoError(t, err)
	txs := gen.GenerateN(n)
	require.Len(t, txs, n)
	out := make([]uint64, n)
	for i, tx := range txs {
		out[i] = tx.EthTx.Gas()
	}
	return out
}

// Same seed + config => byte-identical gas draw sequence across two runs.
func TestSeededRunReplaysIdentically(t *testing.T) {
	require.Equal(t, gasSeq(t, 123, 200), gasSeq(t, 123, 200))
}

// Different seeds must diverge (otherwise the seed is ignored).
func TestDifferentSeedsDiverge(t *testing.T) {
	require.NotEqual(t, gasSeq(t, 1, 200), gasSeq(t, 2, 200))
}

// Worker-count independence: the produced gas values for a given seed do not
// depend on how many worker goroutines consume the generator. We drain the same
// seeded generator serially (1 worker) and via many concurrent goroutines
// (N workers) and assert the produced multisets are identical. Streams are
// keyed by logical config id, not a live-goroutine counter, so the set of draws
// a seed yields is invariant to --workers.
func TestWorkerCountIndependence(t *testing.T) {
	const seed, total = 99, 600

	serial := gasSeq(t, seed, total)

	for _, workers := range []int{2, 4, 8} {
		gen, err := generator.NewConfigBasedGenerator(seededConfig(t, seed))
		require.NoError(t, err)

		got := make([]uint64, total)
		var wg sync.WaitGroup
		var mu sync.Mutex
		next := 0
		for w := 0; w < workers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					mu.Lock()
					if next >= total {
						mu.Unlock()
						return
					}
					idx := next
					next++
					tx, ok := gen.Generate()
					mu.Unlock()
					require.True(t, ok)
					got[idx] = tx.EthTx.Gas()
				}
			}()
		}
		wg.Wait()

		require.ElementsMatch(t, serial, got,
			"workers=%d produced a different multiset of gas draws than serial", workers)
	}
}
