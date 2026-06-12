package sender

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"

	"github.com/sei-protocol/sei-load/generator"
	"github.com/sei-protocol/sei-load/types"
	"github.com/sei-protocol/sei-load/utils"
	"github.com/sei-protocol/sei-load/utils/service"
)

// openLoopScheduler issues transactions on an open-loop arrival clock: tx i is
// scheduled at t₀ + i/λ regardless of whether any sender is free. This is the
// coordinated-omission fix — when the SUT slows, the arrival clock does NOT
// slow with it; overdue txs are dropped and counted instead (REL8/REL9 load
// shedding), so measured latency reflects the backlog rather than hiding it.
//
// λ comes from the shared rate.Limiter, which the ramper drives via SetLimit
// (one rate authority). The limiter is read as a clock source here, not as a
// permit gate: the schedule advances by 1/λ per tx, sampling λ each step so a
// ramping λ is honored. At a fixed λ this telescopes to exactly t₀ + i/λ.
//
// In-flight work is bounded by a semaphore. A tx that cannot acquire a permit
// without blocking is dropped (the senders are saturated); the scheduler never
// blocks on capacity, which is what keeps the arrival clock unthrottled.
//
// The permit is held until the actual network send completes, not until the tx
// is enqueued: the scheduler stamps tx.OnComplete with the release, and the
// worker invokes it after sendTransaction returns. So maxInFlight bounds true
// unacked in-flight sends and dropped measures genuine load-shed, not buffer
// geometry (the worker's send is async enqueue-and-return). Note: schedule_lag
// (PLT-463), not the drop count, remains the primary coordinated-omission gate.
type openLoopScheduler struct {
	generator   generator.Generator
	sender      TxSender
	limiter     *rate.Limiter
	inflight    *utils.Semaphore
	onSent      func(tx *types.LoadTx, err error)
	maxInFlight int

	// dropped counts txs shed because in-flight was saturated at their
	// scheduled instant. Read after Run returns, or concurrently via Dropped.
	dropped atomic.Uint64
}

// minScheduleRate floors λ when computing the inter-arrival gap so a zero or
// negative limit cannot make the gap divide-by-zero or blow up to a +Inf
// duration. It does not bound how long the scheduler sleeps: a small-but-finite
// λ still yields a long gap. The degenerate λ=Inf / TPS=0 open-loop case is
// rejected up front in config validation (see config.Settings.Validate).
const minScheduleRate = 1e-9

func newOpenLoopScheduler(
	gen generator.Generator,
	snd TxSender,
	limiter *rate.Limiter,
	maxInFlight int,
	onSent func(tx *types.LoadTx, err error),
) *openLoopScheduler {
	return &openLoopScheduler{
		generator:   gen,
		sender:      snd,
		limiter:     limiter,
		inflight:    utils.NewSemaphore(maxInFlight),
		onSent:      onSent,
		maxInFlight: maxInFlight,
	}
}

// Dropped returns the number of txs shed so far because in-flight was saturated.
func (s *openLoopScheduler) Dropped() uint64 { return s.dropped.Load() }

// Run drives the open-loop arrival clock until the context is canceled or the
// generator is exhausted. Each accepted tx is sent on its own task spawned into
// scope, bounded by the in-flight semaphore; the send task releases the permit
// on completion, so the bound covers true in-flight (enqueue + send), not just
// queue depth.
func (s *openLoopScheduler) Run(ctx context.Context, scope service.Scope) error {
	t0 := time.Now()
	nextSend := t0
	var i uint64

	for ctx.Err() == nil {
		// Advance the schedule by one inter-arrival gap. Sampling λ here (not
		// once up front) honors a ramping limit; at fixed λ the running sum is
		// exactly t₀ + i/λ.
		lambda := float64(s.limiter.Limit())
		if lambda < minScheduleRate {
			lambda = minScheduleRate
		}
		gap := time.Duration(float64(time.Second) / lambda)

		// Sleep until this tx's scheduled instant. Sleeping to an absolute
		// instant (not "gap from now") prevents per-tx scheduling slop from
		// accumulating into clock drift.
		if err := utils.SleepUntil(ctx, nextSend); err != nil {
			return err
		}

		tx, ok := s.generator.Generate()
		if !ok {
			log.Print("Scheduler: generator returned no more transactions")
			return nil
		}

		// Stamp the TRUE scheduled instant and the arrival index while we are
		// the sole owner of tx (see LoadTx concurrency contract).
		tx.IntendedSendTime = nextSend
		tx.SequenceIndex = i

		nextSend = nextSend.Add(gap)
		i++

		// Non-blocking admission: if senders are saturated, drop and count
		// rather than block — blocking here would throttle the arrival clock
		// and reintroduce coordinated omission.
		release, ok := s.inflight.TryAcquire()
		if !ok {
			s.dropped.Add(1)
			continue
		}

		// Complete fires exactly once when the send finishes: it releases the
		// in-flight permit and reports the result. The worker invokes it via
		// tx.OnComplete after the real send (the happy path), so the permit is
		// held for the whole unacked-in-flight window, not just the enqueue.
		// The Once guards the enqueue-failure fallback below from racing the
		// worker — though by construction only one path runs.
		var once sync.Once
		complete := func(err error) {
			once.Do(func() {
				release()
				if s.onSent != nil {
					s.onSent(tx, err)
				}
			})
		}
		tx.OnComplete = complete

		scope.Spawn(func() error {
			// Send returns at enqueue; the permit release is deferred to the
			// worker via tx.OnComplete on real completion. If the enqueue itself
			// fails (e.g. ctx canceled), the tx never reaches a worker, so we
			// complete it here to avoid leaking the permit.
			if err := s.sender.Send(ctx, tx); err != nil {
				complete(err)
			}
			// A send error must not tear down the campaign; the closed-loop
			// path logs-and-continues identically. Drops/errors are surfaced
			// via counters, not by returning here.
			return nil
		})
	}
	return ctx.Err()
}
