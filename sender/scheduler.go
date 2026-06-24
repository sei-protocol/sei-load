package sender

import (
	"context"
	"log"
	mrand "math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"

	"github.com/sei-protocol/sei-load/generator"
	"github.com/sei-protocol/sei-load/types"
	"github.com/sei-protocol/sei-load/utils"
	"github.com/sei-protocol/sei-load/utils/service"
)

// openLoopScheduler issues tx i at t₀ + i/λ on an arrival clock decoupled from
// sender availability, bounding true in-flight with a semaphore and dropping
// (counting) overdue txs rather than throttling the clock. λ comes from the
// shared limiter (one rate authority). See the package doc for the open-loop
// arrival model: coordinated omission, drop-and-count, and the permit lifecycle.
type openLoopScheduler struct {
	generator   generator.Generator
	sender      TxSender
	limiter     *rate.Limiter
	inflight    *semaphore.Weighted
	onSent      func(tx *types.LoadTx, err error)
	maxInFlight int

	// Written by Run; read after Run returns or concurrently via Dropped/Admitted.
	// See package doc (Conservation) for scheduled = dropped + admitted.
	dropped  atomic.Uint64
	admitted atomic.Uint64
}

// minScheduleRate floors λ so a zero/negative limit can't divide-by-zero or
// yield a +Inf gap; the degenerate λ=Inf/TPS=0 case is rejected up front
// (config.Settings.Validate).
const minScheduleRate = 1e-9

// maxScheduleRate ceilings λ so a mid-run ramp toward rate.Inf can't collapse
// the gap to 0 (gap = 1s/λ truncates to 0 once λ exceeds ~1e9 = 1s in ns),
// which would stall nextSend and spin the scheduler. Startup validation only
// rejects λ=Inf at config time; this defends the ramper driving λ→Inf after the
// run starts. The ceiling is ~1e8 TPS — orders of magnitude above any realistic
// load — so it never affects a real run; at λ=maxScheduleRate the gap is 10ns>0.
const maxScheduleRate = 1e8

func newOpenLoopScheduler(
	gen generator.Generator,
	snd TxSender,
	limiter *rate.Limiter,
	maxInFlight int,
	onSent func(tx *types.LoadTx, err error),
) *openLoopScheduler {
	if maxInFlight < 1 {
		maxInFlight = 1
	}
	return &openLoopScheduler{
		generator:   gen,
		sender:      snd,
		limiter:     limiter,
		inflight:    semaphore.NewWeighted(int64(maxInFlight)),
		onSent:      onSent,
		maxInFlight: maxInFlight,
	}
}

// Dropped returns the number of ticks shed so far because in-flight was saturated.
func (s *openLoopScheduler) Dropped() uint64 { return s.dropped.Load() }

// Admitted returns the admitted-tick count (took a permit and drew a tx), for
// the conservation invariant in tests/audit; mirrors Dropped.
func (s *openLoopScheduler) Admitted() uint64 { return s.admitted.Load() }

// Run drives the arrival clock until ctx is canceled or the generator is
// exhausted, spawning each accepted tx as a send task bounded by the in-flight
// semaphore. See the package doc for the arrival model.
func (s *openLoopScheduler) Run(ctx context.Context, rng *mrand.Rand, scope service.Scope) error {
	t0 := time.Now()
	nextSend := t0
	var i uint64

	for ctx.Err() == nil {
		// Sample λ per step (honors a ramping limit; at fixed λ sums to t₀ + i/λ).
		lambda := float64(s.limiter.Limit())
		if lambda < minScheduleRate {
			lambda = minScheduleRate
		}
		if lambda > maxScheduleRate {
			lambda = maxScheduleRate
		}
		gap := time.Duration(float64(time.Second) / lambda)

		// Sleep to the absolute instant (not "gap from now") to avoid drift.
		if err := utils.SleepUntil(ctx, nextSend); err != nil {
			return err
		}

		// Snapshot the schedule; clock/index advance only when a tick resolves to
		// a real arrival, so the terminal exhaust probe isn't counted (see doc).
		intendedSendTime := nextSend
		seqIndex := i

		// Admit before generating: a dropped tick must not consume a seeded
		// generator draw (determinism). TryAcquire is non-blocking.
		ok := s.inflight.TryAcquire(1)
		if !ok {
			s.dropped.Add(1)
			nextSend = nextSend.Add(gap)
			i++
			continue
		}

		tx, ok := s.generator.Generate(rng)
		if !ok {
			// Generator drained: not an arrival — release the permit and stop.
			s.inflight.Release(1)
			log.Print("Scheduler: generator returned no more transactions")
			return nil
		}

		// Stamp while sole owner (LoadTx concurrency contract), then advance.
		tx.IntendedSendTime = intendedSendTime
		tx.SequenceIndex = seqIndex
		s.admitted.Add(1)
		nextSend = nextSend.Add(gap)
		i++

		// complete releases the permit and reports the result, exactly once: the
		// worker invokes it via tx.OnComplete after the real send; the Once guards
		// the enqueue-failure fallback below from racing the worker.
		var once sync.Once
		complete := func(err error) {
			once.Do(func() {
				s.inflight.Release(1)
				if s.onSent != nil {
					s.onSent(tx, err)
				}
			})
		}
		tx.OnComplete = complete

		scope.Spawn(func() error {
			// On enqueue failure the tx never reaches a worker; complete here so the
			// permit isn't leaked. A send error must not tear down the campaign.
			if err := s.sender.Send(ctx, tx); err != nil {
				complete(err)
			}
			return nil
		})
	}
	return ctx.Err()
}
