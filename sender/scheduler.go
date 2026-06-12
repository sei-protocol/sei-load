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

// openLoopScheduler issues tx i at t₀ + i/λ on an arrival clock decoupled from
// sender availability, bounding true in-flight with a semaphore and dropping
// (counting) overdue txs rather than throttling the clock. λ comes from the
// shared limiter (one rate authority). See the package doc for the open-loop
// arrival model: coordinated omission, drop-and-count, and the permit lifecycle.
type openLoopScheduler struct {
	generator   generator.Generator
	sender      TxSender
	limiter     *rate.Limiter
	inflight    *utils.Semaphore
	onSent      func(tx *types.LoadTx, err error)
	maxInFlight int

	// dropped and admitted partition scheduled ticks; see the package doc
	// (Conservation) for the invariant scheduled = dropped + admitted. A tick
	// whose Generate returns no work (generator exhausted under a held permit) is
	// neither: the permit is released and no arrival is counted. Written by the
	// Run goroutine; read after Run returns or concurrently via Dropped/Admitted.
	dropped  atomic.Uint64
	admitted atomic.Uint64
}

// minScheduleRate floors λ in the inter-arrival gap so a zero/negative limit
// cannot divide-by-zero or yield a +Inf gap. It does not cap the sleep: a small
// finite λ still yields a long gap. The degenerate λ=Inf / TPS=0 case is
// rejected up front (see config.Settings.Validate).
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

// Dropped returns the number of ticks shed so far because in-flight was saturated.
func (s *openLoopScheduler) Dropped() uint64 { return s.dropped.Load() }

// Admitted returns the admitted-tick count (ticks that took a permit and drew a
// real tx). It exposes the conservation invariant for tests/audit, mirroring
// Dropped; no production path consumes it (only Dropped folds into the run
// summary).
func (s *openLoopScheduler) Admitted() uint64 { return s.admitted.Load() }

// Run drives the arrival clock until ctx is canceled or the generator is
// exhausted, spawning each accepted tx as a send task bounded by the in-flight
// semaphore. See the package doc for the arrival model.
func (s *openLoopScheduler) Run(ctx context.Context, scope service.Scope) error {
	t0 := time.Now()
	nextSend := t0
	var i uint64

	for ctx.Err() == nil {
		// Sample λ per step (honors a ramping limit; at fixed λ sums to t₀ + i/λ).
		lambda := float64(s.limiter.Limit())
		if lambda < minScheduleRate {
			lambda = minScheduleRate
		}
		gap := time.Duration(float64(time.Second) / lambda)

		// Sleep to the absolute instant (not "gap from now") to avoid drift.
		if err := utils.SleepUntil(ctx, nextSend); err != nil {
			return err
		}

		// Advance the arrival clock for this scheduled tick before admission, so
		// the schedule walks at λ whether or not the tx is admitted.
		intendedSendTime := nextSend
		seqIndex := i
		nextSend = nextSend.Add(gap)
		i++

		// Admit BEFORE generating: a non-blocking TryAcquire that never throttles
		// the arrival clock (see package doc: coordinated omission). On a drop the
		// tick is counted but the generator is NOT advanced — a dropped slot
		// consumes zero seeded-stream draws and no signing CPU, so admitted txs are
		// a deterministic prefix of the seeded sequence regardless of how many
		// ticks were shed (the per-stream reproducibility contract, PLT-456).
		release, ok := s.inflight.TryAcquire()
		if !ok {
			s.dropped.Add(1)
			continue
		}

		// Permit held: now draw the next tx from the (seeded) generator.
		tx, ok := s.generator.Generate()
		if !ok {
			// No work left: release the permit we will not use and stop.
			release()
			log.Print("Scheduler: generator returned no more transactions")
			return nil
		}

		// Stamp the scheduled instant and arrival index while sole owner (see
		// LoadTx concurrency contract).
		tx.IntendedSendTime = intendedSendTime
		tx.SequenceIndex = seqIndex
		s.admitted.Add(1)

		// complete fires once on send completion: releases the permit and reports
		// the result. The worker invokes it via tx.OnComplete after the real send
		// (see package doc: permit lifecycle). The Once guards against the
		// enqueue-failure fallback below racing the worker.
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
			// Send returns at enqueue; the worker releases the permit on real
			// completion. On enqueue failure the tx never reaches a worker, so
			// complete here to avoid leaking the permit.
			if err := s.sender.Send(ctx, tx); err != nil {
				complete(err)
			}
			// A send error must not tear down the campaign; surfaced via counters.
			return nil
		})
	}
	return ctx.Err()
}
