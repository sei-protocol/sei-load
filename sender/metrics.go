package sender

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var meter = otel.Meter("github.com/sei-protocol/sei-load/sender")

var (
	sendLatency = must(meter.Float64Histogram(
		"send_latency",
		metric.WithDescription("Latency of sending transactions in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.1, 0.2, 0.3, 0.5, 1.0, 2.0, 3.0, 5.0, 10.0, 20.0)))

	receiptLatency = must(meter.Float64Histogram(
		"receipt_latency",
		metric.WithDescription("Latency of sending transactions in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.1, 0.2, 0.3, 0.5, 1.0, 2.0, 3.0, 5.0, 10.0, 20.0)))

	workerQueueLength = must(meter.Int64ObservableGauge(
		"worker_queue_length",
		metric.WithDescription("Length of the worker's queue"),
		metric.WithUnit("{count}"),
		metric.WithInt64Callback(func(ctx context.Context, observer metric.Int64Observer) error {
			meteredChainWorkers.lock.RLock()
			defer meteredChainWorkers.lock.RUnlock()
			for _, worker := range meteredChainWorkers.workers {
				observer.Observe(int64(worker.GetChannelLength()), metric.WithAttributes(
					attribute.String("endpoint", worker.GetEndpoint()),
					attribute.Int("worker_id", worker.id),
					attribute.String("chain_id", worker.seiChainID),
				))
			}
			return nil
		})))

	tpsAchieved = must(meter.Float64ObservableGauge(
		"tps_achieved",
		metric.WithDescription("Most recent TPS sample observed by the sender, per endpoint/scenario"),
		metric.WithUnit("{transactions}/s"),
		metric.WithFloat64Callback(observeTPS)))

	httpErrors = must(meter.Int64Counter(
		"http_errors",
		metric.WithDescription("HTTP error responses from the target endpoint, by status code"),
		metric.WithUnit("{errors}")))

	txsAccepted = must(meter.Int64Counter(
		"txs_accepted",
		metric.WithDescription("Transactions successfully submitted to an endpoint"),
		metric.WithUnit("{transactions}")))

	txsRejected = must(meter.Int64Counter(
		"txs_rejected",
		metric.WithDescription("Transactions rejected by the target or local client, by reason"),
		metric.WithUnit("{transactions}")))
)

// meteredChainWorkers is the registry the worker_queue_length callback reads.
var meteredChainWorkers = &chainWorkerObserver{
	workers: make(map[chainWorkerID]*Worker),
}

type chainWorkerObserver struct {
	lock    sync.RWMutex
	workers map[chainWorkerID]*Worker
}
type chainWorkerID struct {
	workerID int
	chainID  string
}

func meterWorkerQueueLength(worker *Worker) {
	meteredChainWorkers.lock.Lock()
	defer meteredChainWorkers.lock.Unlock()
	id := chainWorkerID{
		workerID: worker.id,
		chainID:  worker.seiChainID,
	}
	if _, exists := meteredChainWorkers.workers[id]; !exists {
		meteredChainWorkers.workers[id] = worker
	}
}

var tpsObserverRegistry = struct {
	lock    sync.RWMutex
	samples map[tpsSampleKey]float64
}{
	samples: make(map[tpsSampleKey]float64),
}

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

func must[V any](v V, err error) V {
	if err != nil {
		panic(err)
	}
	return v
}
