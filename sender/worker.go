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
	"github.com/sei-protocol/sei-load/types"
	"github.com/sei-protocol/sei-load/utils"
	"github.com/sei-protocol/sei-load/utils/service"
)

var tracer = otel.Tracer("github.com/sei-protocol/sei-load/sender")

type sendReq struct {
	tx *types.LoadTx
	done chan struct{}
}

type ethClientConfig struct {
	ChainID string
	ID int
	Endpoint string
	Tasks int
	Debug bool
	TrackReceipts bool
	ReceiptsBuf int
}	

type ethClient struct {
	cfg ethClientConfig	
	reqs chan sendReq
}

func (c *ethClient) Run(ctx context.Context) error {
	return service.Run(ctx, func(ctx context.Context, s service.Scope) error {
		u, err := url.Parse(c.cfg.Endpoint)
		if err != nil {
			return fmt.Errorf("parse endpoint %q: %w", c.cfg.Endpoint, err)
		}
		var opts []rpc.ClientOption
		switch u.Scheme {
		case "http", "https":
			opts = append(opts, rpc.WithHTTPClient(newHttpClient()))
		}
		rpcClient, err := rpc.DialOptions(ctx, c.cfg.Endpoint, opts...)
		if err != nil {
			return fmt.Errorf("rpc.Dial(%q): %w", c.cfg.Endpoint, err)
		}
		client := ethclient.NewClient(rpcClient)
		defer client.Close()
		receiptsChan := make(chan *types.LoadTx, c.cfg.ReceiptsBuf)
		for range c.cfg.Tasks {
			s.Spawn(func() error { return c.runSender(ctx, client, receiptsChan) }) 
		}
		if c.cfg.TrackReceipts {
			s.Spawn(func() error { return c.watchTransactions(ctx, client, receiptsChan) })
		}
		return nil
	})
}

// newHttpClient returns an otelhttp-wrapped client: injects traceparent on
// outbound, emits http.client.* metrics. Requires observability.Setup to have
// installed the global TextMapPropagator.
func newHttpClient() *http.Client {
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
	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: otelhttp.NewTransport(t),
	}
}

// newRPCClient returns a go-ethereum client configured for the endpoint scheme.
// HTTP(S) endpoints reuse the tuned otelhttp-backed transport; WS(S) endpoints
// use the default go-ethereum WebSocket transport.
func newEthClient(ctx context.Context, id int, endpoint string) *ethClient {
	return &ethClient {
		id: id,
		endpoint: endpoint,
		reqs: make(chan sendReq),
	}
}

// Send queues a transaction for this worker to process
func (c *ethClient) Send(ctx context.Context, tx *types.LoadTx) error {
	done := make(chan struct{})
	if err:=utils.Send(ctx,c.reqs,sendReq{tx,done}); err!=nil {
		return err
	}
	_,_,err := utils.RecvOrClosed(ctx,done)
	return err
}

func (c *ethClient) watchTransactions(ctx context.Context, eth *ethclient.Client, sentTxs <-chan *types.LoadTx) error {	
	for ctx.Err() == nil {
		tx, err := utils.Recv(ctx, sentTxs)
		if err != nil {
			return err
		}
		// Cancel per-iteration; defer would leak contexts under sustained load.
		waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		if err := c.waitForReceipt(waitCtx, eth, tx); err != nil {
			log.Printf("❌ %v", err)
		}
		cancel()
	}
	return ctx.Err()
}

func (c *ethClient) waitForReceipt(ctx context.Context, eth *ethclient.Client, tx *types.LoadTx) (_err error) {
	ctx, span := tracer.Start(ctx, "sender.check_receipt", trace.WithAttributes(
		attribute.String("seiload.scenario", tx.Scenario.Name),
		attribute.String("seiload.endpoint", c.endpoint),
		attribute.Int("seiload.worker_id", c.id),
		attribute.String("seiload.chain_id", c.chainID),
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
				attribute.String("endpoint", c.endpoint),
				attribute.String("chain_id", c.chainID),
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
		if c.cfg.Debug {
			log.Printf("✅ tx %s, %s, gas=%d succeeded\n", tx.Scenario.Name, tx.EthTx.Hash().Hex(), receipt.GasUsed)
		}
		return nil
	}
	return ctx.Err()
}

// runSender handles the tx send requests. 
func (c *ethClient) runSender(ctx context.Context, client *ethclient.Client, receiptsChan chan<- *types.LoadTx) error {
	for ctx.Err() == nil {
		tx, err := utils.Recv(ctx,c.reqs)
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
func (c *ethClient) sendTransaction(ctx context.Context, tx *types.LoadTx) (_err error) {
	ctx, span := tracer.Start(ctx, "sender.send_tx", trace.WithAttributes(
		attribute.String("seiload.scenario", tx.Scenario.Name),
		attribute.String("seiload.endpoint", c.endpoint),
		attribute.Int("seiload.worker_id", c.id),
		attribute.String("seiload.chain_id", c.chainID),
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
func (w *Worker) ChannelLength() int { return w.cfg.Queue.Len() }

// Endpoint returns the worker's endpoint
func (w *Worker) Endpoint() string { return w.cfg.Endpoint }
