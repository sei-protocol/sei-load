package stats_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/sei-protocol/sei-load/stats"
	"github.com/stretchr/testify/require"
)

// TestFormatStatsScenarioOrderIsStable pins the report against Go's randomised
// map iteration. FormatStats walked TxCounts directly, so the scenario lines
// came out in a different order on each call, and two runs of one workload could
// not be diffed — which is what this tool writes a summary for.
func TestFormatStatsScenarioOrderIsStable(t *testing.T) {
	c := stats.NewCollector()
	for _, scenario := range []string{"storagerw", "erc721", "evmtransfer", "disperse", "erc20"} {
		c.RecordTransaction(scenario, time.Millisecond, true)
	}

	// Compare the label order, not the whole report: Avg TPS derives from
	// time.Since(StartTime) and moves between calls by design.
	labels := func() []string {
		got := c.GetStats()
		var out []string
		for _, line := range strings.Split(got.FormatStats(), "\n") {
			if label, _, found := strings.Cut(strings.TrimSpace(line), ":"); found {
				out = append(out, label)
			}
		}
		return out
	}

	first := labels()
	require.Subset(t, first, []string{"disperse", "erc20", "erc721", "evmtransfer", "storagerw"})
	for i := 0; i < 20; i++ {
		require.Equal(t, first, labels())
	}
	require.Less(t, slices.Index(first, "disperse"), slices.Index(first, "erc20"))
	require.Less(t, slices.Index(first, "erc721"), slices.Index(first, "storagerw"))
}
