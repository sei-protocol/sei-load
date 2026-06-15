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
	// VerdictNA marks a run where the self-check does not apply (closed-loop, or
	// no fixed arrival rate to compare against). schedule_lag_p99 is still
	// reported, but no pass/fail gate is rendered.
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
// run's arrival model, and the VOID threshold fraction (<=0 falls back to the
// provisional default). p99 is the sorted-slice percentile, matching the repo's
// block-time percentile idiom.
//
// The verdict is N/A — reported, never a gate — when the model is not open-loop
// or when λ is not a single fixed rate (targetTPS <= 0, e.g. a ramping run),
// since there is no single 1/λ to bound against. schedule_lag_p99 is still
// reported in those cases.
func EvaluateScheduleLag(samples []time.Duration, targetTPS float64, openLoop bool, threshold float64) ScheduleLagVerdict {
	if threshold <= 0 {
		threshold = scheduleLagVoidThreshold
	}

	v := ScheduleLagVerdict{
		Verdict:        VerdictNA,
		ScheduleLagP99: scheduleLagPercentile(samples, 99),
		SampleCount:    len(samples),
		Threshold:      threshold,
	}

	if !openLoop || targetTPS <= 0 {
		return v
	}

	arrivalInterval := time.Duration(float64(time.Second) / targetTPS)
	v.ArrivalInterval = arrivalInterval

	// No samples: the run scheduled nothing open-loop. Treat as VALID (nothing
	// disproves open-loop) rather than VOID — VOID is reserved for an observed
	// generator-bound run.
	if len(samples) == 0 {
		v.Verdict = VerdictValid
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
