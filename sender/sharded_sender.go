package sender

import (
	"context"
	"fmt"
	"log"
	"runtime"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"

	"github.com/sei-protocol/sei-load/config"
	"github.com/sei-protocol/sei-load/stats"
	"github.com/sei-protocol/sei-load/types"
	"github.com/sei-protocol/sei-load/utils"
	"github.com/sei-protocol/sei-load/utils/scope"
)

// ShardedSender implements TxSender across multiple endpoints.
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

func (ss *ShardedSender) getNonce(ctx context.Context, client *ethClient, addr common.Address) (uint64, error) {
	for ctx.Err() == nil {
		if err := ss.limiter.Wait(ctx); err != nil {
			return 0, err
		}
		// Nonce lookup is expected to succeed eventually.
		nonce, err := client.Nonce(ctx, addr)
		if err != nil {
			log.Printf("client.Nonce(): %v", err)
			continue
		}
		return nonce, nil
	}
	return 0, ctx.Err()
}

// Run initializes the sender loop.
func (ss *ShardedSender) Run(ctx context.Context) error {
	if len(ss.cfg.Endpoints) == 0 && !ss.cfg.Settings.DryRun {
		return fmt.Errorf("no endpoints configured")
	}
	signer := ethtypes.LatestSignerForChainID(ss.cfg.GetChainID())
	signing := semaphore.NewWeighted(int64(runtime.GOMAXPROCS(0)))
	client, err := newEthClient(ctx, &ethClientConfig{
		ChainID:   ss.cfg.SeiChainID,
		Endpoints: ss.cfg.Endpoints,
		Collector: ss.collector,
		DryRun:    ss.cfg.Settings.DryRun,
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
				// Sign the transaction.
				if err := signing.Acquire(ctx, 1); err != nil {
					return err
				}
				signedTx, err := ethtypes.SignTx(tx.EthTx, signer, tx.Scenario.Sender.PrivKey)
				signing.Release(1)
				if err != nil {
					return fmt.Errorf("sign tx: %w", err)
				}
				tx.EthTx = signedTx

				// Send the transaction.
				if err := client.Send(ctx, tx); err != nil {
					log.Printf("client.Send(): %v", err)
					if !tx.Scenario.Sender.Tracked {
						ss.queue.PopSent(addr)
						return nil
					}
					nonce, err := ss.getNonce(ctx, client, addr)
					if err != nil {
						return err
					}
					ss.queue.Reset(addr, nonce)
					return nil
				}

				// Queue for inclusion check.
				if inclusion, ok := ss.inclusion.Get(); ok {
					inclusion.Register(tx)
				}
				ss.queue.PopSent(addr)
				return nil
			})
		}
	})
}
