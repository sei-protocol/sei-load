package stats

import (
	"sync"

	"go.opentelemetry.io/otel/metric"

	"github.com/sei-protocol/sei-load/observability"
)

type statsMetricsBundle struct {
	gasUsed     metric.Int64Histogram
	blockNumber metric.Int64Gauge
	blockTime   metric.Float64Histogram

	// --- run-summary instruments (per sei-load-observability design) ---
	// These are gauges so their single-emission-at-run-end semantics produce
	// exactly 1 series per run via the Resource join — the right shape for
	// comparative benchmark dashboards across commits.
	runTPSFinal         metric.Float64Gauge
	runDurationSeconds  metric.Float64Gauge
	runTxsAcceptedTotal metric.Int64Gauge
}

var statsMetrics = sync.OnceValue(func() *statsMetricsBundle {
	m := observability.Meter("github.com/sei-protocol/sei-load/stats")
	b := &statsMetricsBundle{}

	b.gasUsed = must(m.Int64Histogram(
		"gas_used",
		metric.WithDescription("Gas used in transactions"),
		metric.WithUnit("{gas}"),
		metric.WithExplicitBucketBoundaries(1, 1000, 10_000, 50_000, 100_000, 200_000, 300_000, 400_000, 500_000, 600_000, 700_000, 800_000, 1_000_000)))
	b.blockNumber = must(m.Int64Gauge(
		"block_number",
		metric.WithDescription("Block number in the chain"),
		metric.WithUnit("{height}")))
	b.blockTime = must(m.Float64Histogram(
		"block_time",
		metric.WithDescription("Time taken to produce a block"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 2.0, 5.0, 10.0, 20.0)))

	b.runTPSFinal = must(m.Float64Gauge(
		"run_tps_final",
		metric.WithDescription("Final observed TPS for this run (emitted once at run end)"),
		metric.WithUnit("{transactions}/s")))
	b.runDurationSeconds = must(m.Float64Gauge(
		"run_duration_seconds",
		metric.WithDescription("Wall-clock duration of this run (emitted once at run end)"),
		metric.WithUnit("s")))
	b.runTxsAcceptedTotal = must(m.Int64Gauge(
		"run_txs_accepted_total",
		metric.WithDescription("Total transactions accepted by endpoints over this run (emitted once at run end)"),
		metric.WithUnit("{transactions}")))

	return b
})

// must panics if err is non-nil, otherwise returns v.
func must[V any](v V, err error) V {
	if err != nil {
		panic(err)
	}
	return v
}
