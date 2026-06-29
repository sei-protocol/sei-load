package sender

import (
	"context"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/sei-protocol/sei-load/stats"
	"github.com/sei-protocol/sei-load/types"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("github.com/sei-protocol/sei-load/sender")

type ethClientConfig struct {
	ChainID   string
	Endpoints []string
	Collector *stats.Collector
}

type ethClient struct {
	cfg     *ethClientConfig
	clients []*ethclient.Client
}

func (c *ethClient) Close() {
	for _, eth := range c.clients {
		eth.Close()
	}
}

func newEthClient(ctx context.Context, cfg *ethClientConfig) (_ *ethClient, err error) {
	var clients []*ethclient.Client
	defer func() {
		if err != nil {
			for _, eth := range clients {
				eth.Close()
			}
		}
	}()
	for _, endpoint := range cfg.Endpoints {
		u, err := url.Parse(endpoint)
		if err != nil {
			return nil, fmt.Errorf("parse endpoint %q: %w", endpoint, err)
		}
		var opts []rpc.ClientOption
		switch u.Scheme {
		case "http", "https":
			opts = append(opts, rpc.WithHTTPClient(newHttpClient()))
		}
		rpcClient, err := rpc.DialOptions(ctx, endpoint, opts...)
		if err != nil {
			return nil, fmt.Errorf("rpc.Dial(%q): %w", endpoint, err)
		}
		clients = append(clients, ethclient.NewClient(rpcClient))
	}
	return &ethClient{cfg: cfg, clients: clients}, nil
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

// Addresses are sharded across endpoints so that each account is handled by a single RPC nonce.
// TODO: make this stickiness optional
func (c *ethClient) shardID(addr common.Address) int {
	addressBigInt := new(big.Int).SetBytes(addr.Bytes())
	mod := new(big.Int).Mod(addressBigInt, big.NewInt(int64(len(c.cfg.Endpoints))))
	return int(mod.Int64())
}

func (c *ethClient) Nonce(ctx context.Context, addr common.Address) (uint64, error) {
	return c.clients[c.shardID(addr)].NonceAt(ctx, addr, nil)
}

func (c *ethClient) Send(ctx context.Context, tx *types.LoadTx) (_err error) {
	id := c.shardID(tx.Scenario.Sender.Address)
	ctx, span := tracer.Start(ctx, "sender.send_tx", trace.WithAttributes(
		attribute.String("seiload.scenario", tx.Scenario.Name),
		attribute.String("seiload.endpoint", c.cfg.Endpoints[id]),
		attribute.Int("seiload.worker_id", id),
		attribute.String("seiload.chain_id", c.cfg.ChainID),
	))
	defer span.End()
	start := time.Now()
	// This goroutine solely owns tx between dequeue and the sentTxs hand-off,
	// so stamping the actual send-attempt time here is race-free (see LoadTx).
	tx.AttemptedSendTime = start
	err := c.clients[id].SendTransaction(ctx, tx.EthTx)
	// Record inside the span ctx so exemplars link to the trace.
	sendLatency.Record(ctx, time.Since(start).Seconds(),
		metric.WithAttributes(
			attribute.String("scenario", tx.Scenario.Name),
			attribute.String("endpoint", c.cfg.Endpoints[id]),
			attribute.String("chain_id", c.cfg.ChainID),
			statusAttrFromError(err)),
	)
	if err != nil {
		txsRejected.Add(ctx, 1, metric.WithAttributes(
			attribute.String("endpoint", c.cfg.Endpoints[id]),
			attribute.String("scenario", tx.Scenario.Name),
			attribute.String("reason", "rpc"),
		))
		span.RecordError(err)
	} else {
		txsAccepted.Add(ctx, 1, metric.WithAttributes(
			attribute.String("endpoint", c.cfg.Endpoints[id]),
			attribute.String("scenario", tx.Scenario.Name),
		))
	}
	c.cfg.Collector.RecordTransaction(tx.Scenario.Name, c.cfg.Endpoints[id], time.Since(start), err == nil)
	return err
}
