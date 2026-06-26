package config_test

import (
	"fmt"
	mrand "math/rand/v2"
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
		gas, err := subject.GenerateGas(rng.NewSource(1).Rand("config:gas:test"))
		require.NoError(t, err)
		require.Zero(t, gas)
	})
	t.Run("fixed", func(t *testing.T) {
		var subject config.GasPicker
		require.NoError(t, subject.UnmarshalJSON([]byte(`{"Name":"fixed","Gas":21000}`)))
		gas, err := subject.GenerateGas(rng.NewSource(1).Rand("config:gas:test"))
		require.NoError(t, err)
		require.Equal(t, uint64(21000), gas)
	})
	t.Run("random", func(t *testing.T) {
		var subject config.GasPicker
		require.NoError(t, subject.UnmarshalJSON([]byte(`{"Name":"random","Min":20000,"Max":30000}`)))
		gas, err := subject.GenerateGas(rng.NewSource(1).Rand("config:gas:test"))
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

func drawN(t *testing.T, gp *config.GasPicker, rng *mrand.Rand, n int) []uint64 {
	t.Helper()
	out := make([]uint64, n)
	for i := range out {
		v, err := gp.GenerateGas(rng)
		require.NoError(t, err)
		out[i] = v
	}
	return out
}

// TestRandomGasPickerStreamSeeds guards the seeded-draw contract: a supplied
// PRNG yields deterministic values.
func TestRandomGasPickerStreamSeeds(t *testing.T) {
	const seed, n = 17, 64

	seededA := drawN(t, randomPicker(t, 20000, 30000), rng.NewSource(seed).Rand("gas:0:base"), n)
	seededB := drawN(t, randomPicker(t, 20000, 30000), rng.NewSource(seed).Rand("gas:0:base"), n)
	require.Equal(t, seededA, seededB, "same seed must reproduce the draw sequence")
}

// TestGenerateGasForFixedAndEmpty confirms fixed/empty pickers still work while
// requiring an explicit rng argument.
func TestGenerateGasForFixedAndEmpty(t *testing.T) {
	var fixed config.GasPicker
	require.NoError(t, fixed.UnmarshalJSON([]byte(`{"Name":"fixed","Gas":21000}`)))
	gas, err := fixed.GenerateGas(rng.NewSource(1).Rand("config:gas:test"))
	require.NoError(t, err)
	require.Equal(t, uint64(21000), gas)

	var empty config.GasPicker
	require.NoError(t, empty.UnmarshalJSON([]byte(`{}`)))
	gas, err = empty.GenerateGas(rng.NewSource(1).Rand("config:gas:test"))
	require.NoError(t, err)
	require.Zero(t, gas)
}
