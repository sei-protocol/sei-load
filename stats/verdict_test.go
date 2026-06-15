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

	v := EvaluateScheduleLag(ScheduleLagInputs{
		Samples: samples, TargetTPS: 100, OpenLoop: true, Admitted: 100,
	})

	require.Equal(t, VerdictVoid, v.Verdict)
	require.NotEmpty(t, v.VoidReason)
	require.Equal(t, 50*time.Millisecond, v.ScheduleLagP99)
	require.Equal(t, 10*time.Millisecond, v.ArrivalInterval)
}

// A healthy run keeps p99 below 10% of the 10ms interval (1ms) → VALID.
func TestEvaluateScheduleLag_HealthyIsValid(t *testing.T) {
	samples := lags(0, 0, 0, 0, 0, 0, 0, 0, 0, 0) // all 0ms, p99 = 0
	samples = append(samples, 200*time.Microsecond)

	v := EvaluateScheduleLag(ScheduleLagInputs{
		Samples: samples, TargetTPS: 100, OpenLoop: true, Admitted: 100,
	})

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
	v := EvaluateScheduleLag(ScheduleLagInputs{
		Samples: samples, TargetTPS: 0, OpenLoop: true, Admitted: 100,
	})
	require.Equal(t, 100*time.Millisecond, v.ScheduleLagP99)
	require.Equal(t, 100, v.SampleCount)
}

// Closed-loop runs are reported but never gated: N/A regardless of lag size.
func TestEvaluateScheduleLag_ClosedLoopIsNA(t *testing.T) {
	samples := lags(500, 500, 500) // huge lag, would be VOID if open-loop

	v := EvaluateScheduleLag(ScheduleLagInputs{
		Samples: samples, TargetTPS: 100, OpenLoop: false, Admitted: 3,
	})

	require.Equal(t, VerdictNA, v.Verdict)
	require.Empty(t, v.VoidReason)
	require.Equal(t, 500*time.Millisecond, v.ScheduleLagP99) // still reported
}

// Open-loop with no fixed λ (TPS=0) cannot bound against 1/λ → N/A.
func TestEvaluateScheduleLag_NoFixedRateIsNA(t *testing.T) {
	v := EvaluateScheduleLag(ScheduleLagInputs{
		Samples: lags(100, 200, 300), TargetTPS: 0, OpenLoop: true, Admitted: 3,
	})
	require.Equal(t, VerdictNA, v.Verdict)
	require.Equal(t, time.Duration(0), v.ArrivalInterval)
}

// A ramped run drives λ via the limiter, so the configured TPS is stale and
// there is no single 1/λ to gate against — N/A regardless of TPS.
func TestEvaluateScheduleLag_RampedIsNA(t *testing.T) {
	// TPS>0 but ramped: must still be N/A, not gated against the stale 1/TPS.
	v := EvaluateScheduleLag(ScheduleLagInputs{
		Samples: lags(500, 500, 500), TargetTPS: 100, OpenLoop: true, Ramped: true, Admitted: 3,
	})
	require.Equal(t, VerdictNA, v.Verdict)
	require.Empty(t, v.VoidReason)
	require.Equal(t, "ramped λ has no single arrival interval", v.NAReason)
	require.Equal(t, time.Duration(0), v.ArrivalInterval)
}

// No samples on a fixed-λ run is N/A, not VALID: it cannot distinguish a SUT
// that kept up from a recorder that never fired.
func TestEvaluateScheduleLag_NoSamplesIsNA(t *testing.T) {
	v := EvaluateScheduleLag(ScheduleLagInputs{
		Samples: nil, TargetTPS: 100, OpenLoop: true, Admitted: 0,
	})
	require.Equal(t, VerdictNA, v.Verdict)
	require.Equal(t, "no schedule_lag samples recorded", v.NAReason)
	require.False(t, v.Anomaly) // zero admitted: no anomaly, just an empty run
	require.Equal(t, time.Duration(0), v.ScheduleLagP99)
}

// Admitted txs but zero samples is an anomaly: the recorder likely never fired.
func TestEvaluateScheduleLag_AdmittedButNoSamplesIsAnomaly(t *testing.T) {
	v := EvaluateScheduleLag(ScheduleLagInputs{
		Samples: nil, TargetTPS: 100, OpenLoop: true, Admitted: 5000,
	})
	require.Equal(t, VerdictNA, v.Verdict)
	require.Equal(t, "no schedule_lag samples recorded", v.NAReason)
	require.True(t, v.Anomaly)
}

