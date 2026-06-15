package sender

import (
	"context"
	"fmt"
	"log"

	"golang.org/x/time/rate"

	"github.com/sei-protocol/sei-load/config"
	"github.com/sei-protocol/sei-load/stats"
	"github.com/sei-protocol/sei-load/types"
	"github.com/sei-protocol/sei-load/utils/scope"
)

// ShardedSender implements TxSender with multiple workers, one per endpoint
type ShardedSender struct {
	cfg     *config.LoadConfig
	limiter *rate.Limiter // Shared rate limiter for transaction sending
	clients []*ethClient
	shards  []*Queue[*types.LoadTx]
}

// NewShardedSender creates a new sharded sender.
// Txs of each shard are sent sequentially, using a single eth client.
func NewShardedSender(cfg *config.LoadConfig, limiter *rate.Limiter, collector *stats.Collector) (*ShardedSender, error) {
	if len(cfg.Endpoints) == 0 {
		return nil, fmt.Errorf("no endpoints configured")
	}
	if cfg.GetNumShards() <= 0 {
		return nil, fmt.Errorf("no shards configured")
	}
	var clients []*ethClient
	for id, endpoint := range cfg.Endpoints {
		clients = append(clients, newEthClient(&ethClientConfig{
			ChainID:       cfg.SeiChainID,
			ID:            id,
			Endpoint:      endpoint,
			Tasks:         cfg.Settings.TasksPerEndpoint,
			DryRun:        cfg.Settings.DryRun,
			Debug:         cfg.Settings.Debug,
			TrackReceipts: cfg.Settings.TrackReceipts,
			ReceiptsBuf:   cfg.Settings.BufferSize,
			Collector:     collector,
		}))
	}
	pool := NewQueuePool[*types.LoadTx](cfg.TotalQueueSize())
	var shards []*Queue[*types.LoadTx]
	for range cfg.GetNumShards() {
		shards = append(shards, pool.NewQueue())
	}
	return &ShardedSender{
		cfg:     cfg,
		limiter: limiter,
		clients: clients,
		shards:  shards,
	}, nil
}

// Send implements TxSender interface - calculates shard ID and routes to appropriate worker
func (s *ShardedSender) Send(ctx context.Context, tx *types.LoadTx) error {
	return s.shards[tx.ShardID(len(s.shards))].Send(ctx, tx)
}

// Start initializes and starts all workers
func (ss *ShardedSender) Run(ctx context.Context) error {
	cancel := meteredSenders.MustRegister(ss)
	defer cancel()
	return scope.Run(ctx, func(ctx context.Context, s scope.Scope) error {
		for _, client := range ss.clients {
			s.Spawn(func() error { return client.Run(ctx) })
		}
		for i, shard := range ss.shards {
			s.Spawn(func() error {
				client := ss.clients[i%len(ss.clients)]
				for ctx.Err() == nil {
					if err := ss.limiter.Wait(ctx); err != nil {
						return err
					}
					tx, err := shard.Recv(ctx)
					if err != nil {
						return err
					}
					if err := client.Send(ctx, tx); err != nil {
						log.Printf("%v", err)
					}
				}
				return ctx.Err()
			})
		}
		return nil
	})
}

type ShardStats struct {
	ChainID   string
	ID        int
	Endpoint  string
	TxsQueued int
}

func (ss *ShardedSender) ShardStats() []ShardStats {
	var stats []ShardStats
	for i, shard := range ss.shards {
		stats = append(stats, ShardStats{
			ChainID:   ss.cfg.SeiChainID,
			ID:        i,
			Endpoint:  ss.clients[i%len(ss.clients)].cfg.Endpoint,
			TxsQueued: shard.Len(),
		})
	}
	return stats
}
