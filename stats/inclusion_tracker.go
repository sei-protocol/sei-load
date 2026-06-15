package stats

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/sei-protocol/sei-load/types"
	"github.com/sei-protocol/sei-load/utils"
	"github.com/sei-protocol/sei-load/utils/service"
)

// blockSource yields the tx hashes of a single block by number. Consumer-side
// interface so tests can drive matching without a live chain.
type blockSource interface {
	BlockTxHashes(ctx context.Context, n uint64) ([]common.Hash, error)
}

// ethBlockSource is the production blockSource backed by an ethclient.
type ethBlockSource struct{ client *ethclient.Client }

func (s ethBlockSource) BlockTxHashes(ctx context.Context, n uint64) ([]common.Hash, error) {
	block, err := s.client.BlockByNumber(ctx, new(big.Int).SetUint64(n))
	if err != nil {
		return nil, err
	}
	txs := block.Transactions()
	hashes := make([]common.Hash, len(txs))
	for i, tx := range txs {
		hashes[i] = tx.Hash()
	}
	return hashes, nil
}

type entry struct {
	tx           *types.LoadTx
	registeredAt time.Time
}

type inclusionState struct {
	inflight           map[common.Hash]*entry
	included           uint64
	expired            uint64
	droppedAtCap       uint64
	inflightAtShutdown uint64
}

// InclusionTracker matches arriving blocks against in-flight txs to stamp
// InclusionTime. Conservation: registered == included + expired +
// inflight_at_shutdown, and registered ⊆ succeeded (see sender/doc.go).
type InclusionTracker struct {
	seiChainID  string
	reapAfter   time.Duration
	maxInflight int
	source      blockSource
	state       utils.Mutex[*inclusionState]
}

// defaultMaxInflight bounds the registry when the caller passes a non-positive
// cap (e.g. --max-in-flight unset in closed-loop): a zero cap would otherwise
// make len(inflight) >= cap always true and drop every registration.
const defaultMaxInflight = 10_000

// NewInclusionTracker builds a tracker bounded at maxInflight in-flight txs that
// reaps un-included txs after reapAfter. The block source is the production
// ethclient impl; tests inject via newInclusionTrackerWithSource.
func NewInclusionTracker(seiChainID string, reapAfter time.Duration, maxInflight int) *InclusionTracker {
	if maxInflight <= 0 {
		maxInflight = defaultMaxInflight
	}
	t := &InclusionTracker{
		seiChainID:  seiChainID,
		reapAfter:   reapAfter,
		maxInflight: maxInflight,
		state: utils.NewMutex(&inclusionState{
			inflight: make(map[common.Hash]*entry),
		}),
	}
	meterInclusionInflight(t)
	return t
}

func newInclusionTrackerWithSource(t *InclusionTracker, source blockSource) *InclusionTracker {
	t.source = source
	return t
}

// Register hands ownership of tx's InclusionTime to the tracker. Caller must
// invoke it only for successful sends, at send-completion (see worker.go), so
// registered ⊆ succeeded holds. At cap the tx is dropped and counted.
func (t *InclusionTracker) Register(tx *types.LoadTx) {
	hash := tx.EthTx.Hash()
	for s := range t.state.Lock() {
		// Cap check and insert share one critical section: race-free admission.
		if len(s.inflight) >= t.maxInflight {
			s.droppedAtCap++
			inclusionOutcome.Add(context.Background(), 1, metric.WithAttributes(
				attribute.String("chain_id", t.seiChainID),
				attribute.String("outcome", "dropped_at_cap"),
			))
			return
		}
		s.inflight[hash] = &entry{tx: tx, registeredAt: time.Now()}
	}
}

// Run subscribes to new heads and matches each arriving block once.
func (t *InclusionTracker) Run(ctx context.Context, firstEndpoint string) error {
	wsEndpoint := utils.GetWSEndpoint(firstEndpoint)
	if t.source == nil {
		client, err := ethclient.Dial(firstEndpoint)
		if err != nil {
			return fmt.Errorf("inclusion tracker: dial %s: %w", firstEndpoint, err)
		}
		defer client.Close()
		t.source = ethBlockSource{client: client}
	}
	return service.Run(ctx, func(ctx context.Context, s service.Scope) error {
		client, err := ethclient.Dial(wsEndpoint)
		if err != nil {
			return fmt.Errorf("inclusion tracker: connect WebSocket %s: %w", wsEndpoint, err)
		}
		headers := make(chan *ethtypes.Header)
		sub, err := client.SubscribeNewHead(ctx, headers)
		if err != nil {
			return fmt.Errorf("inclusion tracker: subscribe new heads: %w", err)
		}
		defer sub.Unsubscribe()
		s.SpawnBg(func() error {
			subErr, err := utils.Recv(ctx, sub.Err())
			if err != nil {
				return err
			}
			return subErr
		})
		s.Spawn(func() error { return t.reapLoop(ctx) })

		var lastSeen uint64 // 0 = unset; first head seeds it (no backfill).
		for ctx.Err() == nil {
			header, err := utils.Recv(ctx, headers)
			if err != nil {
				return err
			}
			lastSeen = t.processHead(ctx, header.Number.Uint64(), time.Now(), lastSeen)
		}
		return ctx.Err()
	})
}

