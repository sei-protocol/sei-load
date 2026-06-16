package sender

import (
	"context"
	"fmt"
	"log"

	"golang.org/x/time/rate"

	"github.com/sei-protocol/sei-load/config"
	"github.com/sei-protocol/sei-load/stats"
	"github.com/sei-protocol/sei-load/types"
<<<<<<< HEAD
	"github.com/sei-protocol/sei-load/utils/scope"
=======
	"github.com/sei-protocol/sei-load/utils"
	"github.com/sei-protocol/sei-load/utils/service"
>>>>>>> origin/main
)

// ShardedSender implements TxSender with multiple workers, one per endpoint
type ShardedSender struct {
	cfg     *config.LoadConfig
	limiter *rate.Limiter // Shared rate limiter for transaction sending
	clients []*ethClient
	shards  []*Queue[*types.LoadTx]
}

<<<<<<< HEAD
// NewShardedSender creates a new sharded sender.
// Txs of each shard are sent sequentially, using a single eth client.
func NewShardedSender(cfg *config.LoadConfig, limiter *rate.Limiter, collector *stats.Collector) (*ShardedSender, error) {
	if len(cfg.Endpoints) == 0 {
		return nil, fmt.Errorf("no endpoints configured")
	}
	numShards := cfg.GetNumShards()
	if numShards <= 0 {
		return nil, fmt.Errorf("no shards configured")
	}
	totalQueueSize := cfg.TotalQueueSize()
	if totalQueueSize <= 0 {
		return nil, fmt.Errorf("queue size has to be positive")
	}
	var clients []*ethClient
	for id, endpoint := range cfg.Endpoints {
		clients = append(clients, newEthClient(&ethClientConfig{
			ChainID:       cfg.SeiChainID,
			ID:            id,
=======
// NewShardedSender creates a new sharded sender with workers for each endpoint.
// inclusion, when present, is shared across all workers so each routes its
// successful sends to the one tracker.
func NewShardedSender(cfg *config.LoadConfig, limiter *rate.Limiter, collector *stats.Collector, inclusion utils.Option[*stats.InclusionTracker]) (*ShardedSender, error) {
	if len(cfg.Endpoints) == 0 {
		return nil, fmt.Errorf("no endpoints configured")
	}

	// Open-loop lets the scheduler own the arrival clock (see doc.go), so the
	// worker skips gating to avoid double-throttling; closed-loop keeps it.
	skipRateLimit := cfg.Settings.ArrivalModel == config.ArrivalModelOpenLoop

	workers := make([]*Worker, len(cfg.Endpoints))
	for i, endpoint := range cfg.Endpoints {
		workers[i] = NewWorker(&WorkerConfig{
			ID:            i,
			SeiChainID:    cfg.SeiChainID,
>>>>>>> origin/main
			Endpoint:      endpoint,
			Tasks:         cfg.Settings.TasksPerEndpoint,
			DryRun:        cfg.Settings.DryRun,
			Debug:         cfg.Settings.Debug,
			TrackReceipts: cfg.Settings.TrackReceipts,
			ReceiptsBuf:   cfg.Settings.BufferSize,
			Collector:     collector,
<<<<<<< HEAD
		}))
=======
			Limiter:       limiter,
			SkipRateLimit: skipRateLimit,
			Inclusion:     inclusion,
		})
>>>>>>> origin/main
	}
	pool := NewQueuePool[*types.LoadTx](totalQueueSize)
	var shards []*Queue[*types.LoadTx]
	for range numShards {
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
					tx, err := shard.Recv(ctx)
					if err != nil {
						return err
					}
					if err := ss.limiter.Wait(ctx); err != nil {
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
<<<<<<< HEAD
=======

// WorkerStats contains statistics for a single worker
type WorkerStats struct {
	WorkerID      int
	Endpoint      string
	ChannelLength int
}

// NumShards returns the number of shards (workers)
func (s *ShardedSender) NumShards() int { return len(s.workers) }
>>>>>>> origin/main
