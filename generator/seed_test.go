package generator_test

import (
	"fmt"
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

