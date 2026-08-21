package stats_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/sei-protocol/sei-load/stats"
	"github.com/stretchr/testify/require"
)

// TestPerOperationLatencySeparates is the claim the dimension exists to support:
// one slow operation must show as slow on its own line, not be averaged into the
// scenario. The pooled percentile cannot answer which operation degraded.
func TestPerOperationLatencySeparates(t *testing.T) {
	c := stats.NewCollector()
	for i := 0; i < 100; i++ {
		c.RecordTransaction("storagerw", "read", 10*time.Millisecond, true)
		c.RecordTransaction("storagerw", "rmw", 500*time.Millisecond, true)
	}

	got := c.GetStats().Operations
	read := got[stats.OperationKey{Scenario: "storagerw", Operation: "read"}]
	rmw := got[stats.OperationKey{Scenario: "storagerw", Operation: "rmw"}]

	require.Equal(t, uint64(100), read.Count)
	require.Equal(t, uint64(100), rmw.Count)
	require.Equal(t, 10*time.Millisecond, read.P99Latency)
	require.Equal(t, 500*time.Millisecond, rmw.P99Latency)
}

// TestOperationsDoNotConflateAcrossScenarios covers the reason the key carries
// both fields: two scenarios issuing the same call share an operation name, so
// the operation alone is not a key.
func TestOperationsDoNotConflateAcrossScenarios(t *testing.T) {
	c := stats.NewCollector()
	c.RecordTransaction("erc20", "erc20_transfer", 5*time.Millisecond, true)
	c.RecordTransaction("erc20noop", "erc20_transfer", 5*time.Millisecond, true)

	got := c.GetStats().Operations
	require.Len(t, got, 2)
	require.Equal(t, uint64(1), got[stats.OperationKey{Scenario: "erc20", Operation: "erc20_transfer"}].Count)
	require.Equal(t, uint64(1), got[stats.OperationKey{Scenario: "erc20noop", Operation: "erc20_transfer"}].Count)
}

// TestFormatStatsOrderIsStable pins the report against Go's randomised map
// iteration. A run summary that reorders its own lines cannot be diffed against
// another run, which is what this tool produces summaries for.
func TestFormatStatsOrderIsStable(t *testing.T) {
	c := stats.NewCollector()
	for _, op := range []string{"write", "read", "rmw"} {
		c.RecordTransaction("storagerw", op, time.Millisecond, true)
	}
	for _, scenario := range []string{"evmtransfer", "erc721", "disperse"} {
		c.RecordTransaction(scenario, "transfer", time.Millisecond, true)
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

// TestRecordTransactionCountsFailures: a rejected tx still happened, so it
// counts, but it contributes no latency sample.
func TestRecordTransactionCountsFailures(t *testing.T) {
	c := stats.NewCollector()
	c.RecordTransaction("storagerw", "rmw", time.Second, false)

	op := c.GetStats().Operations[stats.OperationKey{Scenario: "storagerw", Operation: "rmw"}]
	require.Equal(t, uint64(1), op.Count)
	require.Zero(t, op.SampleCount)
}
