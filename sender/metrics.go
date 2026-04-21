package sender

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/sei-protocol/sei-load/observability"
)

// metricsBundle holds every instrument owned by this package. Acquired lazily
// on first access (see senderMetrics) so package init order can't capture the
// NoOp meter before observability.Setup has installed the real MeterProvider.
type metricsBundle struct {
	// --- existing instruments ---
	sendLatency       metric.Float64Histogram
	receiptLatency    metric.Float64Histogram
	workerQueueLength metric.Int64ObservableGauge

	// --- new instruments (per sei-load-observability design) ---
	// tpsAchieved is a gauge updated periodically by the stats package; it
	// reflects the sender's most recent TPS sample per endpoint/scenario.
	tpsAchieved metric.Float64ObservableGauge
	// httpErrors counts failed HTTP request attempts by status code. Omitted
	// for non-HTTP errors (e.g., DNS); those land in txsRejected.
	httpErrors metric.Int64Counter
	// txsAccepted counts successfully-submitted transactions.
	txsAccepted metric.Int64Counter
	// txsRejected counts transactions rejected by the target (or the local
	// client). Reason attribute narrows the failure mode.
	txsRejected metric.Int64Counter
}

var senderMetrics = sync.OnceValue(func() *metricsBundle {
	m := observability.Meter("github.com/sei-protocol/sei-load/sender")
	b := &metricsBundle{}

	latencyBoundaries := []float64{0.1, 0.2, 0.3, 0.5, 1.0, 2.0, 3.0, 5.0, 10.0, 20.0}
	b.sendLatency = must(m.Float64Histogram(
		"send_latency",
		metric.WithDescription("Latency of sending transactions in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(latencyBoundaries...)))
	b.receiptLatency = must(m.Float64Histogram(
		"receipt_latency",
		metric.WithDescription("Latency of sending transactions in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(latencyBoundaries...)))
	b.workerQueueLength = must(m.Int64ObservableGauge(
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

	b.tpsAchieved = must(m.Float64ObservableGauge(
		"tps_achieved",
		metric.WithDescription("Most recent TPS sample observed by the sender, per endpoint/scenario"),
		metric.WithUnit("{transactions}/s"),
		metric.WithFloat64Callback(observeTPS)))
	b.httpErrors = must(m.Int64Counter(
		"http_errors",
		metric.WithDescription("HTTP error responses from the target endpoint, by status code"),
		metric.WithUnit("{errors}")))
	b.txsAccepted = must(m.Int64Counter(
		"txs_accepted",
		metric.WithDescription("Transactions successfully submitted to an endpoint"),
		metric.WithUnit("{transactions}")))
	b.txsRejected = must(m.Int64Counter(
		"txs_rejected",
		metric.WithDescription("Transactions rejected by the target or local client, by reason"),
		metric.WithUnit("{transactions}")))

	return b
})

// meteredChainWorkers tracks Workers for the queue-length observable. Kept
// as a package-level value because the observable callback needs a stable
// reference; Worker registration is lock-synchronized.
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

// tpsObserverRegistry holds a per-(endpoint,chain_id,scenario) sample that
// Worker.processTransactions updates; the observable gauge reads it on each
// scrape. Writes are under write-lock; reads inside the callback are under
// read-lock.
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

// RecordTPSSample sets the latest TPS sample for a given (endpoint, chain_id,
// scenario) triple. Called from the sender as it rolls its TPS window.
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

// must panics if err is non-nil, otherwise returns v.
func must[V any](v V, err error) V {
	if err != nil {
		panic(err)
	}
	return v
}
