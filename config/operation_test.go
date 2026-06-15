package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sei-protocol/sei-load/config"
	"github.com/sei-protocol/sei-load/utils/rng"
)

// TestOperationMixEmptyFallsBackToRmw: a zero-weight (or nil-equivalent) mix
// selects rmw, the scaffold default, rather than panicking on a zero total.
func TestOperationMixEmptyFallsBackToRmw(t *testing.T) {
	t.Parallel()
	var m config.OperationMix
	m.SetStream(rng.NewSource(1).Stream(rng.OpDistributionStream(0)))
	for i := 0; i < 100; i++ {
		require.Equal(t, config.OpRmw, m.Select())
	}
}

// TestOperationMixHonorsWeights: a single-weighted op is selected exclusively,
// and a balanced mix selects all three.
func TestOperationMixHonorsWeights(t *testing.T) {
	t.Parallel()
	t.Run("single", func(t *testing.T) {
		m := config.OperationMix{Read: 1}
		m.SetStream(rng.NewSource(1).Stream(rng.OpDistributionStream(0)))
		for i := 0; i < 100; i++ {
			require.Equal(t, config.OpRead, m.Select())
		}
	})
	t.Run("balanced", func(t *testing.T) {
		m := config.OperationMix{Read: 1, Write: 1, Rmw: 1}
		m.SetStream(rng.NewSource(1).Stream(rng.OpDistributionStream(0)))
		seen := map[config.Operation]int{}
		for i := 0; i < 3000; i++ {
			seen[m.Select()]++
		}
		require.Positive(t, seen[config.OpRead])
		require.Positive(t, seen[config.OpWrite])
		require.Positive(t, seen[config.OpRmw])
	})
}

// TestOperationMixDeterminism: same seed + same stream id reproduces the
// selection sequence.
func TestOperationMixDeterminism(t *testing.T) {
	t.Parallel()
	draw := func() []config.Operation {
		m := config.OperationMix{Read: 2, Write: 3, Rmw: 5}
		m.SetStream(rng.NewSource(99).Stream(rng.OpDistributionStream(0)))
		out := make([]config.Operation, 256)
		for i := range out {
			out[i] = m.Select()
		}
		return out
	}
	require.Equal(t, draw(), draw())
}
