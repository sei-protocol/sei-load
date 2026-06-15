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

	v := EvaluateScheduleLag(c.ScheduleLagSamples(), 100, true, 0)
	require.Equal(t, VerdictVoid, v.Verdict)
}
