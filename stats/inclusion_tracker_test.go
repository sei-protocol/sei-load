package stats

import (
	"context"
	"math/big"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/sei-protocol/sei-load/types"
)

// MockBlockSource is a deterministic blockSource for tests. Setter style mirrors
// MockBlockStats: SetBlock seeds a block's tx hashes; fetches are counted.
type MockBlockSource struct {
	mu       sync.Mutex
	blocks   map[uint64][]common.Hash
	fetches  atomic.Int64
	fetchErr error
}

func NewMockBlockSource() *MockBlockSource {
	return &MockBlockSource{blocks: make(map[uint64][]common.Hash)}
}

func (m *MockBlockSource) SetBlock(n uint64, hashes ...common.Hash) *MockBlockSource {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.blocks[n] = hashes
	return m
}

func (m *MockBlockSource) BlockTxHashes(_ context.Context, n uint64) ([]common.Hash, error) {
	m.fetches.Add(1)
	if m.fetchErr != nil {
		return nil, m.fetchErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.blocks[n], nil
}

func (m *MockBlockSource) FetchCount() int64 { return m.fetches.Load() }

// newTestTracker builds a tracker wired to a mock source, skipping the live dial.
func newTestTracker(t *testing.T, reapAfter time.Duration, maxInflight int, src blockSource) *InclusionTracker {
	t.Helper()
	return newInclusionTrackerWithSource(
		NewInclusionTracker("test-chain", reapAfter, maxInflight), src)
}

// loadTx builds a LoadTx with a deterministic hash from nonce and an intended
// send time, so latency math is exact.
func loadTx(nonce uint64, intended time.Time) *types.LoadTx {
	eth := ethtypes.NewTx(&ethtypes.LegacyTx{
		Nonce:    nonce,
		GasPrice: big.NewInt(1),
		Gas:      21000,
		To:       &common.Address{},
		Value:    big.NewInt(0),
	})
	return &types.LoadTx{
		EthTx:            eth,
		Scenario:         &types.TxScenario{Name: "test"},
		IntendedSendTime: intended,
	}
}

func inflightLen(t *testing.T, tr *InclusionTracker) int {
	t.Helper()
	for s := range tr.state.Lock() {
		return len(s.inflight)
	}
	panic("unreachable")
}

// Test 1: a match stamps InclusionTime to the injected arrival, advances the
// included count, and shrinks the map.
func TestInclusion_MatchStamps(t *testing.T) {
	src := NewMockBlockSource()
	tr := newTestTracker(t, time.Minute, 100, src)

	intended := time.Unix(1000, 0)
	tx := loadTx(1, intended)
	tr.Register(tx)
	require.Equal(t, 1, inflightLen(t, tr))

	arrival := time.Unix(1002, 0)
	src.SetBlock(5, tx.EthTx.Hash())
	tr.matchBlock(context.Background(), 5, arrival)

	require.Equal(t, arrival, tx.InclusionTime, "InclusionTime is the header-arrival time")
	require.Equal(t, 0, inflightLen(t, tr), "matched tx leaves the registry")
	require.Equal(t, uint64(1), tr.Summary().Included)
}

// Test 2: reaping evicts an un-included tx as expired and leaves no leak.
func TestInclusion_ReapExpires(t *testing.T) {
	tr := newTestTracker(t, 10*time.Millisecond, 100, NewMockBlockSource())

	tx := loadTx(1, time.Now())
	tr.Register(tx)
	require.Equal(t, 1, inflightLen(t, tr))

	time.Sleep(20 * time.Millisecond)
	tr.reap()

	require.Equal(t, 0, inflightLen(t, tr), "reaped tx leaves the registry (no leak)")
	require.True(t, tx.InclusionTime.IsZero(), "reaped tx is never stamped")
	s := tr.Summary()
	require.Equal(t, uint64(1), s.Expired)
	require.Equal(t, uint64(0), s.Included)
}

// Test 3: reap vs late inclusion, both orderings → no double count, no panic.
func TestInclusion_ReapVsLateInclusion(t *testing.T) {
	t.Run("reap_first", func(t *testing.T) {
		src := NewMockBlockSource()
		tr := newTestTracker(t, time.Nanosecond, 100, src)
		tx := loadTx(1, time.Unix(1000, 0))
		tr.Register(tx)
		time.Sleep(time.Millisecond)
		tr.reap() // wins: expired
		src.SetBlock(5, tx.EthTx.Hash())
		tr.matchBlock(context.Background(), 5, time.Unix(1002, 0)) // no-op
		s := tr.Summary()
		require.Equal(t, uint64(1), s.Expired)
		require.Equal(t, uint64(0), s.Included)
		require.Equal(t, uint64(1), s.Expired+s.Included, "exactly one terminal state")
	})
	t.Run("match_first", func(t *testing.T) {
		src := NewMockBlockSource()
		tr := newTestTracker(t, time.Nanosecond, 100, src)
		tx := loadTx(1, time.Unix(1000, 0))
		tr.Register(tx)
		src.SetBlock(5, tx.EthTx.Hash())
		tr.matchBlock(context.Background(), 5, time.Unix(1002, 0)) // wins: included
		time.Sleep(time.Millisecond)
		tr.reap() // no-op
		s := tr.Summary()
		require.Equal(t, uint64(1), s.Included)
		require.Equal(t, uint64(0), s.Expired)
		require.Equal(t, uint64(1), s.Expired+s.Included, "exactly one terminal state")
	})
}

// Test 4: at cap, Register drops-and-counts; the map never exceeds the cap.
func TestInclusion_BoundedCap(t *testing.T) {
	const cap = 3
	tr := newTestTracker(t, time.Minute, cap, NewMockBlockSource())

	for i := range uint64(10) {
		tr.Register(loadTx(i, time.Now()))
		require.LessOrEqual(t, inflightLen(t, tr), cap, "map never exceeds cap")
	}
	s := tr.Summary()
	require.Equal(t, uint64(7), s.DroppedAtCap)
	require.Equal(t, cap, inflightLen(t, tr))
}

// Test 5: conservation identity registered == included + expired +
// inflightAtShutdown, table-driven over register/match/reap mixes.
func TestInclusion_Conservation(t *testing.T) {
	cases := []struct {
		name      string
		registered int
		matched    int
		reaped     int
	}{
		{"all_included", 5, 5, 0},
		{"all_expired", 5, 0, 5},
		{"mixed", 6, 2, 3},
		{"none_terminal", 4, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := NewMockBlockSource()
			tr := newTestTracker(t, time.Hour, 100, src)
			txs := make([]*types.LoadTx, tc.registered)
			for i := range txs {
				txs[i] = loadTx(uint64(i), time.Unix(1000, 0))
				tr.Register(txs[i])
			}
			for i := 0; i < tc.matched; i++ {
				src.SetBlock(uint64(i), txs[i].EthTx.Hash())
				tr.matchBlock(context.Background(), uint64(i), time.Unix(1001, 0))
			}
			// Reap the next `reaped` txs by forcing their registeredAt past cutoff.
			for s := range tr.state.Lock() {
				old := time.Now().Add(-2 * time.Hour)
				reaped := 0
				for i := tc.matched; i < tc.registered && reaped < tc.reaped; i++ {
					if e, ok := s.inflight[txs[i].EthTx.Hash()]; ok {
						e.registeredAt = old
						reaped++
					}
				}
			}
			tr.reap()

			s := tr.Summary()
			require.Equal(t, uint64(tc.registered),
				s.Included+s.Expired+s.InflightAtShutdown,
				"registered == included + expired + inflight_at_shutdown")
		})
	}
}

