package stats

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// See sender/metrics.go for why package-level acquisition is safe before Setup.
var meter = otel.Meter("github.com/sei-protocol/sei-load/stats")

var (
	gasUsed = must(meter.Int64Histogram(
		"gas_used",
		metric.WithDescription("Gas used in transactions"),
		metric.WithUnit("{gas}"),
		metric.WithExplicitBucketBoundaries(1, 1000, 10_000, 50_000, 100_000, 200_000, 300_000, 400_000, 500_000, 600_000, 700_000, 800_000, 1_000_000)))

	blockNumber = must(meter.Int64Gauge(
		"block_number",
		metric.WithDescription("Block number in the chain"),
		metric.WithUnit("{height}")))

	blockTime = must(meter.Float64Histogram(
		"block_time",
		metric.WithDescription("Time taken to produce a block"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 2.0, 5.0, 10.0, 20.0)))

	// Run-summary: gauges emitted once at run end → 1 series/run via Resource join.
	runTPSFinal = must(meter.Float64Gauge(
		"run_tps_final",
		metric.WithDescription("Final observed TPS for this run (emitted once at run end)"),
		metric.WithUnit("{transactions}/s")))

	runDurationSeconds = must(meter.Float64Gauge(
		"run_duration",
		metric.WithDescription("Wall-clock duration of this run (emitted once at run end)"),
		metric.WithUnit("s")))

	runTxsAcceptedTotal = must(meter.Int64Gauge(
		"run_txs_accepted_total",
		metric.WithDescription("Total transactions accepted by endpoints over this run (emitted once at run end)"),
		metric.WithUnit("{transactions}")))

	runTxsDroppedTotal = must(meter.Int64Gauge(
		"run_txs_dropped_total",
		metric.WithDescription("Total open-loop transactions dropped because in-flight was saturated at their scheduled instant (emitted once at run end)"),
		metric.WithUnit("{transactions}")))

	runTxsFailedTotal = must(meter.Int64Gauge(
		"run_txs_failed_total",
		metric.WithDescription("Total open-loop transactions admitted and enqueued but whose send completed with an error (emitted once at run end)"),
		metric.WithUnit("{transactions}")))

	// Inclusion tracker. inclusion_latency is open-loop only (closed-loop
	// IntendedSendTime is enqueue time, not a schedule); its _count is the
	// included count only there. Denominator for inclusion rate is the existing
	// succeeded/txs_accepted series, never a new "registered" series.
	inclusionLatency = must(meter.Float64Histogram(
		"inclusion_latency",
		metric.WithDescription("Latency from intended send to observed on-chain inclusion in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.5, 1, 2, 5, 10, 30, 60, 120)))

	inclusionOutcome = must(meter.Int64Counter(
		"inclusion_outcome",
		metric.WithDescription("In-flight txs that left the registry un-included, by outcome (expired, dropped_at_cap)"),
		metric.WithUnit("{transactions}")))

	inclusionBlockGaps = must(meter.Int64Counter(
		"block_gaps",
		metric.WithDescription("Missed block heights observed by the inclusion tracker (no backfill)"),
		metric.WithUnit("{blocks}")))

	inclusionBlockFetchErrors = must(meter.Int64Counter(
		"block_fetch_errors",
		metric.WithDescription("Block-body fetches that failed; the block's txs go unmatched and reap as expired (no retry)"),
		metric.WithUnit("{blocks}")))

	// Run-summary: the only inclusion tally with no live series, since it is the
	// terminal value of the inclusion_inflight gauge. Emitted once at run end.
	runInflightAtShutdown = must(meter.Int64Gauge(
		"run_inflight_at_shutdown",
		metric.WithDescription("In-flight inclusion registry size at run end (emitted once at run end)"),
		metric.WithUnit("{transactions}")))

	// Open-loop self-check. Emitted once at run end on every run; the
	// verdict label distinguishes VALID / VOID / N/A so a generator-bound run is
	// queryable, not just a log line.
	runScheduleLagP99 = must(meter.Float64Gauge(
		"run_schedule_lag_p99",
		metric.WithDescription("p99 of per-tx send lag (attempted − intended) over this open-loop run (emitted once at run end)"),
		metric.WithUnit("s")))

	// Unsampled tail signal: the reservoir p99 above can dilute a sub-percentile
	// late-run tail, so the verdict also gates on the exact over-bound fraction.
	// Max is diagnostic; fraction is the gate.
	runScheduleLagMax = must(meter.Float64Gauge(
		"run_schedule_lag_max",
		metric.WithDescription("max per-tx send lag (attempted − intended) over this open-loop run, un-sampled (emitted once at run end)"),
		metric.WithUnit("s")))

	runScheduleLagOverBoundFraction = must(meter.Float64Gauge(
		"run_schedule_lag_over_bound_fraction",
		metric.WithDescription("exact fraction of sends whose lag exceeded the VOID bound over this open-loop run (emitted once at run end)"),
		metric.WithUnit("1")))
)

// meteredInclusionTrackers backs the inclusion_inflight gauge: each tracker
// registers so the callback can sample its in-flight map under lock.
var meteredInclusionTrackers = struct {
	lock     sync.RWMutex
	trackers []*InclusionTracker
}{}

func meterInclusionInflight(t *InclusionTracker) {
	meteredInclusionTrackers.lock.Lock()
	defer meteredInclusionTrackers.lock.Unlock()
	meteredInclusionTrackers.trackers = append(meteredInclusionTrackers.trackers, t)
}

func init() {
	must(meter.Int64ObservableGauge(
		"inclusion_inflight",
		metric.WithDescription("Current size of the inclusion tracker's in-flight tx registry"),
		metric.WithUnit("{transactions}"),
		metric.WithInt64Callback(func(_ context.Context, observer metric.Int64Observer) error {
			meteredInclusionTrackers.lock.RLock()
			defer meteredInclusionTrackers.lock.RUnlock()
			for _, t := range meteredInclusionTrackers.trackers {
				for s := range t.state.Lock() {
					observer.Observe(int64(len(s.inflight)),
						metric.WithAttributes(attribute.String("chain_id", t.seiChainID)))
				}
			}
			return nil
		})))
}

func must[V any](v V, err error) V {
	if err != nil {
		panic(err)
	}
	return v
}
