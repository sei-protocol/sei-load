package sender

import (
	"context"
	"fmt"

	"golang.org/x/time/rate"

	"github.com/sei-protocol/sei-load/config"
	"github.com/sei-protocol/sei-load/stats"
	"github.com/sei-protocol/sei-load/types"
	"github.com/sei-protocol/sei-load/utils/service"
)

// ShardedSender implements TxSender with multiple workers, one per endpoint
type ShardedSender struct {
	workers []*Worker
}

// NewShardedSender creates a new sharded sender with workers for each endpoint
func NewShardedSender(cfg *config.LoadConfig, limiter *rate.Limiter, collector *stats.Collector) (*ShardedSender, error) {
	if len(cfg.Endpoints) == 0 {
		return nil, fmt.Errorf("no endpoints configured")
	}

	workers := make([]*Worker, len(cfg.Endpoints))
	for i, endpoint := range cfg.Endpoints {
		workers[i] = NewWorker(&WorkerConfig{
			ID:         i,
			SeiChainID: cfg.SeiChainID,
			Endpoint:   endpoint,
			BufferSize: cfg.Settings.BufferSize,
			Tasks:      cfg.Settings.TasksPerEndpoint,
			Collector:  collector,
			Limiter:    limiter,
		})
	}

	return &ShardedSender{workers: workers}, nil
}

// Start initializes and starts all workers
func (s *ShardedSender) Run(ctx context.Context) error {
	return service.Run(ctx, func(ctx context.Context, scope service.Scope) error {
		for _, worker := range s.workers {
			scope.Spawn(func() error { return worker.Run(ctx) })
		}
		return nil
	})
}

// Send implements TxSender interface - calculates shard ID and routes to appropriate worker
func (s *ShardedSender) Send(ctx context.Context, tx *types.LoadTx) error {
	// Calculate shard ID based on the transaction
	shardID := tx.ShardID(len(s.workers))
	// Send to the appropriate worker
	return s.workers[shardID].Send(ctx, tx)
}

// GetWorkerStats returns statistics for all workers
func (s *ShardedSender) GetWorkerStats() []WorkerStats {
	stats := make([]WorkerStats, len(s.workers))
	for i, worker := range s.workers {
		stats[i] = WorkerStats{
			WorkerID:      i,
			Endpoint:      worker.Endpoint(),
			ChannelLength: worker.ChannelLength(),
		}
	}
	return stats
}

// WorkerStats contains statistics for a single worker
type WorkerStats struct {
	WorkerID      int
	Endpoint      string
	ChannelLength int
}

// GetNumShards returns the number of shards (workers)
func (s *ShardedSender) NumShards() int { return len(s.workers) }