// Test 6: processHead counts gaps and never backfills (no fetch of skipped
// heights). The fake records every fetch call.
func TestInclusion_GapNoBackfill(t *testing.T) {
	src := NewMockBlockSource()
	tr := newTestTracker(t, time.Minute, 100, src)
	ctx := context.Background()

	last := tr.processHead(ctx, 10, time.Now(), 0) // seeds, no gap
	last = tr.processHead(ctx, 11, time.Now(), last)
	last = tr.processHead(ctx, 15, time.Now(), last) // gap 12,13,14
	_ = last

	require.Equal(t, int64(3), src.FetchCount(),
		"exactly one fetch per arriving head; skipped heights never fetched")
}

// Test 7: concurrent register/match/reap is race-free under -race.
func TestInclusion_ConcurrentRaceSafe(t *testing.T) {
	src := NewMockBlockSource()
	tr := newTestTracker(t, time.Millisecond, 10_000, src)
	const n = 500
	txs := make([]*types.LoadTx, n)
	for i := range txs {
		txs[i] = loadTx(uint64(i), time.Unix(1000, 0))
		src.SetBlock(uint64(i), txs[i].EthTx.Hash())
	}

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		for i := range txs {
			tr.Register(txs[i])
		}
	}()
	go func() {
		defer wg.Done()
		for i := range txs {
			tr.matchBlock(context.Background(), uint64(i), time.Unix(1001, 0))
		}
	}()
	go func() {
		defer wg.Done()
		for range 50 {
			tr.reap()
			time.Sleep(100 * time.Microsecond)
		}
	}()
	wg.Wait()
	tr.reap()

	s := tr.Summary()
	require.Equal(t, uint64(n), s.Included+s.Expired+s.InflightAtShutdown,
		"conservation holds under concurrency")
}
