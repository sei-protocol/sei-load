package sender

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/ethereum/go-ethereum"
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
	ID            int
	SeiChainID    string
	Endpoint      string
	BufferSize    int
	Tasks         int
	DryRun        bool
	Debug         bool
	TrackReceipts bool
	Collector     *stats.Collector
	Limiter       *rate.Limiter // Shared rate limiter for transaction sending
}

// Worker handles sending transactions to a specific endpoint
type Worker struct {
	cfg     *WorkerConfig
	txChan  chan *types.LoadTx
	sentTxs chan *types.LoadTx
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
		cfg:     cfg,
		txChan:  make(chan *types.LoadTx, cfg.BufferSize),
		sentTxs: make(chan *types.LoadTx, cfg.BufferSize),
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
		return w.watchTransactions(ctx, client)
	})
}

// Send queues a transaction for this worker to process
func (w *Worker) Send(ctx context.Context, tx *types.LoadTx) error {
	return utils.Send(ctx, w.txChan, tx)
}

func (w *Worker) watchTransactions(ctx context.Context, eth *ethclient.Client) error {
	if w.cfg.DryRun || !w.cfg.TrackReceipts {
		return nil
	}
	for ctx.Err() == nil {
		tx, err := utils.Recv(ctx, w.sentTxs)
		if err != nil {
			return err
		}
		// Cancel per-iteration; defer would leak contexts under sustained load.
		waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		if err := w.waitForReceipt(waitCtx, eth, tx); err != nil {
			log.Printf("❌ %v", err)
		}
		cancel()
	}
	return ctx.Err()
}

func (w *Worker) waitForReceipt(ctx context.Context, eth *ethclient.Client, tx *types.LoadTx) (_err error) {
	ctx, span := tracer.Start(ctx, "sender.check_receipt", trace.WithAttributes(
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
		// Record inside the span ctx so exemplars link to the trace.
		// worker_id stays off the histogram (cardinality); available via span.
		receiptLatency.Record(ctx, time.Since(start).Seconds(),
			metric.WithAttributes(
				attribute.String("scenario", tx.Scenario.Name),
				attribute.String("endpoint", w.cfg.Endpoint),
				attribute.String("chain_id", w.cfg.SeiChainID),
				statusAttrFromError(_err)),
		)
	}(time.Now())
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for ctx.Err() == nil {
		if _, err := utils.Recv(ctx, ticker.C); err != nil {
			return fmt.Errorf("timeout waiting for receipt for tx %s", tx.EthTx.Hash().Hex())
		}
		receipt, err := eth.TransactionReceipt(ctx, tx.EthTx.Hash())
		if err != nil {
			if errors.Is(err, ethereum.NotFound) {
				continue
			}
			log.Printf("❌ error getting receipt for tx %s: %v", tx.EthTx.Hash().Hex(), err)
			continue
		}
		// Receipt found - log status and return
		if receipt.Status != 1 {
			return fmt.Errorf("tx %s failed", tx.EthTx.Hash().Hex())
		}
		if w.cfg.Debug {
			log.Printf("✅ tx %s, %s, gas=%d succeeded\n", tx.Scenario.Name, tx.EthTx.Hash().Hex(), receipt.GasUsed)
		}
		return nil
	}
	return ctx.Err()
}

// runTxSender is the main worker loop that processes transactions
func (w *Worker) runTxSender(ctx context.Context, client *ethclient.Client) error {
	for ctx.Err() == nil {
		// Apply rate limiting before getting the next transaction
		if err := w.cfg.Limiter.Wait(ctx); err != nil {
			return err
		}

		tx, err := utils.Recv(ctx, w.txChan)
		if err != nil {
			return err
		}

		startTime := time.Now()
		// This goroutine solely owns tx between dequeue and the sentTxs hand-off,
		// so stamping the actual send-attempt time here is race-free (see LoadTx).
		tx.AttemptedSendTime = startTime
		err = w.sendTransaction(ctx, client, tx)
		// Record statistics if collector is available
		w.cfg.Collector.RecordTransaction(tx.Scenario.Name, w.cfg.Endpoint, time.Since(startTime), err == nil)
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
		// See receiptLatency above re: span-context recording + no worker_id.
		sendLatency.Record(ctx, time.Since(start).Seconds(),
			metric.WithAttributes(
				attribute.String("scenario", tx.Scenario.Name),
				attribute.String("endpoint", w.cfg.Endpoint),
				attribute.String("chain_id", w.cfg.SeiChainID),
				statusAttrFromError(_err)),
		)
	}(time.Now())
	if w.cfg.DryRun {
		// In dry-run mode, simulate processing time and mark as successful
		// Use very minimal delay to avoid channel overflow
		return utils.Sleep(ctx, 10*time.Microsecond) // Much faster simulation
	}

	// Send through go-ethereum so the same code path supports both HTTP(S) and WS(S) RPC.
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

	// Write to sentTxs channel without blocking
	utils.SendOrDrop(w.sentTxs, tx)
	return nil
}

// ChannelLength returns the current length of the worker's channel (for monitoring).
// This function is safe for concurrent calls.
func (w *Worker) ChannelLength() int { return len(w.txChan) }

// Endpoint returns the worker's endpoint
func (w *Worker) Endpoint() string { return w.cfg.Endpoint }
