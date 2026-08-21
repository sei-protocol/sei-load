package stats_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/sei-protocol/sei-load/stats"
	"github.com/stretchr/testify/require"
)

// TestFinalStatsScenarioOrderIsStable pins the scenario order in the printed
// report against Go's randomised map iteration. Two runs of one workload must
// stay comparable line by line.
func TestFinalStatsScenarioOrderIsStable(t *testing.T) {
	c := stats.NewCollector()
	for _, scenario := range []string{"storagerw", "erc721", "evmtransfer", "disperse", "erc20"} {
		c.RecordTransaction(scenario, "transfer", time.Millisecond, true)
	}
	logger := stats.NewLogger(c, time.Second, "", false)

	labels := func() []string { return reportLabels(t, logger) }

	first := labels()
	require.Subset(t, first, []string{"disperse", "erc20", "erc721", "evmtransfer", "storagerw"})
	for i := 0; i < 20; i++ {
		require.Equal(t, first, labels())
	}
	require.Less(t, slices.Index(first, "disperse"), slices.Index(first, "erc20"))
	require.Less(t, slices.Index(first, "erc721"), slices.Index(first, "storagerw"))
}

// TestOperationReportOrderIsStable pins the printed report against Go's
// randomised map iteration. Two runs of one workload must stay comparable line
// by line. It drives what LogFinalStats calls: BuildFinalStats().String().
func TestOperationReportOrderIsStable(t *testing.T) {
	c := stats.NewCollector()
	for _, op := range []string{"write", "read", "rmw"} {
		c.RecordTransaction("storagerw", op, time.Millisecond, true)
	}
	for _, scenario := range []string{"evmtransfer", "erc721", "disperse"} {
		c.RecordTransaction(scenario, "transfer", time.Millisecond, true)
	}

	logger := stats.NewLogger(c, time.Second, "", false)

	labels := func() []string { return reportLabels(t, logger) }

	first := labels()
	require.Subset(t, first, []string{"storagerw/read", "storagerw/rmw", "storagerw/write"})
	for i := 0; i < 20; i++ {
		require.Equal(t, first, labels())
	}
	require.Less(t,
		slices.Index(first, "storagerw/read"),
		slices.Index(first, "storagerw/rmw"),
		"operations sort by name within a scenario")
	require.Less(t,
		slices.Index(first, "disperse/transfer"),
		slices.Index(first, "storagerw/read"),
		"scenarios sort before their operations are compared")
}

// reportLabels returns the label before each ":" in the printed report, in order.
// Runtime and Avg TPS derive from time.Since(StartTime) and move between calls by
// design, so only the label sequence is comparable.
func reportLabels(t *testing.T, logger *stats.Logger) []string {
	t.Helper()
	var out []string
	for _, line := range strings.Split(logger.BuildFinalStats().String(), "\n") {
		if label, _, found := strings.Cut(strings.TrimSpace(line), ":"); found {
			out = append(out, label)
		}
	}
	return out
}
