package sender

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"

	"github.com/sei-protocol/sei-load/types"
	"github.com/sei-protocol/sei-load/utils/service"
)

// fakeGenerator hands out blank LoadTx values until count is exhausted. It
// records the IntendedSendTime/SequenceIndex the scheduler stamped, since those
// are the open-loop schedule under test.
type fakeGenerator struct {
	mu        sync.Mutex
	remaining int
	issued    []*types.LoadTx
}

func newFakeGenerator(n int) *fakeGenerator { return &fakeGenerator{remaining: n} }

func (g *fakeGenerator) Generate() (*types.LoadTx, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.remaining == 0 {
		return nil, false
	}
	g.remaining--
	tx := &types.LoadTx{Scenario: &types.TxScenario{Name: "fake"}}
	g.issued = append(g.issued, tx)
	return tx, true
}

func (g *fakeGenerator) GenerateN(int) []*types.LoadTx { panic("unused") }
func (g *fakeGenerator) GetAccountPools() []types.AccountPool {
	return nil
}

func (g *fakeGenerator) issuedTxs() []*types.LoadTx {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make([]*types.LoadTx, len(g.issued))
	copy(out, g.issued)
	return out
}

// fakeSender records send count and optionally blocks for delay, modeling a
// slow SUT so the in-flight bound is exercised.
type fakeSender struct {
	delay time.Duration
	sent  atomic.Uint64
}