// processHead handles one arriving head: counts any gap (no backfill), matches
// the block, and returns the new lastSeen. lastSeen==0 seeds on the first head.
func (t *InclusionTracker) processHead(ctx context.Context, num uint64, arrival time.Time, lastSeen uint64) uint64 {
	if lastSeen != 0 && num <= lastSeen {
		return lastSeen // duplicate or out-of-order head: no re-fetch, no spurious gap.
	}
	if lastSeen != 0 && num > lastSeen+1 {
		inclusionBlockGaps.Add(ctx, int64(num-lastSeen-1), metric.WithAttributes(
			attribute.String("chain_id", t.seiChainID)))
	}
	t.matchBlock(ctx, num, arrival)
	return num
}

// matchBlock fetches block num once and stamps every in-flight tx it includes
// with the header-arrival time.
func (t *InclusionTracker) matchBlock(ctx context.Context, num uint64, arrival time.Time) {
	// Explicit per-iteration cancel (not deferred-in-loop): bound the fetch.
	fetchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	hashes, err := t.source.BlockTxHashes(fetchCtx, num)
	cancel()
	if err != nil {
		// No retry (avoids piling RPC onto a struggling SUT): the block's txs go
		// unmatched and reap as expired. Surfaced so the undercount is visible.
		log.Printf("inclusion tracker: fetch block %d: %v", num, err)
		inclusionBlockFetchErrors.Add(ctx, 1, metric.WithAttributes(
			attribute.String("chain_id", t.seiChainID)))
		return
	}
	for s := range t.state.Lock() {
		for _, h := range hashes {
			e, ok := s.inflight[h]
			if !ok {
				continue
			}
			// Single writer of InclusionTime, under the lock; first observation
			// wins (delete-on-touch) — see reorg note in sender/doc.go.
			e.tx.InclusionTime = arrival
			delete(s.inflight, h)
			s.included++
			// Latency needs a submit reference; a zero IntendedSendTime means
			// "not scheduled" (e.g. prewarm txs), so skip the sample rather than
			// record a bogus epoch-based duration. See LoadTx contract.
			if !e.tx.IntendedSendTime.IsZero() {
				inclusionLatency.Record(ctx, arrival.Sub(e.tx.IntendedSendTime).Seconds(),
					metric.WithAttributes(attribute.String("chain_id", t.seiChainID)))
			}
		}
	}
}

// reapLoop sweeps every reapAfter; worst-case eviction latency is ~2×reapAfter
// (a tx registered just after a tick waits a full period for the next sweep) —
// a calibration nuance, not a conservation concern.
func (t *InclusionTracker) reapLoop(ctx context.Context) error {
	ticker := time.NewTicker(t.reapAfter)
	defer ticker.Stop()
	for ctx.Err() == nil {
		if _, err := utils.Recv(ctx, ticker.C); err != nil {
			log.Printf("inclusion tracker: reap loop: %v", err)
			continue
		}
		t.reap()
	}
	return ctx.Err()
}

// reap evicts txs in-flight longer than reapAfter as expired. Delete-on-touch
// under the lock races safely against matchBlock: whoever holds the lock first
// wins, no double count.
func (t *InclusionTracker) reap() {
	cutoff := time.Now().Add(-t.reapAfter)
	for s := range t.state.Lock() {
		for h, e := range s.inflight {
			if e.registeredAt.After(cutoff) {
				continue
			}
			delete(s.inflight, h)
			s.expired++
			inclusionOutcome.Add(context.Background(), 1, metric.WithAttributes(
				attribute.String("chain_id", t.seiChainID),
				attribute.String("outcome", "expired"),
			))
		}
	}
}

// InclusionSummary is the conservation tally. Read only after both workers and
// the tracker have joined, so inflightAtShutdown is final.
type InclusionSummary struct {
	Included           uint64
	Expired            uint64
	DroppedAtCap       uint64
	InflightAtShutdown uint64
}

// Summary snapshots the final tally; call once at shutdown after joins.
func (t *InclusionTracker) Summary() InclusionSummary {
	for s := range t.state.Lock() {
		s.inflightAtShutdown = uint64(len(s.inflight))
		return InclusionSummary{
			Included:           s.included,
			Expired:            s.expired,
			DroppedAtCap:       s.droppedAtCap,
			InflightAtShutdown: s.inflightAtShutdown,
		}
	}
	panic("unreachable")
}
