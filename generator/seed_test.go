package generator_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sei-protocol/sei-load/config"
	"github.com/sei-protocol/sei-load/generator"
	"github.com/sei-protocol/sei-load/generator/scenarios"
	"github.com/sei-protocol/sei-load/types"
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
				Name:            scenarios.EVMTransfer,
				Weight:          2,
				Accounts:        &config.AccountConfig{Accounts: 10, NewAccountRate: 0.5},
				GasPicker:       randomGasPicker(t, 21000, 90000),
				GasTipCapPicker: randomGasPicker(t, 1_000_000_000, 3_000_000_000),
				GasFeeCapPicker: randomGasPicker(t, 100_000_000_000, 300_000_000_000),
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

// gasDraw is the seed-determined gas output of one tx across all three gas
// streams (base/tip/feecap), so reproducibility assertions cover every stream
// bindGasStreams binds, not just the base picker.
type gasDraw struct {
	gas    uint64
	tip    int64
	feeCap int64
}

func draw(tx *types.LoadTx) gasDraw {
	return gasDraw{
		gas:    tx.EthTx.Gas(),
		tip:    tx.EthTx.GasTipCap().Int64(),
		feeCap: tx.EthTx.GasFeeCap().Int64(),
	}
}

// gasSeq drains n txs from a freshly-built generator and returns each tx's
// seed-determined gas draw — the RNG-driven output we replay against.
func gasSeq(t *testing.T, seed uint64, n int) []gasDraw {
	t.Helper()
	gen, err := generator.NewConfigBasedGenerator(seededConfig(t, seed))
	require.NoError(t, err)
	txs := generator.GenerateN(gen, n)
	require.Len(t, txs, n)
	out := make([]gasDraw, n)
	for i, tx := range txs {
		out[i] = draw(tx)
	}
	return out
}

// Same seed + config => identical ordered gas draw sequence at a single worker
// (GenerateN drains serially). This pins the ordered guarantee.
func TestSeededRunReplaysIdentically(t *testing.T) {
	require.Equal(t, gasSeq(t, 123, 200), gasSeq(t, 123, 200))
}

// TestSingleWorkerOrderedReplay pins the real ordered guarantee: at one worker
// the ordered draw/tx sequence is reproducible across two same-seed runs. (The
// multiset — not the order — is what survives above one worker; see
// TestWorkerCountMultisetInvariant.)
func TestSingleWorkerOrderedReplay(t *testing.T) {
	const seed, total = 55, 300
	a := gasSeq(t, seed, total)
	b := gasSeq(t, seed, total)
	require.Equal(t, a, b, "single-worker ordered draw sequence is not reproducible")
}

// Different seeds must diverge (otherwise the seed is ignored).
func TestDifferentSeedsDiverge(t *testing.T) {
	require.NotEqual(t, gasSeq(t, 1, 200), gasSeq(t, 2, 200))
}

// columns transposes a slice of per-tx gas draws into three per-stream column
// slices. The contract guarantees per-*stream* multisets, not per-tx tuples:
// concurrent txs interleave their base/tip/feecap draws across three
// independently-locked streams, so tuples reassemble differently while each
// stream's column multiset is unchanged. We assert columns, never tuples.
func columns(draws []gasDraw) (gas []uint64, tip, feeCap []int64) {
	gas = make([]uint64, len(draws))
	tip = make([]int64, len(draws))
	feeCap = make([]int64, len(draws))
	for i, d := range draws {
		gas[i] = d.gas
		tip[i] = d.tip
		feeCap[i] = d.feeCap
	}
	return gas, tip, feeCap
}

// TestWorkerCountMultisetInvariant asserts the per-stream multiset guarantee,
// not ordered replay: each gas stream's column multiset that a seed yields does
// not depend on how many worker goroutines concurrently consume the generator.
// Streams are keyed by logical config id, not a live-goroutine counter, so the
// per-stream multiset is invariant to --workers.
//
// Two things make this test exercise the real contract rather than a stronger
// false one:
//
//   - gen.Generate() runs OUTSIDE the worker lock, so workers genuinely draw
//     concurrently and the three streams interleave. The lock guards only the
//     work-claim bookkeeping. (Run under -race; -count=10 guards against flake.)
//   - We assert each column's multiset independently via ElementsMatch, NOT the
//     per-tx tuple. Tuples are not worker-invariant; columns are.
//
// Ordering within a column is deliberately NOT asserted; it is non-deterministic
// above one worker.
func TestWorkerCountMultisetInvariant(t *testing.T) {
	const seed, total = 99, 600

	serial := gasSeq(t, seed, total)
	wantGas, wantTip, wantFeeCap := columns(serial)

	for _, workers := range []int{2, 4, 8} {
		gen, err := generator.NewConfigBasedGenerator(seededConfig(t, seed))
		require.NoError(t, err)

		// Each worker collects into its own slice; we merge after the join so the
		// only shared mutable state under lock is the work-claim counter, and
		// Generate() itself runs unlocked and concurrently.
		var mu sync.Mutex
		remaining := total
		perWorker := make([][]gasDraw, workers)

		var wg sync.WaitGroup
		for w := 0; w < workers; w++ {
			wg.Add(1)
			go func(w int) {
				defer wg.Done()
				for {
					mu.Lock()
					if remaining <= 0 {
						mu.Unlock()
						return
					}
					remaining--
					mu.Unlock()

					tx, ok := gen.Generate()
					require.True(t, ok)
					perWorker[w] = append(perWorker[w], draw(tx))
				}
			}(w)
		}
		wg.Wait()

		got := make([]gasDraw, 0, total)
		for _, part := range perWorker {
			got = append(got, part...)
		}
		require.Len(t, got, total)

		gotGas, gotTip, gotFeeCap := columns(got)
		require.ElementsMatch(t, wantGas, gotGas,
			"workers=%d: gas-stream column multiset diverged from serial", workers)
		require.ElementsMatch(t, wantTip, gotTip,
			"workers=%d: tip-stream column multiset diverged from serial", workers)
		require.ElementsMatch(t, wantFeeCap, gotFeeCap,
			"workers=%d: feecap-stream column multiset diverged from serial", workers)
	}
}
