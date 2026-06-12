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

// asyncFakeSender models the production ShardedSender's send semantics: Send
// returns when the tx lands in a buffered channel (enqueue-and-return), NOT when
// the network send completes. Background workers dequeue and, after an optional
// per-send delay (a slow SUT), invoke tx.OnComplete to release the scheduler's
// in-flight permit. This is what exercises the HONEST in-flight bound (B2): a
// synchronous sender that blocks in Send would hide that the permit must be tied
// to real completion, not to enqueue.
type asyncFakeSender struct {
	ch    chan *types.LoadTx
	delay time.Duration
	sent  atomic.Uint64 // incremented when a send actually completes
}

// newAsyncFakeSender starts `workers` background senders draining a buffer of
// `buffer` slots. Mirrors a worker pool behind a bounded channel.
func newAsyncFakeSender(ctx context.Context, buffer, workers int, delay time.Duration) *asyncFakeSender {
	s := &asyncFakeSender{ch: make(chan *types.LoadTx, buffer), delay: delay}
	if workers < 1 {
		workers = 1
	}
	for range workers {
		go s.drain(ctx)
	}
	return s
}

func (s *asyncFakeSender) drain(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case tx := <-s.ch:
			if s.delay > 0 {
				t := time.NewTimer(s.delay)
				select {
				case <-t.C:
				case <-ctx.Done():
					t.Stop()
				}
			}
			s.sent.Add(1)
			if tx.OnComplete != nil {
				tx.OnComplete(nil)
			}
		}
	}
}

