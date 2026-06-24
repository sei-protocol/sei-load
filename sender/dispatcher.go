package sender

import (
	"context"
	"log"
	mrand "math/rand/v2"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/sei-protocol/sei-load/generator"
	"github.com/sei-protocol/sei-load/stats"
)

// Dispatcher continuously generates transactions and dispatches them to the sender
type Dispatcher struct {
	sender     TxSender
	limiter      *rate.Limiter
	maxInFlight  int

	// Conservation counters (doc.go): scheduled = dropped + admitted,
	// admitted = succeeded + failed.
	totalSent uint64 // admitted, nil send error (succeeded)
	failed    uint64 // admitted, non-nil send error
	dropped   uint64
	mu        sync.RWMutex
	collector *stats.Collector
}

// NewDispatcher creates a new dispatcher in the legacy closed-loop arrival model.
func NewDispatcher(sender TxSender, collector *stats.Collector, maxInFlight int) *Dispatcher {
	return &Dispatcher{
		sender:       sender,
		limiter:      rate.NewLimiter(rate.Inf, 1),
		maxInFlight: maxInFlight,
	}
}

// Prewarm runs the prewarm generator to completion before starting the main load test
func (d *Dispatcher) RunPrewarm(ctx context.Context, rng *mrand.Rand, gen generator.Generator) error {
	// Prewarm runs before the scheduler paces anything, so it must self-pace off
	// the shared limiter or it floods the SUT.
	limiter := d.limiter

	processedAccounts := 0
	logInterval := 100

	// Run prewarm generator until completion
	for ctx.Err() == nil {
		if err := limiter.Wait(ctx); err != nil {
			return err
		}

		tx, ok := gen.Generate(rng)
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
func (d *Dispatcher) Run(ctx context.Context, rng *mrand.Rand, gen generator.Generator) error {
	for ctx.Err() == nil {
		// Generate a transaction from main generator
		tx, ok := gen.Generate(rng)
		if !ok {
			log.Print("Dispatcher: Generator returned no more transactions")
			return nil
		}

		// Stamp before hand-off while sole owner: race-free (see LoadTx). This is
		// the back-pressured enqueue time, not a true schedule instant.
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

// GetStats returns dispatcher statistics
func (d *Dispatcher) GetStats() DispatcherStats {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return DispatcherStats{
		TotalSent: d.totalSent,
		Failed:    d.failed,
		Dropped:   d.dropped,
	}
}

// DispatcherStats contains statistics for the dispatcher
type DispatcherStats struct {
	// TotalSent is the number of admitted sends that completed with a nil error
	// (succeeded).
	TotalSent uint64
	// Failed is the number of admitted open-loop sends that completed with a
	// non-nil error: counted, not lost (see package doc, Conservation:
	// admitted = succeeded + failed). Always 0 in closed-loop mode.
	Failed uint64
	// Dropped is the number of open-loop txs shed because in-flight was
	// saturated at their scheduled instant. Always 0 in closed-loop mode.
	Dropped uint64
}
