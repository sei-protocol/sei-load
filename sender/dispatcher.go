package sender

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/sei-protocol/sei-load/generator"
	"github.com/sei-protocol/sei-load/stats"
	"github.com/sei-protocol/sei-load/types"
	"github.com/sei-protocol/sei-load/utils"
	"github.com/sei-protocol/sei-load/utils/service"
)

// ArrivalModel selects how the dispatcher times transaction arrival.
type ArrivalModel string

const (
	// ArrivalClosedLoop is the legacy model: a tx is generated and sent only
	// when a sender is free, so a slow SUT slows the generator (coordinated
	// omission). Kept reachable as the regression baseline.
	ArrivalClosedLoop ArrivalModel = "closed_loop"
	// ArrivalOpenLoop schedules tx i at t₀ + i/λ independent of sender
	// availability; overdue txs are dropped and counted. See scheduler.go.
	ArrivalOpenLoop ArrivalModel = "open_loop"
)

// Dispatcher continuously generates transactions and dispatches them to the sender
type Dispatcher struct {
	generator  generator.Generator
	prewarmGen utils.Option[generator.Generator] // Optional prewarm generator
	sender     TxSender

	// Open-loop arrival configuration. arrivalModel defaults to closed-loop;
	// limiter and maxInFlight are only consulted in open-loop mode.
	arrivalModel ArrivalModel
	limiter      *rate.Limiter
	maxInFlight  int

	// Statistics
	totalSent uint64
	dropped   uint64
	mu        sync.RWMutex
	collector *stats.Collector
}

// NewDispatcher creates a new dispatcher in the legacy closed-loop arrival model.
func NewDispatcher(gen generator.Generator, sender TxSender) *Dispatcher {
	return &Dispatcher{
		generator:    gen,
		sender:       sender,
		arrivalModel: ArrivalClosedLoop,
	}
}

// SetOpenLoop switches the dispatcher to the open-loop arrival model, driven by
// the shared limiter (the one rate authority, also driven by the ramper) and
// bounded by maxInFlight concurrent sends. A non-positive maxInFlight is treated
// as 1 so admission control is always live.
func (d *Dispatcher) SetOpenLoop(limiter *rate.Limiter, maxInFlight int) {
	if maxInFlight < 1 {
		maxInFlight = 1
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.arrivalModel = ArrivalOpenLoop
	d.limiter = limiter
	d.maxInFlight = maxInFlight
}

// ArrivalModel reports the configured arrival model (for recording/reporting).
func (d *Dispatcher) ArrivalModel() ArrivalModel {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.arrivalModel
}

// SetStatsCollector sets the statistics collector for this dispatcher
func (d *Dispatcher) SetStatsCollector(collector *stats.Collector) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.collector = collector
}

// SetPrewarmGenerator sets the prewarm generator for this dispatcher
func (d *Dispatcher) SetPrewarmGenerator(prewarmGen generator.Generator) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.prewarmGen = utils.Some(prewarmGen)
}

// Prewarm runs the prewarm generator to completion before starting the main load test
func (d *Dispatcher) Prewarm(ctx context.Context) error {
	d.mu.RLock()
	prewarmGen := d.prewarmGen
	d.mu.RUnlock()

	gen, ok := prewarmGen.Get()
	if !ok {
		return nil
	} // No prewarming configured

	log.Print("🔥 Starting account prewarming...")
	processedAccounts := 0
	logInterval := 100

	// Run prewarm generator until completion
	for ctx.Err() == nil {
		tx, ok := gen.Generate()
		if !ok {
			break // Prewarming is complete
		}

		// Send the prewarming transaction
		if err := d.sender.Send(ctx, tx); err != nil {
			log.Printf("🔥 Failed to send prewarm transaction for account %s: %v", tx.Scenario.Sender.Address.Hex(), err)
			continue
		}

		processedAccounts++

		// Log progress periodically
		if processedAccounts%logInterval == 0 {
			log.Printf("🔥 Prewarming progress: %d accounts processed...", processedAccounts)
		}
	}

	log.Printf("🔥 Prewarming complete! Processed %d accounts", processedAccounts)
	return nil
}