// Send enqueues without blocking on completion, returning at enqueue. If the
// buffer is full it blocks on the channel until a slot frees or ctx is done —
// like utils.Send in the real worker. The scheduler must never see this block
// throttle its clock because admission is gated by the in-flight permit upstream.
func (s *asyncFakeSender) Send(ctx context.Context, tx *types.LoadTx) error {
	select {
	case s.ch <- tx:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
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

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	snd := newAsyncFakeSender(ctx, 1024, 8, 0)
	limiter := rate.NewLimiter(rate.Limit(lambda), 1)
	sched := newOpenLoopScheduler(gen, snd, limiter, 1024, nil)

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
//
// It uses an ASYNC sender (Send returns at enqueue) so the drop count reflects
// the HONEST in-flight bound (B2): the permit is held until each send actually
// completes, so a slow SUT saturates maxInFlight and forces genuine load-shed —
// not buffer geometry. A synchronous sender would have masked this.
func TestOpenLoopSchedule_NotThrottledBySlowSender(t *testing.T) {
	const lambda = 500.0 // 2ms gap
	gen := newFakeGenerator(200)

	ctx, cancel := context.WithTimeout(t.Context(), 300*time.Millisecond)
	defer cancel()

	// Each send completes after 100ms; with maxInFlight=4 and only 4 draining
	// workers the senders can sustain only ~40 tx/s, an order of magnitude under
	// λ → most txs must be dropped. The buffer is deliberately small: with the
	// honest bound the in-flight permit (not the buffer) is the gate.
	const maxInFlight = 4
	snd := newAsyncFakeSender(ctx, maxInFlight, maxInFlight, 100*time.Millisecond)
	limiter := rate.NewLimiter(rate.Limit(lambda), 1)
	sched := newOpenLoopScheduler(gen, snd, limiter, maxInFlight, nil)

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

	// Overrun is dropped-and-counted, not blocked on. With ~150 issued in 300ms
	// and senders able to complete only ~a dozen, the vast majority must drop.
	require.Positive(t, sched.Dropped(), "overrun must be counted as dropped")
	require.Greater(t, sched.Dropped(), uint64(len(issued)/2),
		"a slow SUT must shed most of the load through the in-flight bound")
}

// gatedSender enqueues instantly but holds completion until release is called,
// letting a test observe the window where a tx is enqueued-but-not-completed.
type gatedSender struct {
	enqueued atomic.Uint64
	mu       sync.Mutex
	pending  []*types.LoadTx
}

func (s *gatedSender) Send(_ context.Context, tx *types.LoadTx) error {
	s.mu.Lock()
	s.pending = append(s.pending, tx)
	s.mu.Unlock()
	s.enqueued.Add(1)
	return nil // returns at enqueue, like the production worker
}

// completeAll fires OnComplete for every tx enqueued so far, releasing permits.
func (s *gatedSender) completeAll() {
	s.mu.Lock()
	pending := s.pending
	s.pending = nil
	s.mu.Unlock()
	for _, tx := range pending {
		if tx.OnComplete != nil {
			tx.OnComplete(nil)
		}
	}
}

// TestOpenLoopSchedule_PermitHeldUntilCompletion is the B2 guard: the in-flight
// permit must be tied to real send completion, not to enqueue. With maxInFlight=1
// and a sender that enqueues instantly but never completes, the first tx takes
// the only permit and holds it; every subsequent tx must drop. If the permit
// released at enqueue (the masked bug), the sender would have enqueued many.
func TestOpenLoopSchedule_PermitHeldUntilCompletion(t *testing.T) {
	gen := newFakeGenerator(100)
	snd := &gatedSender{}
	limiter := rate.NewLimiter(rate.Limit(1000), 1) // 1ms gap → many arrivals
	sched := newOpenLoopScheduler(gen, snd, limiter, 1, nil)

	ctx, cancel := context.WithTimeout(t.Context(), 120*time.Millisecond)
	defer cancel()
	runScheduler(ctx, sched)

	// Exactly one tx held the single permit through the whole run (never
	// completed), so the sender saw exactly one enqueue and everything else
	// dropped. Enqueue-time release would have let many through.
	require.Equal(t, uint64(1), snd.enqueued.Load(),
		"permit must be held until completion: only one tx may be in flight")
	require.Positive(t, sched.Dropped(), "arrivals past the held permit must drop")

	// Release the held permit; conservation still holds (issued == sent+dropped
	// is checked elsewhere). Drain to avoid a leaked OnComplete at teardown.
	snd.completeAll()
}

// TestOpenLoopSchedule_Conservation checks the accounting invariant: with a fast
// async sender that fully drains within the window, every issued tx is either
// completed (sent) or dropped exactly once — no permit leaks, no double-count.
func TestOpenLoopSchedule_Conservation(t *testing.T) {
	gen := newFakeGenerator(300)
	// Generous capacity so most txs complete; a few may drop on brief bursts.
	limiter := rate.NewLimiter(rate.Limit(1000), 1)

	var completed atomic.Uint64
	onSent := func(_ *types.LoadTx, err error) {
		require.NoError(t, err)
		completed.Add(1)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
	defer cancel()
	snd := newAsyncFakeSender(ctx, 256, 16, 0)
	sched := newOpenLoopScheduler(gen, snd, limiter, 256, onSent)
	runScheduler(ctx, sched)

	issued := uint64(len(gen.issuedTxs()))
	require.Positive(t, issued)
	// Allow the in-flight sends spawned just before deadline to settle.
	require.Eventually(t, func() bool {
		return completed.Load()+sched.Dropped() == issued
	}, time.Second, 5*time.Millisecond,
		"every issued tx must be completed or dropped exactly once (issued=%d sent=%d dropped=%d)",
		issued, completed.Load(), sched.Dropped())
}

// TestOpenLoopSchedule_HonorsRampedLambda verifies the schedule responds to a
// λ change applied via the shared limiter (the ramper's rate authority): after
// SetLimit, the inter-arrival gap tracks the new λ.
func TestOpenLoopSchedule_HonorsRampedLambda(t *testing.T) {
	gen := newFakeGenerator(1000)
	ctx, cancel := context.WithTimeout(t.Context(), 600*time.Millisecond)
	defer cancel()
	snd := newAsyncFakeSender(ctx, 1024, 8, 0)
	// Start slow so the first gaps are large and easy to distinguish.
	limiter := rate.NewLimiter(rate.Limit(50), 1) // 20ms gap
	sched := newOpenLoopScheduler(gen, snd, limiter, 1024, nil)

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
	ctx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
	defer cancel()
	snd := newAsyncFakeSender(ctx, 64, 8, 0)
	limiter := rate.NewLimiter(rate.Limit(1000), 1)

	var checked atomic.Uint64
	onSent := func(tx *types.LoadTx, err error) {
		require.NoError(t, err)
		require.False(t, tx.IntendedSendTime.IsZero(), "schedule must be stamped before send")
		checked.Add(1)
	}
	sched := newOpenLoopScheduler(gen, snd, limiter, 64, onSent)

	runScheduler(ctx, sched)

	require.Positive(t, checked.Load(), "onSent must observe stamped txs")
}
