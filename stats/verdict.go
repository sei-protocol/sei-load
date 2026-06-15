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

// scheduleLagOverBoundFraction is the share of recorded sends that may exceed the
// VOID bound before the run is VOID on the unsampled tail signal. The whole-run
// p99 is computed from a uniform reservoir sample, so a sub-percentile late-run
// tail (the generator hiccupping near the end of a long run) can stay under the
// p99 yet still mean the generator fell behind. This exact (un-sampled) fraction
// catches that tail; it is a fraction, not a single max-lag, so a lone GC-pause
// outlier does not trip it. Provisional — tune from first calibration run.
const scheduleLagOverBoundFraction = 0.005

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
	// OverBoundCount is the exact (un-sampled) count of sends whose lag exceeded
	// the VOID bound; OverBoundTotal is the exact total recorded. Their ratio is
	// the tail-degradation signal the reservoir p99 cannot dilute.
	OverBoundCount uint64
	OverBoundTotal uint64
	// MaxLag is the largest lag recorded over the run (un-sampled), surfaced for
	// diagnostics; it is not a gate on its own (a fraction is, to survive a lone
	// outlier).
	MaxLag time.Duration
}

// ScheduleLagInputs carries the verdict inputs. It replaces a long positional
// signature (the tail figures pushed it past the point where adjacent bools and
// uints read clearly at the call site).
type ScheduleLagInputs struct {
	// Samples is the reservoir copy used for the p99.
	Samples []time.Duration
	// TargetTPS is the configured λ; <=0 means no fixed rate → N/A.
	TargetTPS float64
	// OpenLoop and Ramped gate applicability: only a fixed-λ open-loop,
	// non-ramped run is evaluated.
	OpenLoop bool
	Ramped   bool
	// Admitted is the count of admitted txs, used only to flag the
	// admitted-but-no-samples anomaly.
	Admitted uint64
	// Threshold is the VOID fraction of 1/λ for the p99 bound; <=0 falls back to
	// the provisional default.
	Threshold float64
	// OverBoundCount / OverBoundTotal / MaxLag are the collector's exact
	// (un-sampled) tail figures (see Collector.ScheduleLagTail).
	OverBoundCount uint64
	OverBoundTotal uint64
	MaxLag         time.Duration
}

// EvaluateScheduleLag computes the open-loop self-check verdict. p99 is the
// sorted-slice percentile of the reservoir sample, matching the repo's block-time
// percentile idiom; the run is also VOID on the exact (un-sampled) over-bound
// fraction, the tail signal the reservoir cannot dilute.
//
// The verdict is N/A — reported, never a gate — when the model is not open-loop,
// when the run ramped λ (a ramp has no single 1/λ to bound against, and the
// ramper drives the live limit so the configured λ is stale), or when λ is not a
// single fixed rate (TargetTPS <= 0). A fixed-λ open-loop run that recorded zero
// schedule_lag samples is also N/A, not VALID: zero samples cannot distinguish a
// SUT that kept up from a recorder that never fired. When Admitted > 0 yet no
// samples landed, Anomaly is set so the caller logs the mis-wiring loudly.
// schedule_lag_p99 is still reported in every case.
func EvaluateScheduleLag(in ScheduleLagInputs) ScheduleLagVerdict {
	threshold := in.Threshold
	if threshold <= 0 {
		threshold = scheduleLagVoidThreshold
	}

	v := ScheduleLagVerdict{
		Verdict:        VerdictNA,
		ScheduleLagP99: scheduleLagPercentile(in.Samples, 99),
		SampleCount:    len(in.Samples),
		Threshold:      threshold,
		OverBoundCount: in.OverBoundCount,
		OverBoundTotal: in.OverBoundTotal,
		MaxLag:         in.MaxLag,
	}

	if !in.OpenLoop {
		v.NAReason = "closed-loop run: open-loop self-check does not apply"
		return v
	}
	if in.Ramped {
		v.NAReason = "ramped λ has no single arrival interval"
		return v
	}
	if in.TargetTPS <= 0 {
		v.NAReason = "no fixed arrival rate (λ): nothing to bound against"
		return v
	}

	arrivalInterval := time.Duration(float64(time.Second) / in.TargetTPS)
	v.ArrivalInterval = arrivalInterval

	// Zero samples is N/A, not VALID: it cannot tell a SUT that kept up from a
	// recorder that never fired or a run that dropped every tick. Admitted txs
	// with no samples is an outright anomaly — flag it for the caller.
	if len(in.Samples) == 0 {
		v.NAReason = "no schedule_lag samples recorded"
		v.Anomaly = in.Admitted > 0
		return v
	}

	bound := ScheduleLagBound(in.TargetTPS, threshold)

	// Whole-run p99 over bound: the run was generator-bound across the sample.
	if v.ScheduleLagP99 > bound {
		v.Verdict = VerdictVoid
		v.VoidReason = formatP99VoidReason(v.ScheduleLagP99, bound, threshold, arrivalInterval)
		return v
	}
	// Unsampled tail: a sub-percentile share over the bound that the reservoir
	// p99 diluted. Checked only when the bound was armed (OverBoundTotal > 0).
	if in.OverBoundTotal > 0 {
		if frac := float64(in.OverBoundCount) / float64(in.OverBoundTotal); frac > scheduleLagOverBoundFraction {
			v.Verdict = VerdictVoid
			v.VoidReason = formatTailVoidReason(in.OverBoundCount, in.OverBoundTotal, frac, bound)
			return v
		}
	}
	v.Verdict = VerdictValid
	return v
}

// ScheduleLagBound is the VOID bound, threshold × 1/λ, for a fixed-λ open-loop
// run. Returns zero when there is no single fixed rate (targetTPS <= 0), so the
// caller leaves the collector's over-bound counter inert. threshold <= 0 falls
// back to the provisional default, matching EvaluateScheduleLag.
func ScheduleLagBound(targetTPS, threshold float64) time.Duration {
	if targetTPS <= 0 {
		return 0
	}
	if threshold <= 0 {
		threshold = scheduleLagVoidThreshold
	}
	arrivalInterval := time.Duration(float64(time.Second) / targetTPS)
	return time.Duration(threshold * float64(arrivalInterval))
}

func formatP99VoidReason(p99, bound time.Duration, threshold float64, arrivalInterval time.Duration) string {
	return fmt.Sprintf(
		"generator-bound: schedule_lag_p99 %s exceeds %s bound (%.0f%% of arrival interval %s) — load was not open-loop",
		p99.Round(time.Microsecond), bound.Round(time.Microsecond), threshold*100, arrivalInterval.Round(time.Microsecond))
}

func formatTailVoidReason(overBound, total uint64, frac float64, bound time.Duration) string {
	return fmt.Sprintf(
		"tail degradation: %.2f%% of sends (%d/%d) exceeded the %s bound — generator fell behind on a sub-percentile tail the p99 missed",
		frac*100, overBound, total, bound.Round(time.Microsecond))
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
