package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestScenarioValidateSizeBuckets: a negative pad length (makeslice panic on the
// hot path) and an over-cap pad length (OOM risk) are both rejected; a valid
// histogram, the cap boundary, and an empty/nil bucket list pass.
func TestScenarioValidateSizeBuckets(t *testing.T) {
	t.Parallel()
	t.Run("negative rejected", func(t *testing.T) {
		s := Scenario{Name: "s", SizeDistribution: &Distribution{}, SizeBuckets: []int{0, -1}}
		require.ErrorContains(t, s.Validate(), "negative")
	})
	t.Run("over cap rejected", func(t *testing.T) {
		s := Scenario{Name: "s", SizeDistribution: &Distribution{}, SizeBuckets: []int{maxCalldataPadBytes + 1}}
		require.ErrorContains(t, s.Validate(), "cap")
	})
	t.Run("valid accepted", func(t *testing.T) {
		s := Scenario{Name: "s", SizeDistribution: &Distribution{}, SizeBuckets: []int{0, 64, maxCalldataPadBytes}}
		require.NoError(t, s.Validate())
	})
	t.Run("empty accepted", func(t *testing.T) {
		require.NoError(t, (&Scenario{Name: "s"}).Validate())
	})
}

// TestScenarioValidateRecordCountCap: the keyspace is capped because the zipfian
// sampler precomputes zeta in O(n) under a mutex on the generator's only
// goroutine, so a stray extra digit stalls the run rather than failing it.
func TestScenarioValidateRecordCountCap(t *testing.T) {
	t.Parallel()
	at := Scenario{Name: "s", KeyDistribution: &Distribution{}, RecordCount: maxRecordCount}
	require.NoError(t, at.Validate())
	over := Scenario{Name: "s", KeyDistribution: &Distribution{}, RecordCount: maxRecordCount + 1}
	require.ErrorContains(t, over.Validate(), "cap")
}

// TestScenarioValidateAxisPairing: an axis needs a sampler and a space to sample.
// Configured with only one half it silently runs the baseline instead of the
// experiment, so each direction is rejected.
func TestScenarioValidateAxisPairing(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		scenario Scenario
		wantErr  string
	}{
		"key distribution without a keyspace": {
			Scenario{Name: "s", KeyDistribution: &Distribution{}},
			"recordCount is 0",
		},
		"keyspace without a key distribution": {
			Scenario{Name: "s", RecordCount: 1000},
			"no keyDistribution",
		},
		"size distribution without buckets": {
			Scenario{Name: "s", SizeDistribution: &Distribution{}},
			"sizeBuckets is empty",
		},
		"buckets without a size distribution": {
			Scenario{Name: "s", SizeBuckets: []int{0, 32}},
			"no sizeDistribution",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			require.ErrorContains(t, tc.scenario.Validate(), tc.wantErr)
		})
	}

	both := Scenario{
		Name:             "s",
		KeyDistribution:  &Distribution{},
		RecordCount:      1000,
		SizeDistribution: &Distribution{},
		SizeBuckets:      []int{0, 32},
	}
	require.NoError(t, both.Validate())
}

// TestScenarioValidateOperationsPresentButEmpty: an explicit all-zero mix is a
// misconfiguration, not the default. Omitting the field is the default, and the
// picker's zero-total guard stays a safety net for that case rather than a
// swallower of this one.
func TestScenarioValidateOperationsPresentButEmpty(t *testing.T) {
	t.Parallel()
	empty := Scenario{Name: "storagerw", Operations: OperationMix{}}
	require.ErrorContains(t, empty.Validate(), "every weight is 0")

	require.NoError(t, (&Scenario{Name: "storagerw"}).Validate())
	require.NoError(t, (&Scenario{Name: "storagerw", Operations: OperationMix{OpRmw: 1}}).Validate())
}

// TestValidateScenariosReportsOffendingScenario: validation runs across every
// scenario and names the one that failed, so an operator can find it in a
// multi-scenario profile.
func TestValidateScenariosReportsOffendingScenario(t *testing.T) {
	t.Parallel()
	cfg := LoadConfig{Scenarios: []Scenario{
		{Name: "good"},
		{Name: "storagerw", Operations: OperationMix{}},
	}}
	require.ErrorContains(t, cfg.ValidateScenarios(), `scenario "storagerw"`)

	cfg.Scenarios[1].Operations = OperationMix{OpRead: 1}
	require.NoError(t, cfg.ValidateScenarios())
}
