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

// newTestTracker builds an open-loop tracker wired to a mock source, skipping
// the live dial. Open-loop is the default so latency-bearing tests exercise the
// inclusion_latency path; closed-loop is covered explicitly via newTestTrackerLoop.
func newTestTracker(t *testing.T, reapAfter time.Duration, maxInflight int, src blockSource) *InclusionTracker {
	t.Helper()
	return newTestTrackerLoop(t, reapAfter, maxInflight, src, true)
}

func newTestTrackerLoop(t *testing.T, reapAfter time.Duration, maxInflight int, src blockSource, openLoop bool) *InclusionTracker {
	t.Helper()
	return newInclusionTrackerWithSource(
		NewInclusionTracker("test-chain", reapAfter, maxInflight, openLoop), src)
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
		Sender:           common.Address{},
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

// Test 1b: closed-loop still counts inclusions and stamps InclusionTime; only
// the inclusion_latency sample is gated off (IntendedSendTime is enqueue time
// there, so arrival-IntendedSendTime is not a real inclusion latency). The
// latency histogram is a global OTel instrument with no test-readable provider,
// so this asserts the observable contract: counts and stamping are unaffected.
func TestInclusion_ClosedLoopCountsNoLatency(t *testing.T) {
	src := NewMockBlockSource()
	tr := newTestTrackerLoop(t, time.Minute, 100, src, false /* closed-loop */)
	require.False(t, tr.openLoop, "tracker built closed-loop: latency sample is gated")

	tx := loadTx(1, time.Unix(1000, 0))
	tr.Register(tx)
	arrival := time.Unix(1002, 0)
	src.SetBlock(5, tx.EthTx.Hash())
	tr.matchBlock(context.Background(), 5, arrival)

	require.Equal(t, arrival, tx.InclusionTime, "InclusionTime still stamped in closed-loop")
	require.Equal(t, uint64(1), tr.Summary().Included, "included count tracked in closed-loop")
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

// Test 4b: a non-positive cap falls back to the default bound instead of
// dropping every registration (len >= 0 is always true at cap 0).
func TestInclusion_NonPositiveCapFallsBack(t *testing.T) {
	tr := newTestTracker(t, time.Minute, 0, NewMockBlockSource())
	for i := range uint64(5) {
		tr.Register(loadTx(i, time.Now()))
	}
	require.Equal(t, 5, inflightLen(t, tr), "registrations are admitted, not all dropped")
	require.Equal(t, uint64(0), tr.Summary().DroppedAtCap)
}

// Test 4c: a non-positive reapAfter is floored to a positive default. A zero
// period would otherwise panic time.NewTicker and crash reapLoop, so we also
// confirm reapLoop starts and stops cleanly with the floored value.
func TestInclusion_NonPositiveReapAfterFloored(t *testing.T) {
	tr := newTestTracker(t, 0, 100, NewMockBlockSource())
	require.Positive(t, tr.reapAfter, "non-positive reapAfter is floored to a positive default")
	require.Equal(t, defaultInclusionReapAfter, tr.reapAfter)

	// reapLoop must not panic on the floored period; cancel returns it promptly.
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- tr.reapLoop(ctx) }()
	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
}

// Test 5: conservation identity registered == included + expired +
// inflightAtShutdown, table-driven over register/match/reap mixes.
func TestInclusion_Conservation(t *testing.T) {
	cases := []struct {
		name     string
		attempts int // Register calls
		cap      int // tracker maxInflight
		matched  int
		reaped   int
	}{
		{"all_included", 5, 100, 5, 0},
		{"all_expired", 5, 100, 0, 5},
		{"mixed", 6, 100, 2, 3},
		{"none_terminal", 4, 100, 0, 0},
		// attempts (10) > cap (5): 5 are dropped_at_cap (never registered), the
		// other 5 terminate as included/expired/inflight — proving dropped_at_cap
		// sits OUTSIDE the registered identity.
		{"with_cap_drops", 10, 5, 2, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := NewMockBlockSource()
			tr := newTestTracker(t, time.Hour, tc.cap, src)
			txs := make([]*types.LoadTx, tc.attempts)
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
				for i := tc.matched; i < tc.attempts && reaped < tc.reaped; i++ {
					if e, ok := s.inflight[txs[i].EthTx.Hash()]; ok {
						e.registeredAt = old
						reaped++
					}
				}
			}
			tr.reap()

			s := tr.Summary()
			// dropped_at_cap is excluded from the registered set, so every Register
			// attempt is accounted by exactly one of the four buckets.
			require.Equal(t, uint64(tc.attempts),
				s.Included+s.Expired+s.InflightAtShutdown+s.DroppedAtCap,
				"attempts == included + expired + inflight_at_shutdown + dropped_at_cap")
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
