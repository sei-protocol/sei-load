package sender

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/time/rate"

	"github.com/sei-protocol/sei-load/stats"
	"github.com/sei-protocol/sei-load/types"
	"github.com/sei-protocol/sei-load/utils"
	"github.com/sei-protocol/sei-load/utils/service"
)

var tracer = otel.Tracer("github.com/sei-protocol/sei-load/sender")

type WorkerConfig struct {
	ID         int
	SeiChainID string
	Endpoint   string
	BufferSize int
	Tasks      int
	DryRun     bool
	Debug      bool
	Collector  *stats.Collector
	Limiter    *rate.Limiter // Shared rate authority; nil disables gating.
	// SkipRateLimit opts a worker out of limiter gating. Zero value (false) is the
	// safe default (gate when Limiter is set); set true only in open-loop, where
	// the scheduler owns the clock (see doc.go).
	SkipRateLimit bool
	// Inclusion, when present, receives each successful send at send-completion so
	// the tracker can stamp InclusionTime (see doc.go). None disables tracking.
	Inclusion utils.Option[*stats.InclusionTracker]
}

// Worker handles sending transactions to a specific endpoint
type Worker struct {
	cfg    *WorkerConfig
	txChan chan *types.LoadTx
}

// HttpClientOption configures the Transport used by newHttpClient.
type HttpClientOption func(*http.Transport)

// WithMaxIdleConns overrides the global idle-connection pool size.
func WithMaxIdleConns(n int) HttpClientOption {
	return func(t *http.Transport) { t.MaxIdleConns = n }
}

// WithMaxIdleConnsPerHost overrides the per-host idle-connection pool size.
// Scale with goroutine count to avoid TCP re-dial on each completion.
func WithMaxIdleConnsPerHost(n int) HttpClientOption {
	return func(t *http.Transport) { t.MaxIdleConnsPerHost = n }
}

// newHttpTransport is the base transport factory. Exists separately so tests
// can inspect the unwrapped *http.Transport; newHttpClient returns it wrapped
// in otelhttp, whose inner transport isn't publicly accessible.
func newHttpTransport(opts ...HttpClientOption) *http.Transport {
	t := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          500,
		MaxIdleConnsPerHost:   50,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableKeepAlives:     false,
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// newHttpClient returns an otelhttp-wrapped client: injects traceparent on
// outbound, emits http.client.* metrics. Requires observability.Setup to have
// installed the global TextMapPropagator.
func newHttpClient(opts ...HttpClientOption) *http.Client {
	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: otelhttp.NewTransport(newHttpTransport(opts...)),
	}
}

// newRPCClient returns a go-ethereum client configured for the endpoint scheme.
// HTTP(S) endpoints reuse the tuned otelhttp-backed transport; WS(S) endpoints
// use the default go-ethereum WebSocket transport.
func newRPCClient(ctx context.Context, endpoint string, opts ...HttpClientOption) (*ethclient.Client, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse endpoint %q: %w", endpoint, err)
	}

	switch u.Scheme {
	case "http", "https":
		rpcClient, err := rpc.DialOptions(ctx, endpoint, rpc.WithHTTPClient(newHttpClient(opts...)))
		if err != nil {
			return nil, err
		}
		return ethclient.NewClient(rpcClient), nil
	case "ws", "wss", "":
		return ethclient.DialContext(ctx, endpoint)
	default:
		return nil, fmt.Errorf("unsupported RPC scheme %q for endpoint %s", u.Scheme, endpoint)
	}
}

// NewWorker creates a new worker for a specific endpoint
func NewWorker(cfg *WorkerConfig) *Worker {
	w := &Worker{
		cfg:    cfg,
		txChan: make(chan *types.LoadTx, cfg.BufferSize),
	}
	meterWorkerQueueLength(w)
	return w
}

// Start begins the worker's processing loop
func (w *Worker) Run(ctx context.Context) error {
	client, err := newRPCClient(ctx, w.cfg.Endpoint)
	if err != nil {
		return fmt.Errorf("dial %s: %w", w.cfg.Endpoint, err)
	}
	defer client.Close()
	return service.Run(ctx, func(ctx context.Context, s service.Scope) error {
		// Start multiple goroutines that share the same channel and RPC client.
		for range w.cfg.Tasks {
			s.Spawn(func() error { return w.runTxSender(ctx, client) })
		}
		return nil
	})
}

