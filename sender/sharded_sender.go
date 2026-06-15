package sender

import (
	"context"
	"fmt"

	"golang.org/x/time/rate"

	"github.com/sei-protocol/sei-load/config"
	"github.com/sei-protocol/sei-load/stats"
	"github.com/sei-protocol/sei-load/types"
	"github.com/sei-protocol/sei-load/utils/scope"
)

// ShardedSender implements TxSender with multiple workers, one per endpoint
type ShardedSender struct {
	cfg *config.LoadConfig
	collector     *stats.Collector
	limiter       *rate.Limiter // Shared rate limiter for transaction sending
	clients 			[]*ethClient
	shards 				[]*Queue[*types.LoadTx]
}

// NewShardedSender creates a new sharded sender with workers for each endpoint
func NewShardedSender(ctx context.Context, cfg *config.LoadConfig, limiter *rate.Limiter, collector *stats.Collector) (*ShardedSender, error) {
	if len(cfg.Endpoints) == 0 {
		return nil, fmt.Errorf("no endpoints configured")
	}
	var clients []*ethClient
	for id,endpoint := range cfg.Endpoints {
		clients = append(clients, newEthClient(&ethClientConfig {
			ChainID: cfg.SeiChainID,
			ID: id,
			Endpoint: endpoint,
			Tasks: cfg.Settings.TasksPerEndpoint,
			Debug: cfg.Settings.Debug,
			TrackReceipts: cfg.Settings.TrackReceipts,
			ReceiptsBuf: cfg.Settings.BufferSize,
		}))
	}
	numShards := len(cfg.Endpoints)
	poolSize := numShards * cfg.Settings.BufferSize
	pool := NewQueuePool[*types.LoadTx](poolSize)
	var shards []*Queue[*types.LoadTx]
	for range shards {
		q := pool.NewQueue()
		shards = append(shards,q)
		meterWorkerQueueLength(q)
	}
	return &ShardedSender{
		cfg:cfg,
		collector:collector,
		limiter:limiter,
		clients:clients,
		shards:shards,
	}, nil
}

// Start initializes and starts all workers
func (ss *ShardedSender) Run(ctx context.Context) error {
	return scope.Run(ctx, func(ctx context.Context, s scope.Scope) error {
		for _,client := range ss.clients {
			s.Spawn(func() error { return client.Run(ctx) })
		}
		for i, shard := range ss.shards {
			s.Spawn(func() error {
				for ctx.Err()==nil {
					// Apply rate limiting before getting the next transaction
					if err := ss.limiter.Wait(ctx); err != nil {
						return err
					}
					return w.runTxSender(ctx, client)
				}
				return ctx.Err()
			})
		}
		return nil
	})
}

// Send implements TxSender interface - calculates shard ID and routes to appropriate worker
func (s *ShardedSender) Send(ctx context.Context, tx *types.LoadTx) error {
	return s.shards[tx.ShardID(len(s.shards))].Send(ctx, tx)
}
