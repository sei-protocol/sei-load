package config_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sei-protocol/sei-load/config"
)

// tokenOperations is a basket shaped like an upcoming token workload: many
// operations, none of them read/write/rmw, declared in an order that is neither
// sorted nor reverse-sorted.
var tokenOperations = config.NewOperationSet(
	"erc20_transfer",
	"erc20_mint",
	"erc20_approve",
	"erc721_mint",
	"erc721_transfer",
	"erc20_burn",
	"erc721_approve",
	"erc20_transfer_from",
	"erc721_burn",
	"erc20_permit",
)

// uniformOracle predicts the selection sequence a set produces under equal
// weights, from the set's declared order and the same one-draw-per-selection
// RNG budget. It is independent of the picker: an implementation that walked the
// weight map, or that sorted the names, disagrees with it.
func uniformOracle(set *config.OperationSet, seed uint64, draws int) []string {
	names := set.Names()
	rng := newTestRng(seed)
	out := make([]string, draws)
	for i := range out {
		out[i] = names[rng.Uint64N(uint64(len(names)))]
	}
	return out
}

// equalMix weights every operation in set at 1.
func equalMix(set *config.OperationSet) config.OperationMix {
	names := set.Names()
	mix := make(config.OperationMix, len(names))
	for _, name := range names {
		mix[name] = 1
	}
	return mix
}

// selectN drains draws selections from a picker built for mix.
func selectN(set *config.OperationSet, mix config.OperationMix, seed uint64, draws int) []string {
	picker := set.Picker(mix)
	rng := newTestRng(seed)
	out := make([]string, draws)
	for i := range out {
		out[i] = picker.Select(rng)
	}
	return out
}

// marshalMix renders a mix to its wire form.
func marshalMix(t *testing.T, mix config.OperationMix) []byte {
	t.Helper()
	data, err := json.Marshal(mix)
	require.NoError(t, err)
	return data
}

// TestOperationMixAbsentSelectsDefault: an absent mix selects the set's first
// declared operation, rather than dividing by a zero total.
func TestOperationMixAbsentSelectsDefault(t *testing.T) {
	t.Parallel()
	picker := config.StorageRWOperations.Picker(nil)
	rng := newTestRng(1)
	for i := 0; i < 100; i++ {
		require.Equal(t, config.OpRmw, picker.Select(rng))
	}
}

// TestOperationMixAbsentDrawsNoRandomness: the zero-total fallback returns
// before touching the RNG, so an unconfigured mix cannot perturb other draws
// that share the same stream.
func TestOperationMixAbsentDrawsNoRandomness(t *testing.T) {
	t.Parallel()
	picker := config.StorageRWOperations.Picker(nil)
	rng := newTestRng(5)
	for i := 0; i < 100; i++ {
		picker.Select(rng)
	}
	require.Equal(t, newTestRng(5).Uint64(), rng.Uint64())
}

// TestOperationMixHonorsWeights: a single-weighted operation is selected
// exclusively, a balanced mix reaches every operation in its basket, and a zero
// weight is unreachable.
func TestOperationMixHonorsWeights(t *testing.T) {
	t.Parallel()

	t.Run("single weight is exclusive", func(t *testing.T) {
		t.Parallel()
		for _, got := range selectN(config.StorageRWOperations, config.OperationMix{config.OpRead: 1}, 1, 100) {
			require.Equal(t, config.OpRead, got)
		}
	})

	t.Run("balanced mix reaches every operation", func(t *testing.T) {
		t.Parallel()
		seen := map[string]int{}
		for _, got := range selectN(tokenOperations, equalMix(tokenOperations), 1, 10_000) {
			seen[got]++
		}
		for _, name := range tokenOperations.Names() {
			require.Positive(t, seen[name], "operation %q never selected", name)
		}
		require.Len(t, seen, len(tokenOperations.Names()), "selected an operation outside the basket")
	})

	t.Run("zero weight is never selected", func(t *testing.T) {
		t.Parallel()
		mix := config.OperationMix{config.OpRmw: 0, config.OpRead: 1, config.OpWrite: 1}
		for _, got := range selectN(config.StorageRWOperations, mix, 4, 1000) {
			require.NotEqual(t, config.OpRmw, got)
		}
	})
}