func (s *fakeSender) Send(ctx context.Context, _ *types.LoadTx) error {
	s.sent.Add(1)
	if s.delay > 0 {
		t := time.NewTimer(s.delay)
		defer t.Stop()
		select {
		case <-t.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

// runScheduler drives the scheduler in its own scope until the context expires,
// returning the scheduler so the caller can read Dropped().
func runScheduler(ctx context.Context, sched *openLoopScheduler) {
	_ = service.Run(ctx, func(ctx context.Context, s service.Scope) error {
		return sched.Run(ctx, s)
	})
}

// TestOpenLoopSchedule_TracksT0PlusIOverLambda is the core Done-criterion test:
// at a fixed λ against a fast sender, the IntendedSendTime stamped on tx i must
// track t₀ + i/λ within tolerance, independent of completion.
func TestOpenLoopSchedule_TracksT0PlusIOverLambda(t *testing.T) {
	const lambda = 200.0 // tx/s → 5ms gap
	gen := newFakeGenerator(40)
	snd := &fakeSender{}
	limiter := rate.NewLimiter(rate.Limit(lambda), 1)
	sched := newOpenLoopScheduler(gen, snd, limiter, 1024, nil)

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	start := time.Now()
	runScheduler(ctx, sched)

	issued := gen.issuedTxs()
	require.GreaterOrEqual(t, len(issued), 30, "scheduler should issue most txs within the window")

	gap := time.Second / time.Duration(lambda)
	// t₀ is the scheduler's internal start; bound it to [start, start+gap].
	t0 := issued[0].IntendedSendTime
	require.WithinDuration(t, start, t0, gap, "t₀ must be the campaign start")

	const tol = 2 * time.Millisecond
	for i, tx := range issued {
		require.Equal(t, uint64(i), tx.SequenceIndex, "sequence index must be monotonic from 0")
		want := t0.Add(time.Duration(i) * gap)
		require.WithinDuration(t, want, tx.IntendedSendTime, tol,
			"tx %d IntendedSendTime must track t₀ + i/λ", i)
	}
}

// TestOpenLoopSchedule_NotThrottledBySlowSender proves the arrival clock is not
// dragged by a slow SUT: with a sender far slower than the in-flight bound can
// absorb, the schedule must still advance at λ and the overrun must be dropped,
// not absorbed by blocking.
func TestOpenLoopSchedule_NotThrottledBySlowSender(t *testing.T) {
	const lambda = 500.0 // 2ms gap
	gen := newFakeGenerator(200)
	// Each send takes 100ms; with maxInFlight=4 the senders can sustain only
	// ~40 tx/s, an order of magnitude under λ → most txs must be dropped.
	snd := &fakeSender{delay: 100 * time.Millisecond}
	limiter := rate.NewLimiter(rate.Limit(lambda), 1)
	sched := newOpenLoopScheduler(gen, snd, limiter, 4, nil)

	ctx, cancel := context.WithTimeout(t.Context(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	runScheduler(ctx, sched)

	issued := gen.issuedTxs()
	gap := time.Second / time.Duration(lambda)

	// The clock must have kept advancing at λ despite the slow sender: the
	// schedule should have walked far past what the senders could absorb.
	require.GreaterOrEqual(t, len(issued), 100,
		"arrival clock must not be throttled by the slow sender")

	// Schedule accuracy still holds for the issued txs.
	t0 := issued[0].IntendedSendTime
	require.WithinDuration(t, start, t0, gap)
	const tol = 3 * time.Millisecond
	for i, tx := range issued {
		want := t0.Add(time.Duration(i) * gap)
		require.WithinDuration(t, want, tx.IntendedSendTime, tol,
			"tx %d schedule must hold under a slow sender", i)
	}

	// Overrun is dropped-and-counted, not blocked on.
	require.Positive(t, sched.Dropped(), "overrun must be counted as dropped")
	require.Equal(t, uint64(len(issued)), sched.Dropped()+snd.sent.Load(),
		"every issued tx is either sent or dropped exactly once")
}

// TestOpenLoopSchedule_HonorsRampedLambda verifies the schedule responds to a
// λ change applied via the shared limiter (the ramper's rate authority): after
// SetLimit, the inter-arrival gap tracks the new λ.
func TestOpenLoopSchedule_HonorsRampedLambda(t *testing.T) {
	gen := newFakeGenerator(1000)
	snd := &fakeSender{}
	// Start slow so the first gaps are large and easy to distinguish.
	limiter := rate.NewLimiter(rate.Limit(50), 1) // 20ms gap
	sched := newOpenLoopScheduler(gen, snd, limiter, 1024, nil)

	ctx, cancel := context.WithTimeout(t.Context(), 600*time.Millisecond)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		runScheduler(ctx, sched)
	}()

	// Let it run at 50 tps, then ramp to 500 tps and let it run more.
	time.Sleep(200 * time.Millisecond)
	limiter.SetLimit(rate.Limit(500)) // 2ms gap
	wg.Wait()

	issued := gen.issuedTxs()
	require.GreaterOrEqual(t, len(issued), 2, "scheduler must issue txs")

	// The min gap observed in the back half must reflect the faster λ: with a
	// 2ms target the later gaps are far under the initial 20ms gap.
	var minGap time.Duration = time.Hour
	for i := 1; i < len(issued); i++ {
		g := issued[i].IntendedSendTime.Sub(issued[i-1].IntendedSendTime)
		if g < minGap {
			minGap = g
		}
	}
	require.Less(t, minGap, 10*time.Millisecond,
		"ramped-up λ must shrink the inter-arrival gap below the initial 20ms")
}

// TestOpenLoopSchedule_StampsBeforeHandoff guards the LoadTx concurrency
// contract: the scheduler stamps IntendedSendTime and SequenceIndex before the
// send task can touch the tx. Run under -race to catch a regression.
func TestOpenLoopSchedule_StampsBeforeHandoff(t *testing.T) {
	gen := newFakeGenerator(50)
	snd := &fakeSender{}
	limiter := rate.NewLimiter(rate.Limit(1000), 1)

	var checked atomic.Uint64
	onSent := func(tx *types.LoadTx, err error) {
		require.NoError(t, err)
		require.False(t, tx.IntendedSendTime.IsZero(), "schedule must be stamped before send")
		checked.Add(1)
	}
	sched := newOpenLoopScheduler(gen, snd, limiter, 64, onSent)

	ctx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
	defer cancel()
	runScheduler(ctx, sched)

	require.Positive(t, checked.Load(), "onSent must observe stamped txs")
}
