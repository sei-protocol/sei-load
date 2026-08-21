package stats_test

import (
	"testing"
	"time"

	"github.com/sei-protocol/sei-load/stats"
	"github.com/stretchr/testify/require"
)

// TestPerOperationLatencySeparates is why the dimension exists: one slow
// operation keeps its own P99 instead of being averaged into the scenario. The
// pooled percentile cannot say which operation degraded.
func TestPerOperationLatencySeparates(t *testing.T) {
	c := stats.NewCollector()
	for i := 0; i < 100; i++ {
		c.RecordTransaction("storagerw", "read", 10*time.Millisecond, true)
		c.RecordTransaction("storagerw", "rmw", 500*time.Millisecond, true)
	}

	got := c.GetOperationStats()
	read := got[stats.OperationKey{Scenario: "storagerw", Operation: "read"}]
	rmw := got[stats.OperationKey{Scenario: "storagerw", Operation: "rmw"}]

	require.Equal(t, uint64(100), read.Count)
	require.Equal(t, uint64(100), rmw.Count)
	require.Equal(t, 10*time.Millisecond, read.P99Latency)
	require.Equal(t, 500*time.Millisecond, rmw.P99Latency)
}

// TestOperationsDoNotConflateAcrossScenarios covers the reason OperationKey
// carries both fields.
func TestOperationsDoNotConflateAcrossScenarios(t *testing.T) {
	c := stats.NewCollector()
	c.RecordTransaction("erc20", "erc20_transfer", 5*time.Millisecond, true)
	c.RecordTransaction("erc20noop", "erc20_transfer", 5*time.Millisecond, true)

	got := c.GetOperationStats()
	require.Len(t, got, 2)
	require.Equal(t, uint64(1), got[stats.OperationKey{Scenario: "erc20", Operation: "erc20_transfer"}].Count)
	require.Equal(t, uint64(1), got[stats.OperationKey{Scenario: "erc20noop", Operation: "erc20_transfer"}].Count)
}

// TestRecordTransactionCountsFailures shows that a rejected tx still counts but
// contributes no latency sample.
func TestRecordTransactionCountsFailures(t *testing.T) {
	c := stats.NewCollector()
	c.RecordTransaction("storagerw", "rmw", time.Second, false)

	op := c.GetOperationStats()[stats.OperationKey{Scenario: "storagerw", Operation: "rmw"}]
	require.Equal(t, uint64(1), op.Count)
	require.Zero(t, op.SampleCount)
}

// TestOperationWindowIsReported covers why the window is in the report at all.
// The retention limit is a sample count, not a duration, so a low-rate operation
// keeps samples reaching further back than a high-rate one. Two percentiles over
// different stretches of a run are not comparable, and the span is what lets a
// reader see that.
func TestOperationWindowIsReported(t *testing.T) {
	c := stats.NewCollector()
	for i := 0; i < 4; i++ {
		c.RecordTransaction("erc20", "erc20_transfer", 8*time.Millisecond, true)
		time.Sleep(5 * time.Millisecond)
	}

	op := c.GetOperationStats()[stats.OperationKey{Scenario: "erc20", Operation: "erc20_transfer"}]
	require.Equal(t, 4, op.SampleCount)
	require.Equal(t, uint64(4), op.Successes)
	require.Positive(t, op.Window, "the span the retained samples cover")
}

// TestOperationStatsSeparatesAttemptsFromSamples: Count is attempts, Successes
// is what could yield a latency, and SampleCount is what the retention limit
// kept. All three differ once an operation both fails and saturates.
func TestOperationStatsSeparatesAttemptsFromSamples(t *testing.T) {
	c := stats.NewCollector()
	c.RecordTransaction("storagerw", "rmw", time.Millisecond, true)
	c.RecordTransaction("storagerw", "rmw", time.Second, false)
	c.RecordTransaction("storagerw", "rmw", time.Second, false)

	op := c.GetOperationStats()[stats.OperationKey{Scenario: "storagerw", Operation: "rmw"}]
	require.Equal(t, uint64(3), op.Count, "attempts")
	require.Equal(t, uint64(1), op.Successes, "could yield a latency")
	require.Equal(t, 1, op.SampleCount, "retained")
}
