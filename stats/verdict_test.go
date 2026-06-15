package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// lags builds a sample slice from millisecond values for readability.
func lags(ms ...int) []time.Duration {
	out := make([]time.Duration, len(ms))
	for i, m := range ms {
		out[i] = time.Duration(m) * time.Millisecond
	}
	return out
}

// At 100 TPS the arrival interval is 10ms, so the VOID bound at the default 10%
// threshold is 1ms. An over-driven run whose p99 sits well above that is VOID.
func TestEvaluateScheduleLag_OverDrivenIsVoid(t *testing.T) {
	// 100 samples mostly small, but the top tail (p99 index = 99) is large.
	samples := make([]time.Duration, 0, 100)
	for range 99 {
		samples = append(samples, 100*time.Microsecond)
	}
	samples = append(samples, 50*time.Millisecond) // the p99 element

	v := EvaluateScheduleLag(samples, 100, true, false, 100, 0)

	require.Equal(t, VerdictVoid, v.Verdict)
	require.NotEmpty(t, v.VoidReason)
	require.Equal(t, 50*time.Millisecond, v.ScheduleLagP99)
	require.Equal(t, 10*time.Millisecond, v.ArrivalInterval)
}

// A healthy run keeps p99 below 10% of the 10ms interval (1ms) → VALID.
func TestEvaluateScheduleLag_HealthyIsValid(t *testing.T) {
	samples := lags(0, 0, 0, 0, 0, 0, 0, 0, 0, 0) // all 0ms, p99 = 0
	samples = append(samples, 200*time.Microsecond)

	v := EvaluateScheduleLag(samples, 100, true, false, 100, 0)

	require.Equal(t, VerdictValid, v.Verdict)
	require.Empty(t, v.VoidReason)
	require.Less(t, v.ScheduleLagP99, time.Millisecond)
}

// p99 must match the repo's sorted-slice index rule for a known set.
func TestEvaluateScheduleLag_P99ComputedCorrectly(t *testing.T) {
	// 100 samples 1ms..100ms; index = (100*99)/100 = 99 → sorted[99] = 100ms.
	samples := make([]time.Duration, 0, 100)
	for i := 1; i <= 100; i++ {
		samples = append(samples, time.Duration(i)*time.Millisecond)
	}

	// targetTPS=0 keeps verdict N/A but still reports p99.
	v := EvaluateScheduleLag(samples, 0, true, false, 100, 0)
	require.Equal(t, 100*time.Millisecond, v.ScheduleLagP99)
	require.Equal(t, 100, v.SampleCount)
}

// Closed-loop runs are reported but never gated: N/A regardless of lag size.
func TestEvaluateScheduleLag_ClosedLoopIsNA(t *testing.T) {
	samples := lags(500, 500, 500) // huge lag, would be VOID if open-loop

	v := EvaluateScheduleLag(samples, 100, false, false, 3, 0)

	require.Equal(t, VerdictNA, v.Verdict)
	require.Empty(t, v.VoidReason)
	require.Equal(t, 500*time.Millisecond, v.ScheduleLagP99) // still reported
}

// Open-loop with no fixed λ (TPS=0) cannot bound against 1/λ → N/A.
func TestEvaluateScheduleLag_NoFixedRateIsNA(t *testing.T) {
	v := EvaluateScheduleLag(lags(100, 200, 300), 0, true, false, 3, 0)
	require.Equal(t, VerdictNA, v.Verdict)
	require.Equal(t, time.Duration(0), v.ArrivalInterval)
}

// A ramped run drives λ via the limiter, so the configured TPS is stale and
// there is no single 1/λ to gate against — N/A regardless of TPS.
func TestEvaluateScheduleLag_RampedIsNA(t *testing.T) {
	// TPS>0 but ramped: must still be N/A, not gated against the stale 1/TPS.
	v := EvaluateScheduleLag(lags(500, 500, 500), 100, true, true, 3, 0)
	require.Equal(t, VerdictNA, v.Verdict)
	require.Empty(t, v.VoidReason)
	require.Equal(t, "ramped λ has no single arrival interval", v.NAReason)
	require.Equal(t, time.Duration(0), v.ArrivalInterval)
}

// No samples on a fixed-λ run is N/A, not VALID: it cannot distinguish a SUT
// that kept up from a recorder that never fired.
func TestEvaluateScheduleLag_NoSamplesIsNA(t *testing.T) {
	v := EvaluateScheduleLag(nil, 100, true, false, 0, 0)
	require.Equal(t, VerdictNA, v.Verdict)
	require.Equal(t, "no schedule_lag samples recorded", v.NAReason)
	require.False(t, v.Anomaly) // zero admitted: no anomaly, just an empty run
	require.Equal(t, time.Duration(0), v.ScheduleLagP99)
}

// Admitted txs but zero samples is an anomaly: the recorder likely never fired.
func TestEvaluateScheduleLag_AdmittedButNoSamplesIsAnomaly(t *testing.T) {
	v := EvaluateScheduleLag(nil, 100, true, false, 5000, 0)
	require.Equal(t, VerdictNA, v.Verdict)
	require.Equal(t, "no schedule_lag samples recorded", v.NAReason)
	require.True(t, v.Anomaly)
}

// A configured threshold overrides the default boundary.
func TestEvaluateScheduleLag_ConfiguredThreshold(t *testing.T) {
	samples := lags(2) // p99 = 2ms; interval at 100 TPS = 10ms
	// 10% bound = 1ms → VOID; 50% bound = 5ms → VALID.
	require.Equal(t, VerdictVoid, EvaluateScheduleLag(samples, 100, true, false, 1, 0.10).Verdict)
	require.Equal(t, VerdictValid, EvaluateScheduleLag(samples, 100, true, false, 1, 0.50).Verdict)
}
