package sender

import (
	"context"
	"fmt"
	"log"
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
	cfg       *config.LoadConfig
	queue     *TxsQueue
	limiter   *rate.Limiter // Shared rate limiter for transaction sending
	collector *stats.Collector
	inclusion utils.Option[*stats.InclusionTracker]
}

// NewShardedSender creates a new sharded sender.
// Txs of each shard are sent sequentially, using a single eth client.
func NewShardedSender(cfg *config.LoadConfig, limiter *rate.Limiter, collector *stats.Collector, inclusion utils.Option[*stats.InclusionTracker]) *ShardedSender {
	return &ShardedSender{
		cfg:       cfg,
		queue:     NewTxsQueue(cfg.Settings.MaxInFlight),
		limiter:   limiter,
		collector: collector,
		inclusion: inclusion,
	}
}

func (ss *ShardedSender) Send(ctx context.Context, tx *types.LoadTx) error {
	return ss.queue.Push(ctx, tx)
}

func (ss *ShardedSender) Nonce(acc types.Account) uint64 {
	return ss.queue.Nonce(acc)
}

func (ss *ShardedSender) Flush(ctx context.Context) error {
	return ss.queue.WaitUntilEmpty(ctx)
}

func (ss *ShardedSender) handleSendFailure(ctx context.Context, client *ethClient, tx *types.LoadTx) error {
	if !tx.Scenario.Sender.Tracked {
		return nil
	}
	addr := tx.Scenario.Sender.Address
	for {
		if err := ss.limiter.Wait(ctx); err != nil {
			return err
		}
		// Nonce lookup is expected to succeed eventually.
		nonce, err := client.Nonce(ctx, addr)
		if err != nil {
			log.Printf("client.Nonce(): %v", err)
			continue
		}
		ss.queue.Reset(addr, nonce)
		return nil
	}
}

// Start initializes and starts all workers
func (ss *ShardedSender) Run(ctx context.Context) error {
	if len(ss.cfg.Endpoints) == 0 {
		return fmt.Errorf("no endpoints configured")
	}
	cancel := meteredSenders.MustRegister(ss)
	defer cancel()
	client, err := newEthClient(ctx, &ethClientConfig{
		ChainID:   ss.cfg.SeiChainID,
		Endpoints: ss.cfg.Endpoints,
		Collector: ss.collector,
	})
	if err != nil {
		return fmt.Errorf("newEthClient(): %w", err)
	}
	defer client.Close()
	return scope.Run(ctx, func(ctx context.Context, s scope.Scope) error {
		for {
			if err := ss.limiter.Wait(ctx); err != nil {
				return err
			}
			tx, err := ss.queue.PopReady(ctx)
			if err != nil {
				return err
			}
			addr := tx.Scenario.Sender.Address
			s.Spawn(func() error {
				defer ss.queue.PopSent(addr)
				if ss.cfg.Settings.DryRun {
					// In dry-run mode, simulate processing time and mark as successful
					// Use very minimal delay to avoid channel overflow
					if err := utils.Sleep(ctx, 10*time.Millisecond); err != nil {
						return err
					}
					if inclusion, ok := ss.inclusion.Get(); ok {
						inclusion.Register(tx)
					}
					return nil
				}
				if err := client.Send(ctx, tx); err != nil {
					log.Printf("client.Send(): %v", err)
					if err := ss.handleSendFailure(ctx, client, tx); err != nil {
						return err
					}
					return nil
				}
				if inclusion, ok := ss.inclusion.Get(); ok {
					inclusion.Register(tx)
				}
				return nil
			})
		}
	})
}