// TestOperationMixApproximatesWeights: over many draws the selection converges
// on the configured proportions, which is the property a weighted workload
// depends on.
func TestOperationMixApproximatesWeights(t *testing.T) {
	t.Parallel()
	const draws = 100_000
	want := map[string]float64{config.OpRmw: 5, config.OpRead: 3, config.OpWrite: 2}

	mix := config.OperationMix{}
	for name, weight := range want {
		mix[name] = uint64(weight)
	}

	seen := map[string]int{}
	for _, got := range selectN(config.StorageRWOperations, mix, 9, draws) {
		seen[got]++
	}

	// Total weight is 10, so each operation should land within a fraction of a
	// point of its own weight once scaled back. The tolerance is loose enough not
	// to flake and tight enough to catch a mis-ordered cumulative table.
	for name, wantWeight := range want {
		got := float64(seen[name]) / draws * 10
		require.InDelta(t, wantWeight, got, 0.15, "operation %q proportion", name)
	}
}

// TestOperationMixDeterminism: the same seed reproduces the selection sequence,
// so a run is repeatable when the RNG is seeded.
func TestOperationMixDeterminism(t *testing.T) {
	t.Parallel()
	mix := config.OperationMix{config.OpRead: 2, config.OpWrite: 3, config.OpRmw: 5}
	require.Equal(t,
		selectN(config.StorageRWOperations, mix, 99, 256),
		selectN(config.StorageRWOperations, mix, 99, 256))
}

// TestOperationMixDrawOrderFollowsTheSetNotTheMap guards the property that
// keying the weights by name puts at risk. Go map iteration order is unspecified
// and re-randomized per range, so a picker built by walking the mix would hand
// the same sub-range of a draw to a different operation on each rebuild. Every
// rebuild here must reproduce one sequence, and that sequence must be the one
// the set's declared order predicts, which also fails a picker that sorted the
// names instead.
func TestOperationMixDrawOrderFollowsTheSetNotTheMap(t *testing.T) {
	t.Parallel()

	for _, set := range []*config.OperationSet{config.StorageRWOperations, tokenOperations} {
		t.Run(set.Names()[0], func(t *testing.T) {
			t.Parallel()
			const rebuilds = 200
			const draws = 64
			want := uniformOracle(set, 2026, draws)

			for i := 0; i < rebuilds; i++ {
				var mix config.OperationMix
				require.NoError(t, json.Unmarshal(marshalMix(t, equalMix(set)), &mix))
				require.Equal(t, want, selectN(set, mix, 2026, draws), "rebuild %d", i)
			}
		})
	}
}

// TestOperationMixSelectAllocatesNothing: selection runs once per transaction,
// so the picker resolves the mix into its draw order ahead of the run and the
// draw itself walks that order without allocating.
func TestOperationMixSelectAllocatesNothing(t *testing.T) {
	picker := tokenOperations.Picker(equalMix(tokenOperations))
	rng := newTestRng(3)
	require.Zero(t, testing.AllocsPerRun(1000, func() { picker.Select(rng) }))
}

// TestOperationMixWireForm pins the JSON contract: an object of operation name
// to weight, decoded and re-encoded without gaining or losing a key.
func TestOperationMixWireForm(t *testing.T) {
	t.Parallel()

	t.Run("decodes the storagerw weights", func(t *testing.T) {
		t.Parallel()
		var mix config.OperationMix
		require.NoError(t, json.Unmarshal([]byte(`{"read":2,"write":3,"rmw":5}`), &mix))
		require.Equal(t, config.OperationMix{config.OpRead: 2, config.OpWrite: 3, config.OpRmw: 5}, mix)
	})

	t.Run("rejects a weight that is not a whole count", func(t *testing.T) {
		t.Parallel()
		for _, bad := range []string{`{"read":-1}`, `{"read":1.5}`, `{"read":"2"}`, `[1,2]`} {
			var mix config.OperationMix
			require.Error(t, json.Unmarshal([]byte(bad), &mix), "input %s", bad)
		}
	})

	t.Run("omits zero weights", func(t *testing.T) {
		t.Parallel()
		require.JSONEq(t, `{"write":1}`,
			string(marshalMix(t, config.OperationMix{config.OpRead: 0, config.OpWrite: 1})))
	})

	t.Run("an absent mix adds no key to a scenario", func(t *testing.T) {
		t.Parallel()
		encoded, err := json.Marshal(config.Scenario{Name: "storagerw", Weight: 1})
		require.NoError(t, err)
		require.NotContains(t, string(encoded), "operations")
	})

	t.Run("a scenario round-trips its weights", func(t *testing.T) {
		t.Parallel()
		const wire = `{"name":"storagerw","weight":1,"operations":{"read":2,"rmw":5,"write":3}}`
		var scenario config.Scenario
		require.NoError(t, json.Unmarshal([]byte(wire), &scenario))
		require.NoError(t, scenario.Validate())

		encoded, err := json.Marshal(scenario)
		require.NoError(t, err)
		require.JSONEq(t, wire, string(encoded))
	})
}

