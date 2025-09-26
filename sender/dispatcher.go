package sender

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/sei-protocol/sei-load/generator"
	"github.com/sei-protocol/sei-load/stats"
	"github.com/sei-protocol/sei-load/types"
	"github.com/sei-protocol/sei-load/utils"
)

// Dispatcher continuously generates transactions and dispatches them to the sender
type Dispatcher struct {
	generator  generator.Generator
	prewarmGen utils.Option[generator.Generator] // Optional prewarm generator
	prewarmRPC string
	sender     TxSender

	// Statistics
	totalSent uint64
	mu        sync.RWMutex
	collector *stats.Collector
}

// NewDispatcher creates a new dispatcher
func NewDispatcher(gen generator.Generator, sender TxSender) *Dispatcher {
	return &Dispatcher{
		generator: gen,
		sender:    sender,
	}
}

// SetStatsCollector sets the statistics collector for this dispatcher
func (d *Dispatcher) SetStatsCollector(collector *stats.Collector) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.collector = collector
}

// SetPrewarmGenerator sets the prewarm generator for this dispatcher
func (d *Dispatcher) SetPrewarmGenerator(prewarmGen generator.Generator, rpcEndpoint string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.prewarmGen = utils.Some(prewarmGen)
	d.prewarmRPC = rpcEndpoint
}

// Prewarm runs the prewarm generator to completion before starting the main load test
func (d *Dispatcher) Prewarm(ctx context.Context) error {
	d.mu.RLock()
	prewarmGen := d.prewarmGen
	endpoint := d.prewarmRPC
	d.mu.RUnlock()

	gen, ok := prewarmGen.Get()
	if !ok {
		return nil
	} // No prewarming configured

	if endpoint == "" {
		return fmt.Errorf("prewarm endpoint not configured")
	}

	client, err := ethclient.Dial(endpoint)
	if err != nil {
		return fmt.Errorf("failed to connect to prewarm endpoint: %w", err)
	}
	defer client.Close()

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

		if err := waitForReceipt(ctx, client, tx); err != nil {
			return fmt.Errorf("failed waiting for prewarm receipt for account %s: %w", tx.Scenario.Sender.Address.Hex(), err)
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

func waitForReceipt(ctx context.Context, client *ethclient.Client, tx *types.LoadTx) error {
	const receiptTimeout = 30 * time.Second
	const pollInterval = 200 * time.Millisecond

	waitCtx, cancel := context.WithTimeout(ctx, receiptTimeout)
	defer cancel()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-waitCtx.Done():
			return fmt.Errorf("timeout waiting for receipt for tx %s", tx.EthTx.Hash().Hex())
		case <-ticker.C:
			receipt, err := client.TransactionReceipt(waitCtx, tx.EthTx.Hash())
			if err != nil {
				if errors.Is(err, ethereum.NotFound) {
					continue
				}
				log.Printf("🔥 Error fetching receipt for tx %s: %v", tx.EthTx.Hash().Hex(), err)
				continue
			}
			if receipt.Status != 1 {
				return fmt.Errorf("transaction %s failed with status %d", tx.EthTx.Hash().Hex(), receipt.Status)
			}
			return nil
		}
	}
}

// Start begins the dispatcher's transaction generation and sending loop
func (d *Dispatcher) Run(ctx context.Context) error {
	for ctx.Err() == nil {
		// Generate a transaction from main generator
		tx, ok := d.generator.Generate()
		if !ok {
			log.Print("Dispatcher: Generator returned no more transactions")
			return nil
		}

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
	}
}

// DispatcherStats contains statistics for the dispatcher
type DispatcherStats struct {
	TotalSent uint64
}
