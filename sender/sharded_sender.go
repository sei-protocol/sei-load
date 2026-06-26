package sender

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/time/rate"

	"github.com/sei-protocol/sei-load/config"
	"github.com/sei-protocol/sei-load/stats"
	"github.com/sei-protocol/sei-load/types"
	"github.com/sei-protocol/sei-load/utils"
	"github.com/sei-protocol/sei-load/utils/scope"
)

// ShardedSender implements TxSender with multiple workers, one per endpoint
type ShardedSender struct {
	cfg     *config.LoadConfig
	limiter *rate.Limiter // Shared rate limiter for transaction sending
	collector *stats.Collector
	inclusion utils.Option[*stats.InclusionTracker]
}

// NewShardedSender creates a new sharded sender.
// Txs of each shard are sent sequentially, using a single eth client.
func NewShardedSender(cfg *config.LoadConfig, limiter *rate.Limiter, collector *stats.Collector, inclusion utils.Option[*stats.InclusionTracker]) *ShardedSender {
	return &ShardedSender{
		cfg:     cfg,
		limiter: limiter,
		collector: collector,
		inclusion: inclusion,
	}
}

// Start initializes and starts all workers
func (ss *ShardedSender) Run(ctx context.Context, q *types.TxsQueue) error {	
	if len(ss.cfg.Endpoints) == 0 {
		return fmt.Errorf("no endpoints configured")
	}
	cancel := meteredSenders.MustRegister(ss)
	defer cancel()
	client,err := newEthClient(ctx,&ethClientConfig{
		ChainID:   ss.cfg.SeiChainID,
		Endpoints:  ss.cfg.Endpoints,
		Collector: ss.collector,
	})
	if err!=nil {
		return fmt.Errorf("newEthClient(): %w",err)
	}
	defer client.Close()
	return scope.Run(ctx, func(ctx context.Context, s scope.Scope) error {
		for {
			if err := ss.limiter.Wait(ctx); err != nil {
				return err
			}
			tx,ack,err := q.Pop(ctx)
			if err!=nil { return err }
			s.Spawn(func() error {
				defer ack()
				if ss.cfg.Settings.DryRun {
					// In dry-run mode, simulate processing time and mark as successful
					// Use very minimal delay to avoid channel overflow
					return utils.Sleep(ctx, 10*time.Microsecond) // Much faster simulation
				}
				if err := client.Send(ctx,tx); err != nil {
					return err
				}
				if inclusion, ok := ss.inclusion.Get(); ok {
					inclusion.Register(tx)
				}
				return nil
			})
		}
	})
}
