package config_test

import (
	"fmt"
	"testing"

	"github.com/sei-protocol/sei-load/config"
	"github.com/sei-protocol/sei-load/utils/rng"
	"github.com/stretchr/testify/require"
)

func TestGasPicker(t *testing.T) {
	t.Parallel()
	t.Run("empty", func(t *testing.T) {
		var subject config.GasPicker
		require.NoError(t, subject.UnmarshalJSON([]byte(`{}`)))
		gas, err := subject.GenerateGas()
		require.NoError(t, err)
		require.Zero(t, gas)
	})
	t.Run("fixed", func(t *testing.T) {
		var subject config.GasPicker
		require.NoError(t, subject.UnmarshalJSON([]byte(`{"Name":"fixed","Gas":21000}`)))
		gas, err := subject.GenerateGas()
		require.NoError(t, err)
		require.Equal(t, uint64(21000), gas)
	})
	t.Run("random", func(t *testing.T) {
		var subject config.GasPicker
		require.NoError(t, subject.UnmarshalJSON([]byte(`{"Name":"random","Min":20000,"Max":30000}`)))
		gas, err := subject.GenerateGas()
		require.NoError(t, err)
		require.GreaterOrEqual(t, gas, uint64(20000))
		require.LessOrEqual(t, gas, uint64(30000))
	})
	t.Run("unknown", func(t *testing.T) {
		var subject config.GasPicker
		require.Error(t, subject.UnmarshalJSON([]byte(`{"Name":"unknown"}`)))
	})
}

// randomPicker unmarshals a fresh random gas picker over [min,max].
func randomPicker(t *testing.T, min, max uint64) *config.GasPicker {
	t.Helper()
	var gp config.GasPicker
	require.NoError(t, gp.UnmarshalJSON(fmt.Appendf(nil, `{"Name":"random","Min":%d,"Max":%d}`, min, max)))
	return &gp
}

// drawN binds the picker to the given stream and pulls n draws.
func drawN(t *testing.T, gp *config.GasPicker, s *rng.Stream, n int) []uint64 {
	t.Helper()
	gp.SetStream(s)
	out := make([]uint64, n)
	for i := range out {
		v, err := gp.GenerateGas()
		require.NoError(t, err)
		out[i] = v
	}
	return out
}

// TestRandomGasPickerStreamSeeds guards the binding contract: after SetStream a
// random picker draws seed-determined values. Two same-seed builds must match
// AND differ from an unseeded build. This fails loudly if a refactor (e.g. a
// deep copy of config.Scenario) breaks the pointer aliasing bindGasStreams
// relies on, so the binding silently reverting to the global RNG cannot pass.
func TestRandomGasPickerStreamSeeds(t *testing.T) {
	const seed, n = 17, 64

	seededA := drawN(t, randomPicker(t, 20000, 30000), rng.NewSource(seed).Stream("gas:0:base"), n)
	seededB := drawN(t, randomPicker(t, 20000, 30000), rng.NewSource(seed).Stream("gas:0:base"), n)
	require.Equal(t, seededA, seededB, "same seed must reproduce the draw sequence")

	unseeded := drawN(t, randomPicker(t, 20000, 30000), nil, n)
	require.NotEqual(t, seededA, unseeded, "seeded draws must differ from the unseeded global RNG")
}

// TestSetStreamNoOpsForFixedAndEmpty confirms fixed/empty pickers ignore
// SetStream (they have no randomness to seed) rather than erroring.
func TestSetStreamNoOpsForFixedAndEmpty(t *testing.T) {
	stream := rng.NewSource(1).Stream("gas:0:base")

	var fixed config.GasPicker
	require.NoError(t, fixed.UnmarshalJSON([]byte(`{"Name":"fixed","Gas":21000}`)))
	fixed.SetStream(stream)
	gas, err := fixed.GenerateGas()
	require.NoError(t, err)
	require.Equal(t, uint64(21000), gas)

	var empty config.GasPicker
	require.NoError(t, empty.UnmarshalJSON([]byte(`{}`)))
	empty.SetStream(stream)
	gas, err = empty.GenerateGas()
	require.NoError(t, err)
	require.Zero(t, gas)
}
