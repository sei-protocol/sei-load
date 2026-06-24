package sender

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/sei-protocol/sei-load/stats"
	"github.com/sei-protocol/sei-load/types"
	"github.com/sei-protocol/sei-load/utils"
	"github.com/sei-protocol/sei-load/utils/scope"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("github.com/sei-protocol/sei-load/sender")

type sendReq struct {
	tx   *types.LoadTx
	done chan error
}

type ethClientConfig struct {
	ChainID   string
	ID        int
	Endpoint  string
	Tasks     int
	Debug     bool
	DryRun    bool
	Collector *stats.Collector
	Inclusion utils.Option[*stats.InclusionTracker]
}

type ethClient struct {
	cfg  *ethClientConfig
	reqs chan sendReq
}

func (c *ethClient) Run(ctx context.Context) error {
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
	return scope.Run(ctx, func(ctx context.Context, s scope.Scope) error {
		for range c.cfg.Tasks {
			s.Spawn(func() error { return c.runSender(ctx, client) })
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

func newEthClient(cfg *ethClientConfig) *ethClient {
	return &ethClient{
		cfg:  cfg,
		reqs: make(chan sendReq),
	}
}

// Send queues a transaction for this endpoint client to process.
func (c *ethClient) Send(ctx context.Context, tx *types.LoadTx) error {
	done := make(chan error, 1)
	if err := utils.Send(ctx, c.reqs, sendReq{tx, done}); err != nil {
		return err
	}
	err, recvErr := utils.Recv(ctx, done)
	if recvErr != nil {
		return recvErr
	}
	return err
}

// runSender handles the tx send requests.
func (c *ethClient) runSender(ctx context.Context, client *ethclient.Client) error {
	for ctx.Err() == nil {
		req, err := utils.Recv(ctx, c.reqs)
		if err != nil {
			return err
		}

		startTime := time.Now()
		// This goroutine solely owns tx between dequeue and the sentTxs hand-off,
		// so stamping the actual send-attempt time here is race-free (see LoadTx).
		req.tx.AttemptedSendTime = startTime
		err = c.sendTx(ctx, client, req.tx)
		req.done <- err
		c.cfg.Collector.RecordTransaction(req.tx.Scenario.Name, c.cfg.Endpoint, time.Since(startTime), err == nil)
		if err == nil {
			if t, ok := c.cfg.Inclusion.Get(); ok {
				t.Register(req.tx)
			}
		}
	}
	return ctx.Err()
}

func (c *ethClient) sendTx(ctx context.Context, eth *ethclient.Client, tx *types.LoadTx) (_err error) {
	ctx, span := tracer.Start(ctx, "sender.send_tx", trace.WithAttributes(
		attribute.String("seiload.scenario", tx.Scenario.Name),
		attribute.String("seiload.endpoint", c.cfg.Endpoint),
		attribute.Int("seiload.worker_id", c.cfg.ID),
		attribute.String("seiload.chain_id", c.cfg.ChainID),
	))
	defer func(start time.Time) {
		if _err != nil {
			span.RecordError(_err)
		}
		span.End()
		// Record inside the span ctx so exemplars link to the trace.
		sendLatency.Record(ctx, time.Since(start).Seconds(),
			metric.WithAttributes(
				attribute.String("scenario", tx.Scenario.Name),
				attribute.String("endpoint", c.cfg.Endpoint),
				attribute.String("chain_id", c.cfg.ChainID),
				statusAttrFromError(_err)),
		)
	}(time.Now())
	if c.cfg.DryRun {
		// In dry-run mode, simulate processing time and mark as successful
		// Use very minimal delay to avoid channel overflow
		return utils.Sleep(ctx, 10*time.Microsecond) // Much faster simulation
	}

	// Send through go-ethereum so the same code path supports both HTTP(S) and WS(S) RPC.
	if err := eth.SendTransaction(ctx, tx.EthTx); err != nil {
		txsRejected.Add(ctx, 1, metric.WithAttributes(
			attribute.String("endpoint", c.cfg.Endpoint),
			attribute.String("scenario", tx.Scenario.Name),
			attribute.String("reason", "rpc"),
		))
		return fmt.Errorf("eth.SendTransaction(): %w", err)
	}

	txsAccepted.Add(ctx, 1, metric.WithAttributes(
		attribute.String("endpoint", c.cfg.Endpoint),
		attribute.String("scenario", tx.Scenario.Name),
	))
	return nil
}
