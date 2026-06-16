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
		s := Scenario{Name: "s", SizeBuckets: []int{0, -1}}
		require.ErrorContains(t, s.Validate(), "negative")
	})
	t.Run("over cap rejected", func(t *testing.T) {
		s := Scenario{Name: "s", SizeBuckets: []int{maxCalldataPadBytes + 1}}
		require.ErrorContains(t, s.Validate(), "cap")
	})
	t.Run("valid accepted", func(t *testing.T) {
		s := Scenario{Name: "s", SizeBuckets: []int{0, 64, maxCalldataPadBytes}}
		require.NoError(t, s.Validate())
	})
	t.Run("empty accepted", func(t *testing.T) {
		require.NoError(t, (&Scenario{Name: "s"}).Validate())
	})
}
