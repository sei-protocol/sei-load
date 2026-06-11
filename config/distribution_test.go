package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sei-protocol/sei-load/config"
	"github.com/stretchr/testify/require"
)

func TestDistribution(t *testing.T) {
	t.Parallel()
	t.Run("empty", func(t *testing.T) {
		var subject config.Distribution
		require.NoError(t, subject.UnmarshalJSON([]byte(`{}`)))
		require.Empty(t, subject.Name())
		idx, err := subject.SampleIndex(100)
		require.NoError(t, err)
		require.Zero(t, idx)
	})
	t.Run("uniform", func(t *testing.T) {
		var subject config.Distribution
		require.NoError(t, subject.UnmarshalJSON([]byte(`{"Name":"uniform"}`)))
		require.Equal(t, "uniform", subject.Name())
	})
	t.Run("zipfian", func(t *testing.T) {
		var subject config.Distribution
		require.NoError(t, subject.UnmarshalJSON([]byte(`{"Name":"zipfian","theta":0.99}`)))
		require.Equal(t, "zipfian", subject.Name())
	})
	t.Run("zipfian_theta_below_range", func(t *testing.T) {
		var subject config.Distribution
		require.Error(t, subject.UnmarshalJSON([]byte(`{"Name":"zipfian","theta":-0.1}`)))
	})
	t.Run("zipfian_theta_above_range", func(t *testing.T) {
		var subject config.Distribution
		require.Error(t, subject.UnmarshalJSON([]byte(`{"Name":"zipfian","theta":1.0}`)))
	})
	t.Run("unknown", func(t *testing.T) {
		var subject config.Distribution
		require.Error(t, subject.UnmarshalJSON([]byte(`{"Name":"weibull"}`)))
	})
}

// TestScenarioDistributionAdditive proves the new fields are additive: a profile
// carrying no distribution fields parses unchanged and round-trips without
// introducing any distribution keys.
func TestScenarioDistributionAdditive(t *testing.T) {
	t.Parallel()
	path := filepath.Join("..", "profiles", "conflict.json")
	original, err := os.ReadFile(path)
	require.NoError(t, err)

	var cfg config.LoadConfig
	require.NoError(t, json.Unmarshal(original, &cfg))

	for _, s := range cfg.Scenarios {
		require.Nil(t, s.KeyDistribution, "no distribution expected in baseline profile")
		require.Nil(t, s.SizeDistribution, "no distribution expected in baseline profile")
	}

	remarshaled, err := json.Marshal(cfg)
	require.NoError(t, err)
	require.NotContains(t, string(remarshaled), "keyDistribution")
	require.NotContains(t, string(remarshaled), "sizeDistribution")
}