// A configured threshold overrides the default boundary.
func TestEvaluateScheduleLag_ConfiguredThreshold(t *testing.T) {
	samples := lags(2) // p99 = 2ms; interval at 100 TPS = 10ms
	// 10% bound = 1ms → VOID; 50% bound = 5ms → VALID.
	require.Equal(t, VerdictVoid, EvaluateScheduleLag(ScheduleLagInputs{
		Samples: samples, TargetTPS: 100, OpenLoop: true, Admitted: 1, Threshold: 0.10,
	}).Verdict)
	require.Equal(t, VerdictValid, EvaluateScheduleLag(ScheduleLagInputs{
		Samples: samples, TargetTPS: 100, OpenLoop: true, Admitted: 1, Threshold: 0.50,
	}).Verdict)
}

// A late-run sub-percentile tail: whole-run p99 sits UNDER the bound (the
// reservoir diluted the tail), but the exact over-bound fraction exceeds the
// threshold → VOID with the tail reason. At 100 TPS / 10% the bound is 1ms.
func TestEvaluateScheduleLag_TailDegradationIsVoid(t *testing.T) {
	// p99 of the sample is comfortably under the 1ms bound.
	samples := lags(0, 0, 0, 0, 0, 0, 0, 0, 0, 0)
	// 0.8% of 100k sends exceeded the bound — above the 0.5% fraction.
	v := EvaluateScheduleLag(ScheduleLagInputs{
		Samples: samples, TargetTPS: 100, OpenLoop: true, Admitted: 100_000,
		OverBoundCount: 800, OverBoundTotal: 100_000, MaxLag: 80 * time.Millisecond,
	})
	require.Equal(t, VerdictVoid, v.Verdict)
	require.Contains(t, v.VoidReason, "tail degradation")
	require.Contains(t, v.VoidReason, "0.80%")
	require.Less(t, v.ScheduleLagP99, time.Millisecond) // p99 alone would pass
	require.Equal(t, 80*time.Millisecond, v.MaxLag)     // surfaced for diagnostics
}

// A single over-bound outlier (one GC pause) well under the fraction must NOT
// trip the tail gate: the run stays VALID. This is why the gate is a fraction,
// not maxLag alone.
func TestEvaluateScheduleLag_LoneOutlierStaysValid(t *testing.T) {
	samples := lags(0, 0, 0, 0, 0, 0, 0, 0, 0, 0)
	// 1 / 100k = 0.001% over bound, far below the 0.5% fraction.
	v := EvaluateScheduleLag(ScheduleLagInputs{
		Samples: samples, TargetTPS: 100, OpenLoop: true, Admitted: 100_000,
		OverBoundCount: 1, OverBoundTotal: 100_000, MaxLag: 200 * time.Millisecond,
	})
	require.Equal(t, VerdictValid, v.Verdict)
	require.Empty(t, v.VoidReason)
	require.Equal(t, 200*time.Millisecond, v.MaxLag) // still surfaced
}

// A clean run with no over-bound sends is VALID; the tail gate is a no-op.
func TestEvaluateScheduleLag_NoOverBoundIsValid(t *testing.T) {
	samples := lags(0, 0, 0, 0, 0, 0, 0, 0, 0, 0)
	v := EvaluateScheduleLag(ScheduleLagInputs{
		Samples: samples, TargetTPS: 100, OpenLoop: true, Admitted: 100_000,
		OverBoundCount: 0, OverBoundTotal: 100_000, MaxLag: 500 * time.Microsecond,
	})
	require.Equal(t, VerdictValid, v.Verdict)
	require.Empty(t, v.VoidReason)
}

// ScheduleLagBound returns threshold × 1/λ, zero when λ is not fixed, and falls
// back to the default threshold — the single source the collector arms from.
func TestScheduleLagBound(t *testing.T) {
	require.Equal(t, time.Millisecond, ScheduleLagBound(100, 0.10)) // 10% of 10ms
	require.Equal(t, time.Millisecond, ScheduleLagBound(100, 0))    // default 0.10
	require.Equal(t, time.Duration(0), ScheduleLagBound(0, 0.10))   // no fixed λ
}
