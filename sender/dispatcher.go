package sender

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/ethclient"
	"golang.org/x/time/rate"

	"github.com/sei-protocol/sei-load/generator"
	"github.com/sei-protocol/sei-load/stats"
	"github.com/sei-protocol/sei-load/types"
	"github.com/sei-protocol/sei-load/utils"
)

// Dispatcher continuously generates transactions and dispatches them to the sender
type Dispatcher struct {
	generator          generator.Generator
	prewarmGen         utils.Option[generator.Generator] // Optional prewarm generator
	prewarmRPC         string
	prewarmRate        float64
	prewarmMaxInFlight int
	sender             TxSender

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
func (d *Dispatcher) SetPrewarmGenerator(prewarmGen generator.Generator, rpcEndpoint string, rate float64, maxInFlight int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.prewarmGen = utils.Some(prewarmGen)
	d.prewarmRPC = rpcEndpoint
	if maxInFlight <= 0 {
		maxInFlight = 1
	}
	d.prewarmRate = rate
	d.prewarmMaxInFlight = maxInFlight
}

// Prewarm runs the prewarm generator to completion before starting the main load test
func (d *Dispatcher) Prewarm(ctx context.Context) error {
	d.mu.RLock()
	prewarmGen := d.prewarmGen
	endpoint := d.prewarmRPC
	rateLimit := d.prewarmRate
	maxInFlight := d.prewarmMaxInFlight
	d.mu.RUnlock()

	gen, ok := prewarmGen.Get()
	if !ok {
		return nil
	} // No prewarming configured

	if endpoint == "" {
		return fmt.Errorf("prewarm endpoint not configured")
	}

	if maxInFlight <= 0 {
		maxInFlight = 1
	}

	client, err := ethclient.Dial(endpoint)
	if err != nil {
		return fmt.Errorf("failed to connect to prewarm endpoint: %w", err)
	}
	defer client.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var limiter *rate.Limiter
	if rateLimit > 0 {
		burst := int(math.Ceil(rateLimit))
		if burst < maxInFlight {
			burst = maxInFlight
		}
		limiter = rate.NewLimiter(rate.Limit(rateLimit), burst)
	}

	type prewarmResult struct {
		account string
		txHash  string
		err     error
	}

	log.Print("🔥 Starting account prewarming...")
	logInterval := 100
	processedAccounts := 0
	inFlight := 0
	results := make(chan prewarmResult, maxInFlight)
	generatorDone := false
	var prewarmErr error

	handleResult := func(res prewarmResult) {
		inFlight--
		if res.err != nil {
			if prewarmErr == nil {
				prewarmErr = fmt.Errorf("failed waiting for prewarm receipt for account %s (tx %s): %w", res.account, res.txHash, res.err)
			}
			cancel()
			return
		}
		processedAccounts++
		if processedAccounts%logInterval == 0 {
			log.Printf("🔥 Prewarming progress: %d accounts processed...", processedAccounts)
		}
	}

	for {
		if generatorDone && inFlight == 0 {
			break
		}
		if ctx.Err() != nil && inFlight == 0 {
			break
		}

		if inFlight > 0 {
			select {
			case res := <-results:
				handleResult(res)
				continue
			default:
			}
		}

		if generatorDone || ctx.Err() != nil {
			if inFlight > 0 {
				res := <-results
				handleResult(res)
				continue
			}
			break
		}

		if maxInFlight > 0 && inFlight >= maxInFlight {
			res := <-results
			handleResult(res)
			continue
		}

		tx, ok := gen.Generate()
		if !ok {
			generatorDone = true
			continue
		}

		if limiter != nil {
			if err := limiter.Wait(ctx); err != nil {
				if prewarmErr == nil {
					prewarmErr = fmt.Errorf("prewarm rate limiter wait failed: %w", err)
				}
				cancel()
				generatorDone = true
				continue
			}
		}

		if err := d.sender.Send(ctx, tx); err != nil {
			log.Printf("🔥 Failed to send prewarm transaction for account %s: %v", tx.Scenario.Sender.Address.Hex(), err)
			continue
		}

		inFlight++
		account := tx.Scenario.Sender.Address.Hex()
		txHash := tx.EthTx.Hash().Hex()

		go func(tx *types.LoadTx, account string, txHash string) {
			err := waitForReceipt(ctx, client, tx)
			results <- prewarmResult{account: account, txHash: txHash, err: err}
		}(tx, account, txHash)
	}

	if prewarmErr != nil {
		return prewarmErr
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
