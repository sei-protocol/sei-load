package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sei-protocol/sei-load/config"
)

// TestOperationMixEmptyFallsBackToRmw: a zero-weight mix selects rmw, the
// default, rather than dividing by a zero total.
func TestOperationMixEmptyFallsBackToRmw(t *testing.T) {
	t.Parallel()
	var m config.OperationMix
	rng := newTestRng(1)
	for i := 0; i < 100; i++ {
		require.Equal(t, config.OpRmw, m.Select(rng))
	}
}

// TestOperationMixEmptyDrawsNoRandomness: the zero-weight fallback returns
// before touching the RNG, so an unconfigured mix cannot perturb other draws
// that share the same stream.
func TestOperationMixEmptyDrawsNoRandomness(t *testing.T) {
	t.Parallel()
	var m config.OperationMix
	rng := newTestRng(5)
	for i := 0; i < 100; i++ {
		m.Select(rng)
	}
	require.Equal(t, newTestRng(5).Uint64(), rng.Uint64())
}

// TestOperationMixHonorsWeights: a single-weighted op is selected exclusively,
// and a balanced mix reaches all three.
func TestOperationMixHonorsWeights(t *testing.T) {
	t.Parallel()

	t.Run("single weight is exclusive", func(t *testing.T) {
		t.Parallel()
		m := config.OperationMix{Read: 1}
		rng := newTestRng(1)
		for i := 0; i < 100; i++ {
			require.Equal(t, config.OpRead, m.Select(rng))
		}
	})

	t.Run("balanced mix reaches every op", func(t *testing.T) {
		t.Parallel()
		m := config.OperationMix{Read: 1, Write: 1, Rmw: 1}
		rng := newTestRng(1)
		seen := map[config.Operation]int{}
		for i := 0; i < 3000; i++ {
			seen[m.Select(rng)]++
		}
		require.Positive(t, seen[config.OpRead])
		require.Positive(t, seen[config.OpWrite])
		require.Positive(t, seen[config.OpRmw])
	})
}

// TestOperationMixApproximatesWeights: over many draws the selection converges
// on the configured proportions, which is the property a weighted workload
// depends on.
func TestOperationMixApproximatesWeights(t *testing.T) {
	t.Parallel()
	const draws = 100_000
	m := config.OperationMix{Rmw: 5, Read: 3, Write: 2}
	rng := newTestRng(9)

	seen := map[config.Operation]int{}
	for i := 0; i < draws; i++ {
		seen[m.Select(rng)]++
	}

	// Total weight is 10, so each op should land within a fraction of a point of
	// its own weight once scaled back. The tolerance is loose enough not to flake
	// and tight enough to catch a mis-ordered comparison chain.
	for op, wantWeight := range map[config.Operation]float64{
		config.OpRmw:   5,
		config.OpRead:  3,
		config.OpWrite: 2,
	} {
		got := float64(seen[op]) / draws * 10
		require.InDelta(t, wantWeight, got, 0.15, "operation %d proportion", op)
	}
}

// TestOperationMixDeterminism: the same seed reproduces the selection sequence,
// so a run is repeatable when the RNG is seeded.
func TestOperationMixDeterminism(t *testing.T) {
	t.Parallel()
	draw := func() []config.Operation {
		m := config.OperationMix{Read: 2, Write: 3, Rmw: 5}
		rng := newTestRng(99)
		out := make([]config.Operation, 256)
		for i := range out {
			out[i] = m.Select(rng)
		}
		return out
	}
	require.Equal(t, draw(), draw())
}
