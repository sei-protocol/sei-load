package stats

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter = otel.Meter("seiload/stats")

	metrics = struct {
		gasUsed     metric.Int64Histogram
		blockNumber metric.Int64Gauge
		blockTime   metric.Float64Histogram
	}{
		gasUsed: must(meter.Int64Histogram(
			"gas_used",
			metric.WithDescription("Gas used in transactions"),
			metric.WithUnit("{gas}"))),
		blockNumber: must(meter.Int64Gauge(
			"block_number",
			metric.WithDescription("Block number in the chain"),
			metric.WithUnit("{height}"))),
		blockTime: must(meter.Float64Histogram(
			"block_time",
			metric.WithDescription("Time taken to produce a block"),
			metric.WithUnit("s"))),
	}
)

// must panics if err is non-nil, otherwise returns v.
func must[V any](v V, err error) V {
	if err != nil {
		panic(err)
	}
	return v
}
