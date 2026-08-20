package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestScenarioValidateSizeBuckets: a negative pad length would panic makeslice
// on the hot path and an over-cap length risks an OOM, so both are rejected at
// load. A valid histogram, the cap boundary, and an absent bucket list pass.
func TestScenarioValidateSizeBuckets(t *testing.T) {
	t.Parallel()

	t.Run("negative rejected", func(t *testing.T) {
		t.Parallel()
		s := Scenario{Name: "s", SizeBuckets: []int{0, -1}}
		require.ErrorContains(t, s.Validate(), "negative")
	})

	t.Run("over cap rejected", func(t *testing.T) {
		t.Parallel()
		s := Scenario{Name: "s", SizeBuckets: []int{maxCalldataPadBytes + 1}}
		require.ErrorContains(t, s.Validate(), "cap")
	})

	t.Run("valid accepted", func(t *testing.T) {
		t.Parallel()
		s := Scenario{Name: "s", SizeBuckets: []int{0, 64, maxCalldataPadBytes}}
		require.NoError(t, s.Validate())
	})

	t.Run("absent accepted", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, (&Scenario{Name: "s"}).Validate())
	})
}

// TestValidateScenariosReportsOffendingScenario: validation runs across every
// scenario and the error names the one that failed, so an operator can find it
// in a multi-scenario profile.
func TestValidateScenariosReportsOffendingScenario(t *testing.T) {
	t.Parallel()

	cfg := LoadConfig{Scenarios: []Scenario{
		{Name: "good", SizeBuckets: []int{0, 32}},
		{Name: "bad", SizeBuckets: []int{-1}},
	}}
	require.ErrorContains(t, cfg.ValidateScenarios(), `scenario "bad"`)

	cfg.Scenarios[1].SizeBuckets = []int{16}
	require.NoError(t, cfg.ValidateScenarios())
}