// TestOperationMixRejectedAtLoad: a profile weighting an operation the scenario
// does not declare fails validation at load, naming both, rather than surfacing
// later as a reweighted workload nobody asked for.
func TestOperationMixRejectedAtLoad(t *testing.T) {
	t.Parallel()

	t.Run("unknown operation", func(t *testing.T) {
		t.Parallel()
		cfg := decodeConfig(t, `{"scenarios":[{"name":"StorageRW","weight":1,"operations":{"rmw":1,"erc20_mint":2}}]}`)
		err := cfg.ValidateScenarios()
		require.ErrorContains(t, err, `scenario "StorageRW"`)
		require.ErrorContains(t, err, `unknown operation "erc20_mint"`)
		require.ErrorContains(t, err, "rmw, read, write")
	})

	t.Run("scenario with no operations", func(t *testing.T) {
		t.Parallel()
		cfg := decodeConfig(t, `{"scenarios":[{"name":"erc20","weight":1,"operations":{"rmw":1}}]}`)
		require.ErrorContains(t, cfg.ValidateScenarios(),
			`scenario "erc20": operations is set but this scenario has no operations`)
	})

	t.Run("every weight zero", func(t *testing.T) {
		t.Parallel()
		cfg := decodeConfig(t, `{"scenarios":[{"name":"storagerw","weight":1,"operations":{"read":0}}]}`)
		require.ErrorContains(t, cfg.ValidateScenarios(), "every weight is 0")
	})

	t.Run("weights sum past uint64", func(t *testing.T) {
		t.Parallel()
		cfg := decodeConfig(t, `{"scenarios":[{"name":"storagerw","weight":1,"operations":{"read":18446744073709551615,"write":1}}]}`)
		require.ErrorContains(t, cfg.ValidateScenarios(), "sum past uint64")
	})

	t.Run("a declared mix passes", func(t *testing.T) {
		t.Parallel()
		cfg := decodeConfig(t, `{"scenarios":[{"name":"storagerw","weight":1,"operations":{"read":2,"write":3,"rmw":5}}]}`)
		require.NoError(t, cfg.ValidateScenarios())
	})
}

// decodeConfig parses a profile the way loadConfig does.
func decodeConfig(t *testing.T, wire string) *config.LoadConfig {
	t.Helper()
	var cfg config.LoadConfig
	require.NoError(t, json.Unmarshal([]byte(wire), &cfg))
	return &cfg
}

// TestOperationSetRejectsAnUnusableDeclaration: an empty set leaves no default
// and a repeated name would claim two sub-ranges of one draw, so both fail where
// they are declared rather than at the first draw.
func TestOperationSetRejectsAnUnusableDeclaration(t *testing.T) {
	t.Parallel()
	require.Panics(t, func() { config.NewOperationSet() })
	require.Panics(t, func() { config.NewOperationSet(config.OpRead, config.OpWrite, config.OpRead) })
}

// TestOperationSetPickerRejectsAnUndeclaredName: Scenario.Validate is the gate a
// profile passes through, so a name that reaches Picker came from Go and is a
// programming error rather than operator input.
func TestOperationSetPickerRejectsAnUndeclaredName(t *testing.T) {
	t.Parallel()
	require.PanicsWithValue(t,
		`operation "erc20_mint" is not one of rmw, read, write`,
		func() { config.StorageRWOperations.Picker(config.OperationMix{"erc20_mint": 1}) })
}

// TestOperationSetNamesAreCallerOwned: Names hands back a copy, so a caller
// cannot reorder the draw order under a picker.
func TestOperationSetNamesAreCallerOwned(t *testing.T) {
	t.Parallel()
	names := config.StorageRWOperations.Names()
	require.Equal(t, []string{config.OpRmw, config.OpRead, config.OpWrite}, names)
	names[0] = "clobbered"
	require.Equal(t, config.OpRmw, config.StorageRWOperations.Names()[0])
}
