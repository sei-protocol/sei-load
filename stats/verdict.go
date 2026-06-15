package stats

import (
	"fmt"
	"sort"
	"time"
)

// scheduleLagVoidThreshold is the fraction of the arrival interval (1/λ) that
// schedule_lag_p99 may reach before the run is VOID: a p99 send lag above this
// fraction means the generator could not keep up with its own schedule, so the
// load was generator-bound, not open-loop, and the run does not measure the SUT.
// Provisional value — tune from first calibration run.
const scheduleLagVoidThreshold = 0.10

// Verdict labels for a run's open-loop self-check.
const (
	VerdictValid = "VALID"
	VerdictVoid  = "VOID"
	// VerdictNA marks a run where the self-check does not apply: closed-loop, a
	// ramped λ (no single 1/λ), no fixed arrival rate, or a fixed-λ run that
	// recorded zero schedule_lag samples (cannot prove open-loop either way).
	// schedule_lag_p99 is still reported, but no pass/fail gate is rendered.
	VerdictNA = "N/A"
)

// ScheduleLagVerdict is the self-check result that proves an open-loop run was
// actually open-loop. schedule_lag = AttemptedSendTime − IntendedSendTime per
// tx; its p99 is checked against threshold × (1/λ). It is computed on every run
// and reported regardless of outcome.
type ScheduleLagVerdict struct {
	// Verdict is VerdictValid, VerdictVoid, or VerdictNA.
	Verdict string
	// VoidReason is a human-readable explanation, empty unless Verdict is VOID.
	VoidReason string
	// NAReason explains an N/A verdict (why no gate applies); empty otherwise.
	NAReason string
	// Anomaly is true when the inputs are self-inconsistent — admitted txs but
	// zero schedule_lag samples — so the caller can log loudly: the recorder is
	// likely mis-wired rather than the run being clean.
	Anomaly bool
	// ScheduleLagP99 is the 99th-percentile send lag across sampled open-loop
	// txs; zero when no open-loop samples were recorded.
	ScheduleLagP99 time.Duration
	// SampleCount is the number of schedule_lag samples the verdict is based on.
	SampleCount int
	// ArrivalInterval is 1/λ, the bound's reference interval; zero when λ is not
	// a single fixed rate (e.g. ramping with no configured TPS).
	ArrivalInterval time.Duration
	// Threshold is the fraction of ArrivalInterval used as the VOID boundary.
	Threshold float64
}

// EvaluateScheduleLag computes the open-loop self-check verdict from the
// recorded schedule_lag samples, the configured arrival rate targetTPS (λ), the
// run's arrival model, whether the run ramped λ, the count of admitted txs, and
// the VOID threshold fraction (<=0 falls back to the provisional default). p99
// is the sorted-slice percentile, matching the repo's block-time percentile
// idiom.
//
// The verdict is N/A — reported, never a gate — when the model is not open-loop,
// when the run ramped λ (a ramp has no single 1/λ to bound against, and the
// ramper drives the live limit so targetTPS is stale), or when λ is not a single
// fixed rate (targetTPS <= 0). A fixed-λ open-loop run that recorded zero
// schedule_lag samples is also N/A, not VALID: zero samples cannot distinguish a
// SUT that kept up from a recorder that never fired. When admitted > 0 yet no
// samples landed, Anomaly is set so the caller logs the mis-wiring loudly.
// schedule_lag_p99 is still reported in every case.
func EvaluateScheduleLag(samples []time.Duration, targetTPS float64, openLoop, ramped bool, admitted uint64, threshold float64) ScheduleLagVerdict {
	if threshold <= 0 {
		threshold = scheduleLagVoidThreshold
	}

	v := ScheduleLagVerdict{
		Verdict:        VerdictNA,
		ScheduleLagP99: scheduleLagPercentile(samples, 99),
		SampleCount:    len(samples),
		Threshold:      threshold,
	}

	if !openLoop {
		v.NAReason = "closed-loop run: open-loop self-check does not apply"
		return v
	}
	if ramped {
		v.NAReason = "ramped λ has no single arrival interval"
		return v
	}
	if targetTPS <= 0 {
		v.NAReason = "no fixed arrival rate (λ): nothing to bound against"
		return v
	}

	arrivalInterval := time.Duration(float64(time.Second) / targetTPS)
	v.ArrivalInterval = arrivalInterval

	// Zero samples is N/A, not VALID: it cannot tell a SUT that kept up from a
	// recorder that never fired or a run that dropped every tick. Admitted txs
	// with no samples is an outright anomaly — flag it for the caller.
	if len(samples) == 0 {
		v.NAReason = "no schedule_lag samples recorded"
		v.Anomaly = admitted > 0
		return v
	}

	bound := time.Duration(threshold * float64(arrivalInterval))
	if v.ScheduleLagP99 > bound {
		v.Verdict = VerdictVoid
		v.VoidReason = formatVoidReason(v.ScheduleLagP99, bound, threshold, arrivalInterval)
		return v
	}
	v.Verdict = VerdictValid
	return v
}

func formatVoidReason(p99, bound time.Duration, threshold float64, arrivalInterval time.Duration) string {
	return fmt.Sprintf(
		"generator-bound: schedule_lag_p99 %s exceeds %s (%.0f%% of arrival interval %s) — load was not open-loop",
		p99.Round(time.Microsecond), bound.Round(time.Microsecond), threshold*100, arrivalInterval.Round(time.Microsecond))
}

// scheduleLagPercentile returns the percentile of a copy-then-sort of samples,
// reusing the repo's calculatePercentile index rule. Copies so the caller's
// slice order is preserved.
func scheduleLagPercentile(samples []time.Duration, percentile int) time.Duration {
	if len(samples) == 0 {
		return 0
	}
	sorted := make([]time.Duration, len(samples))
	copy(sorted, samples)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	return calculatePercentile(sorted, percentile)
}
