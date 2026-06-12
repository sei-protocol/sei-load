package config_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sei-protocol/sei-load/config"
	"github.com/sei-protocol/sei-load/utils/rng"
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

// distribution unmarshals a fresh Distribution from a JSON fragment.
func distribution(t *testing.T, raw string) *config.Distribution {
	t.Helper()
	var d config.Distribution
	require.NoError(t, d.UnmarshalJSON([]byte(raw)))
	return &d
}

// sample binds d to stream and pulls count draws over keyspace n.
func sample(t *testing.T, d *config.Distribution, s *rng.Stream, n uint64, count int) []uint64 {
	t.Helper()
	d.SetStream(s)
	out := make([]uint64, count)
	for i := range out {
		v, err := d.SampleIndex(n)
		require.NoError(t, err)
		require.Less(t, v, n, "draw out of range [0, n)")
		out[i] = v
	}
	return out
}

// TestSampleIndexEmptyKeyspace: a zero keyspace is a caller error, not a silent
// zero, for the real samplers (the zero-value Distribution still returns 0).
func TestSampleIndexEmptyKeyspace(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{`{"Name":"uniform"}`, `{"Name":"zipfian","theta":0.9}`} {
		_, err := distribution(t, raw).SampleIndex(0)
		require.Error(t, err, raw)
	}
}

// TestSampleIndexDeterminism: same seed + same stream id => identical draw
// sequence, for both samplers. This is the per-stream reproducibility contract.
func TestSampleIndexDeterminism(t *testing.T) {
	t.Parallel()
	const seed, n, count = 99, 1000, 256
	for _, raw := range []string{`{"Name":"uniform"}`, `{"Name":"zipfian","theta":0.8}`} {
		a := sample(t, distribution(t, raw), rng.NewSource(seed).Stream(rng.KeyDistributionStream(0)), n, count)
		b := sample(t, distribution(t, raw), rng.NewSource(seed).Stream(rng.KeyDistributionStream(0)), n, count)
		require.Equal(t, a, b, "same seed must reproduce the draw sequence: %s", raw)
	}
}

// TestSampleIndexSeededDiffersFromUnseeded guards the binding the way
// TestRandomGasPickerStreamSeeds does for gas: a bound sampler draws
// seed-determined values that differ from the unseeded global RNG path. If a
// refactor silently broke the binding, the seeded and unseeded sequences would
// match by accident only with probability ~0.
func TestSampleIndexSeededDiffersFromUnseeded(t *testing.T) {
	t.Parallel()
	const seed, n, count = 7, 1000, 128
	for _, raw := range []string{`{"Name":"uniform"}`, `{"Name":"zipfian","theta":0.8}`} {
		seeded := sample(t, distribution(t, raw), rng.NewSource(seed).Stream(rng.KeyDistributionStream(0)), n, count)
		unseeded := sample(t, distribution(t, raw), nil, n, count)
		require.NotEqual(t, seeded, unseeded, "seeded draws must differ from the unseeded global RNG: %s", raw)
	}
}

// TestUniformIsUniform: a chi-square goodness-of-fit test over evenly-sized
// buckets. With B buckets and N draws the statistic should sit well under the
// upper critical value; a badly skewed "uniform" would blow far past it.
func TestUniformIsUniform(t *testing.T) {
	t.Parallel()
	const n, buckets, perBucket = 1000, 20, 5000
	const draws = buckets * perBucket // 100k draws, expected 5k per bucket.

	got := sample(t, distribution(t, `{"Name":"uniform"}`), rng.NewSource(1).Stream("x"), n, draws)
	counts := make([]float64, buckets)
	width := uint64(n / buckets)
	for _, v := range got {
		counts[v/width]++
	}
	expected := float64(draws) / buckets
	var chi2 float64
	for _, c := range counts {
		d := c - expected
		chi2 += d * d / expected
	}
	// df = 19; chi-square upper ~0.1% critical value is ~43.8. A uniform draw
	// clears this comfortably; the loose bound keeps the test non-flaky.
	require.Less(t, chi2, 50.0, "uniform draws failed chi-square (chi2=%.2f)", chi2)
}

// TestZipfianSkewRisesWithTheta: the mass on the hottest top-k indices must
// increase monotonically with theta, and theta->0 must approach the uniform
// baseline. This is the defining property of the generator.
func TestZipfianSkewRisesWithTheta(t *testing.T) {
	t.Parallel()
	const n, draws, topK = 10000, 100000, 100 // top 1% of the keyspace.

	topKMass := func(theta float64) float64 {
		raw := fmt.Sprintf(`{"Name":"zipfian","theta":%v}`, theta)
		got := sample(t, distribution(t, raw), rng.NewSource(5).Stream("x"), n, draws)
		var hot int
		for _, v := range got {
			if v < topK {
				hot++
			}
		}
		return float64(hot) / float64(draws)
	}

	uniformBaseline := float64(topK) / float64(n) // 0.01
	m0 := topKMass(0.0)
	m5 := topKMass(0.5)
	m9 := topKMass(0.9)

	require.InDelta(t, uniformBaseline, m0, 0.01, "theta=0 should approximate uniform")
	require.Greater(t, m5, m0, "skew must rise from theta=0 to 0.5 (m0=%.4f m5=%.4f)", m0, m5)
	require.Greater(t, m9, m5, "skew must rise from theta=0.5 to 0.9 (m5=%.4f m9=%.4f)", m5, m9)
	require.Greater(t, m9, 0.1, "theta=0.9 should concentrate >10%% on the top 1%% (m9=%.4f)", m9)
}

// TestZipfianInitCostBounded: precomputing zeta for a 1e6 keyspace must finish
// quickly (it is O(n), done once), and subsequent draws must be O(1) — proven
// here by the whole operation, init plus 1000 draws, staying well under budget.
func TestZipfianInitCostBounded(t *testing.T) {
	t.Parallel()
	const n = 1_000_000
	d := distribution(t, `{"Name":"zipfian","theta":0.99}`)
	d.SetStream(rng.NewSource(1).Stream("x"))

	start := time.Now()
	for i := 0; i < 1000; i++ {
		v, err := d.SampleIndex(n)
		require.NoError(t, err)
		require.Less(t, v, uint64(n))
	}
	elapsed := time.Since(start)
	require.Less(t, elapsed, 2*time.Second, "zipfian init+draws for n=1e6 too slow: %s", elapsed)
}

// TestZipfianNoNaNAcrossThetaRange: across the valid theta range and small
// edge-case keyspaces, every draw must be a valid in-range index — guarding the
// numerical-stability constants (eta, alpha) from producing NaN/overflow.
func TestZipfianNoNaNAcrossThetaRange(t *testing.T) {
	t.Parallel()
	for _, theta := range []float64{0.0, 0.001, 0.5, 0.9, 0.99, 0.999} {
		for _, n := range []uint64{2, 3, 100, 1000} {
			raw := fmt.Sprintf(`{"Name":"zipfian","theta":%v}`, theta)
			d := distribution(t, raw)
			d.SetStream(rng.NewSource(1).Stream("x"))
			for i := 0; i < 100; i++ {
				v, err := d.SampleIndex(n)
				require.NoError(t, err)
				// v is a uint64 index; the in-range check is the real guard that
				// the internal zeta/eta math never produced a bad (NaN-derived) draw.
				require.Less(t, v, n, "theta=%v n=%d produced out-of-range index", theta, n)
			}
		}
	}
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
