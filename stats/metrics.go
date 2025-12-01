package stats

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter = otel.Meter("seiload/stats")

	metrics = struct {
		gasUsed      metric.Int64Histogram
		blockNumber  metric.Int64Gauge
		blockTime    metric.Float64Histogram
		blockTxCount metric.Int64Histogram
		blockTPS     metric.Float64Histogram
	}{
		gasUsed: must(meter.Int64Histogram(
			"gas_used",
			metric.WithDescription("Gas used in transactions"),
			metric.WithUnit("{gas}"),
			metric.WithExplicitBucketBoundaries(1, 1000, 10_000, 50_000, 100_000, 200_000, 300_000, 400_000, 500_000, 600_000, 700_000, 800_000, 1_000_000))),
		blockNumber: must(meter.Int64Gauge(
			"block_number",
			metric.WithDescription("Block number in the chain"),
			metric.WithUnit("{height}"))),
		blockTime: must(meter.Float64Histogram(
			"block_time",
			metric.WithDescription("Time taken to produce a block"),
			metric.WithUnit("s"),
			metric.WithExplicitBucketBoundaries(0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 2.0, 5.0, 10.0, 20.0))),
		blockTxCount: must(meter.Int64Histogram(
			"block_tx_count",
			metric.WithDescription("Number of transactions per block"),
			metric.WithUnit("{tx}"),
			metric.WithExplicitBucketBoundaries(0, 1, 10, 50, 100, 200, 500, 1000, 2000, 5000, 10000))),
		blockTPS: must(meter.Float64Histogram(
			"block_tps",
			metric.WithDescription("Actual on-chain TPS (transactions per second based on block time)"),
			metric.WithUnit("{tx/s}"),
			metric.WithExplicitBucketBoundaries(1, 10, 50, 100, 200, 500, 1000, 2000, 5000, 10000, 20000))),
	}
)

// must panics if err is non-nil, otherwise returns v.
func must[V any](v V, err error) V {
	if err != nil {
		panic(err)
	}
	return v
}
