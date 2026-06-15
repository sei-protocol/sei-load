package sender

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"github.com/sei-protocol/sei-load/utils"
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

	receiptLatency = utils.OrPanic1(meter.Float64Histogram(
		"receipt_latency",
		metric.WithDescription("Latency from transaction submission to receipt confirmation in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.1, 0.2, 0.3, 0.5, 1.0, 2.0, 3.0, 5.0, 10.0, 20.0)))

	txsAccepted = utils.OrPanic1(meter.Int64Counter(
		"txs_accepted",
		metric.WithDescription("Transactions successfully submitted to an endpoint"),
		metric.WithUnit("{transactions}")))

	txsRejected = utils.OrPanic1(meter.Int64Counter(
		"txs_rejected",
		metric.WithDescription("Transactions rejected by the target or local client, by reason"),
		metric.WithUnit("{transactions}")))
)

// Observable instruments — registered in init for their callback side effect.
// Return values are discarded because OTel invokes the callbacks on each
// collection; we never read the instrument handles.
func init() {
	utils.OrPanic1(meter.Int64ObservableGauge(
		"worker_queue_length",
		metric.WithDescription("Length of the worker's queue"),
		metric.WithUnit("{count}"),
		metric.WithInt64Callback(func(ctx context.Context, observer metric.Int64Observer) error {
			for _,senders := range meteredSenders.RLock() {
				for _,ss := range senders {
					for _, shard := range ss.shards {
						observer.Observe(int64(worker.ChannelLength()), metric.WithAttributes(
							attribute.String("endpoint", worker.Endpoint()),
							attribute.Int("worker_id", worker.cfg.ID),
							attribute.String("chain_id", worker.cfg.SeiChainID),
						))
					}
				}
			}
			return nil
		})))

	utils.OrPanic1(meter.Float64ObservableGauge(
		"tps_achieved",
		metric.WithDescription("Most recent TPS sample observed by the sender, per endpoint/scenario"),
		metric.WithUnit("{transactions}/s"),
		metric.WithFloat64Callback(observeTPS)))
}

// meteredChainWorkers is the registry the worker_queue_length callback reads.
var meteredSenders = utils.NewRWMutex(map[*ShardedSender]struct{}{})

var tpsObserverRegistry = utils.NewRWMutex(map[tpsSampleKey]float64{})

type tpsSampleKey struct {
	endpoint string
	chainID  string
	scenario string
}

// RecordTPSSample publishes the latest TPS sample read by the tps_achieved gauge.
func RecordTPSSample(endpoint, chainID, scenario string, tps float64) {
	tpsObserverRegistry.lock.Lock()
	defer tpsObserverRegistry.lock.Unlock()
	tpsObserverRegistry.samples[tpsSampleKey{endpoint, chainID, scenario}] = tps
}

func observeTPS(_ context.Context, observer metric.Float64Observer) error {
	tpsObserverRegistry.lock.RLock()
	defer tpsObserverRegistry.lock.RUnlock()
	for k, v := range tpsObserverRegistry.samples {
		observer.Observe(v, metric.WithAttributes(
			attribute.String("endpoint", k.endpoint),
			attribute.String("chain_id", k.chainID),
			attribute.String("scenario", k.scenario),
		))
	}
	return nil
}

func statusAttrFromError(err error) attribute.KeyValue {
	const key = "status"
	if err == nil {
		return attribute.String(key, "success")
	}
	return attribute.String(key, "failure")
}
