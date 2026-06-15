package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRecordScheduleLag_SamplesRoundTrip(t *testing.T) {
	c := NewCollector()
	c.RecordScheduleLag(1 * time.Millisecond)
	c.RecordScheduleLag(2 * time.Millisecond)
	c.RecordScheduleLag(3 * time.Millisecond)

	got := c.ScheduleLagSamples()
	require.ElementsMatch(t, []time.Duration{1 * time.Millisecond, 2 * time.Millisecond, 3 * time.Millisecond}, got)

	// Returned slice is a copy: mutating it must not affect the collector.
	got[0] = 999 * time.Second
	require.NotContains(t, c.ScheduleLagSamples(), 999*time.Second)
}

// Negative lags (scheduler/worker clock-read skew) clamp to zero so they cannot
// deflate the p99.
func TestRecordScheduleLag_NegativeClampsToZero(t *testing.T) {
	c := NewCollector()
	c.RecordScheduleLag(-5 * time.Millisecond)
	require.Equal(t, []time.Duration{0}, c.ScheduleLagSamples())
}

// The reservoir is bounded: recording far past the cap never grows the sample
// set beyond scheduleLagReservoirCap.
func TestRecordScheduleLag_ReservoirBounded(t *testing.T) {
	c := NewCollector()
	for i := range scheduleLagReservoirCap * 4 {
		c.RecordScheduleLag(time.Duration(i) * time.Nanosecond)
	}
	require.Len(t, c.ScheduleLagSamples(), scheduleLagReservoirCap)
}

// End-to-end through the collector: a known sample set yields the expected p99
// verdict, proving the record → sample → evaluate path agrees.
func TestRecordScheduleLag_FeedsVerdict(t *testing.T) {
	c := NewCollector()
	for range 99 {
		c.RecordScheduleLag(100 * time.Microsecond)
	}
	c.RecordScheduleLag(50 * time.Millisecond)

	v := EvaluateScheduleLag(ScheduleLagInputs{
		Samples: c.ScheduleLagSamples(), TargetTPS: 100, OpenLoop: true, Admitted: 100,
	})
	require.Equal(t, VerdictVoid, v.Verdict)
}

// The unsampled tail counters are exact, not reservoir-diluted: with the bound
// armed, every over-bound lag is counted regardless of reservoir replacement,
// and the max is the true max.
func TestRecordScheduleLag_UnsampledTailCounters(t *testing.T) {
	c := NewCollector()
	// Bound = 10% of 1/100 = 1ms (matches ScheduleLagBound(100, 0.10)).
	c.SetScheduleLagBound(ScheduleLagBound(100, 0.10))

	// Record far more than the reservoir cap so sampling is in play.
	const over = 50
	for range scheduleLagReservoirCap * 2 {
		c.RecordScheduleLag(100 * time.Microsecond) // under bound
	}
	for range over {
		c.RecordScheduleLag(5 * time.Millisecond) // over the 1ms bound
	}
	c.RecordScheduleLag(80 * time.Millisecond) // the max

	total, overBound, max := c.ScheduleLagTail()
	require.Equal(t, uint64(scheduleLagReservoirCap*2+over+1), total)
	require.Equal(t, uint64(over+1), overBound) // exact, not sampled
	require.Equal(t, 80*time.Millisecond, max)
}

// Without an armed bound the over-bound counter stays inert (ramped /
// closed-loop / no-λ runs), but the max is still tracked for diagnostics.
func TestRecordScheduleLag_OverBoundInertWhenUnset(t *testing.T) {
	c := NewCollector()
	c.RecordScheduleLag(500 * time.Millisecond)
	total, overBound, max := c.ScheduleLagTail()
	require.Equal(t, uint64(1), total)
	require.Equal(t, uint64(0), overBound) // no bound armed → inert
	require.Equal(t, 500*time.Millisecond, max)
}
