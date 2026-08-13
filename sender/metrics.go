package sender

import (
	"github.com/sei-protocol/sei-load/utils"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Acquired at package init, before observability.Setup installs the real
// MeterProvider. Safe because OTel Go's global is a delegating provider:
// meters and instruments created against it forward to the real provider
// once SetMeterProvider is called. See go.opentelemetry.io/otel/internal/global.
var meter = otel.Meter("github.com/sei-protocol/sei-load/sender")

// Synchronous instruments — read by Record/Add call sites.
var (
	sendLatency = utils.OrPanic1(meter.Float64Histogram(
		"send_latency",
		metric.WithDescription("Latency of sending transactions in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.1, 0.2, 0.3, 0.5, 1.0, 2.0, 3.0, 5.0, 10.0, 20.0)))

	txsAccepted = utils.OrPanic1(meter.Int64Counter(
		"txs_accepted",
		metric.WithDescription("Transactions successfully submitted"),
		metric.WithUnit("{transactions}")))

	txsRejected = utils.OrPanic1(meter.Int64Counter(
		"txs_rejected",
		metric.WithDescription("Transactions rejected by the target or local client, by reason"),
		metric.WithUnit("{transactions}")))
)

func statusAttrFromError(err error) attribute.KeyValue {
	const key = "status"
	if err == nil {
		return attribute.String(key, "success")
	}
	return attribute.String(key, "failure")
}