// Send queues a transaction for this worker to process
func (w *Worker) Send(ctx context.Context, tx *types.LoadTx) error {
	return utils.Send(ctx, w.txChan, tx)
}

// runTxSender is the main worker loop that processes transactions
func (w *Worker) runTxSender(ctx context.Context, client *ethclient.Client) error {
	for ctx.Err() == nil {
		// Closed-loop gates on the limiter before dequeue; open-loop skips it.
		if !w.cfg.SkipRateLimit && w.cfg.Limiter != nil {
			if err := w.cfg.Limiter.Wait(ctx); err != nil {
				return err
			}
		}

		tx, err := utils.Recv(ctx, w.txChan)
		if err != nil {
			return err
		}

		startTime := time.Now()
		// Sole owner between dequeue and hand-off: stamp is race-free (see LoadTx).
		tx.AttemptedSendTime = startTime
		// schedule_lag self-check: only open-loop txs carry a true scheduled
		// instant. A zero IntendedSendTime (prewarm) is excluded here; the
		// closed-loop enqueue time is excluded at the run level (the verdict gates
		// on the arrival model, see stats.EvaluateScheduleLag).
		if !tx.IntendedSendTime.IsZero() {
			w.cfg.Collector.RecordScheduleLag(startTime.Sub(tx.IntendedSendTime))
		}
		err = w.sendTransaction(ctx, client, tx)
		// OnComplete must fire only after the real send returns — that is what
		// bounds true unacked in-flight (see doc.go). Nil on closed-loop/batch.
		if tx.OnComplete != nil {
			tx.OnComplete(err)
		}
		w.cfg.Collector.RecordTransaction(tx.Scenario.Name, w.cfg.Endpoint, time.Since(startTime), err == nil)
		// Register at send-completion, only on success: registered ⊆ succeeded.
		// (The tracker is wired only for live runs — see main.go; DryRun never
		// gets a tracker, so simulated sends are not inclusion-tracked.)
		if err == nil {
			if t, ok := w.cfg.Inclusion.Get(); ok {
				t.Register(tx)
			}
		}
		if err != nil {
			log.Printf("%v", err)
		}
	}
	return ctx.Err()
}

// sendTransaction sends a single transaction to the endpoint
func (w *Worker) sendTransaction(ctx context.Context, client *ethclient.Client, tx *types.LoadTx) (_err error) {
	ctx, span := tracer.Start(ctx, "sender.send_tx", trace.WithAttributes(
		attribute.String("seiload.scenario", tx.Scenario.Name),
		attribute.String("seiload.endpoint", w.cfg.Endpoint),
		attribute.Int("seiload.worker_id", w.cfg.ID),
		attribute.String("seiload.chain_id", w.cfg.SeiChainID),
	))
	defer func(start time.Time) {
		if _err != nil {
			span.RecordError(_err)
		}
		span.End()
		// Record inside the span ctx so exemplars link to the trace; worker_id
		// stays off the histogram (cardinality), available via the span.
		sendLatency.Record(ctx, time.Since(start).Seconds(),
			metric.WithAttributes(
				attribute.String("scenario", tx.Scenario.Name),
				attribute.String("endpoint", w.cfg.Endpoint),
				attribute.String("chain_id", w.cfg.SeiChainID),
				statusAttrFromError(_err)),
		)
	}(time.Now())
	if w.cfg.DryRun {
		return utils.Sleep(ctx, 10*time.Microsecond) // minimal delay, no RPC
	}

	if err := client.SendTransaction(ctx, tx.EthTx); err != nil {
		txsRejected.Add(ctx, 1, metric.WithAttributes(
			attribute.String("endpoint", w.cfg.Endpoint),
			attribute.String("scenario", tx.Scenario.Name),
			attribute.String("reason", "rpc"),
		))
		return fmt.Errorf("Worker %d: Failed to send transaction: %w", w.cfg.ID, err)
	}

	txsAccepted.Add(ctx, 1, metric.WithAttributes(
		attribute.String("endpoint", w.cfg.Endpoint),
		attribute.String("scenario", tx.Scenario.Name),
	))
	return nil
}

// ChannelLength returns the current length of the worker's channel (for monitoring).
// This function is safe for concurrent calls.
func (w *Worker) ChannelLength() int { return len(w.txChan) }

// Endpoint returns the worker's endpoint
func (w *Worker) Endpoint() string { return w.cfg.Endpoint }
