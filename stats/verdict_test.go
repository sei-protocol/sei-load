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

	v := EvaluateScheduleLag(samples, 100, true, 0)

	require.Equal(t, VerdictVoid, v.Verdict)
	require.NotEmpty(t, v.VoidReason)
	require.Equal(t, 50*time.Millisecond, v.ScheduleLagP99)
	require.Equal(t, 10*time.Millisecond, v.ArrivalInterval)
}

// A healthy run keeps p99 below 10% of the 10ms interval (1ms) → VALID.
func TestEvaluateScheduleLag_HealthyIsValid(t *testing.T) {
	samples := lags(0, 0, 0, 0, 0, 0, 0, 0, 0, 0) // all 0ms, p99 = 0
	samples = append(samples, 200*time.Microsecond)

	v := EvaluateScheduleLag(samples, 100, true, 0)

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
	v := EvaluateScheduleLag(samples, 0, true, 0)
	require.Equal(t, 100*time.Millisecond, v.ScheduleLagP99)
	require.Equal(t, 100, v.SampleCount)
}

// Closed-loop runs are reported but never gated: N/A regardless of lag size.
func TestEvaluateScheduleLag_ClosedLoopIsNA(t *testing.T) {
	samples := lags(500, 500, 500) // huge lag, would be VOID if open-loop

	v := EvaluateScheduleLag(samples, 100, false, 0)

	require.Equal(t, VerdictNA, v.Verdict)
	require.Empty(t, v.VoidReason)
	require.Equal(t, 500*time.Millisecond, v.ScheduleLagP99) // still reported
}

// Open-loop with no fixed λ (e.g. ramping, TPS=0) cannot bound against 1/λ → N/A.
func TestEvaluateScheduleLag_NoFixedRateIsNA(t *testing.T) {
	v := EvaluateScheduleLag(lags(100, 200, 300), 0, true, 0)
	require.Equal(t, VerdictNA, v.Verdict)
	require.Equal(t, time.Duration(0), v.ArrivalInterval)
}

// No open-loop samples is VALID, not VOID: VOID needs an observed bad run.
func TestEvaluateScheduleLag_NoSamplesIsValid(t *testing.T) {
	v := EvaluateScheduleLag(nil, 100, true, 0)
	require.Equal(t, VerdictValid, v.Verdict)
	require.Equal(t, time.Duration(0), v.ScheduleLagP99)
}

// A configured threshold overrides the default boundary.
func TestEvaluateScheduleLag_ConfiguredThreshold(t *testing.T) {
	samples := lags(2) // p99 = 2ms; interval at 100 TPS = 10ms
	// 10% bound = 1ms → VOID; 50% bound = 5ms → VALID.
	require.Equal(t, VerdictVoid, EvaluateScheduleLag(samples, 100, true, 0.10).Verdict)
	require.Equal(t, VerdictValid, EvaluateScheduleLag(samples, 100, true, 0.50).Verdict)
}