// Run begins the dispatcher's transaction generation and sending loop, using
// the configured arrival model.
func (d *Dispatcher) Run(ctx context.Context) error {
	if d.ArrivalModel() == ArrivalOpenLoop {
		return d.runOpenLoop(ctx)
	}
	return d.runClosedLoop(ctx)
}

// runClosedLoop is the legacy model: generate-then-send in lockstep, so a slow
// SUT back-pressures the generator. Kept as the regression baseline.
func (d *Dispatcher) runClosedLoop(ctx context.Context) error {
	for ctx.Err() == nil {
		// Generate a transaction from main generator
		tx, ok := d.generator.Generate()
		if !ok {
			log.Print("Dispatcher: Generator returned no more transactions")
			return nil
		}

		// Stamp before hand-off: the dispatcher is sole owner here (tx just
		// returned by the generator, not yet enqueued), so this write is race-free.
		// This is the back-pressured enqueue time, not a true schedule instant.
		tx.IntendedSendTime = time.Now()

		// Send the transaction
		if err := d.sender.Send(ctx, tx); err != nil {
			return err
		}
		d.mu.Lock()
		d.totalSent++
		d.mu.Unlock()
	}
	return ctx.Err()
}

// runOpenLoop drives the open-loop scheduler (see scheduler.go), which owns the
// arrival clock (t₀, sequence index i) and the in-flight bound. Send tasks are
// spawned into a scope so they all complete on shutdown.
func (d *Dispatcher) runOpenLoop(ctx context.Context) error {
	d.mu.RLock()
	limiter, maxInFlight := d.limiter, d.maxInFlight
	d.mu.RUnlock()

	sched := newOpenLoopScheduler(d.generator, d.sender, limiter, maxInFlight, d.onSent)
	err := service.Run(ctx, func(ctx context.Context, s service.Scope) error {
		return sched.Run(ctx, s)
	})
	// Fold the scheduler's drop count into the dispatcher's accounting so the
	// final summary can report it.
	d.mu.Lock()
	d.dropped = sched.Dropped()
	d.mu.Unlock()
	return err
}

// onSent records a completed open-loop send. A successful send advances
// totalSent; the scheduler counts drops separately.
func (d *Dispatcher) onSent(tx *types.LoadTx, err error) {
	if err != nil {
		log.Printf("Scheduler: send failed (seq %d): %v", tx.SequenceIndex, err)
		return
	}
	d.mu.Lock()
	d.totalSent++
	d.mu.Unlock()
}

// StartBatch generates and sends a specific number of transactions then stops
func (d *Dispatcher) RunBatch(ctx context.Context, count int) error {
	if count <= 0 {
		return fmt.Errorf("count must be positive")
	}
	for i := range count {
		// Generate a transaction
		tx, ok := d.generator.Generate()
		if !ok {
			return fmt.Errorf("dispatcher: generator returned nil transaction (batch %d/%d)", i+1, count)
		}
		// Stamp before hand-off (see Run).
		tx.IntendedSendTime = time.Now()

		// Send the transaction
		if err := d.sender.Send(ctx, tx); err != nil {
			log.Printf("Dispatcher: Failed to send transaction %d/%d: %v", i+1, count, err)
			// Continue despite errors
		} else {
			d.mu.Lock()
			d.totalSent++
			d.mu.Unlock()
		}
	}
	return ctx.Err()
}

// GetStats returns dispatcher statistics
func (d *Dispatcher) GetStats() DispatcherStats {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return DispatcherStats{
		TotalSent: d.totalSent,
		Dropped:   d.dropped,
	}
}

// DispatcherStats contains statistics for the dispatcher
type DispatcherStats struct {
	TotalSent uint64
	// Dropped is the number of open-loop txs shed because in-flight was
	// saturated at their scheduled instant. Always 0 in closed-loop mode.
	Dropped uint64
}
