package stats

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sei-protocol/sei-load/utils"
)

// Logger handles periodic statistics logging and dry-run transaction printing
type Logger struct {
	collector *Collector
	interval  time.Duration
	debug     bool
}

// NewLogger creates a new statistics logger
func NewLogger(collector *Collector, interval time.Duration, debug bool) *Logger {
	return &Logger{
		collector: collector,
		interval:  interval,
		debug:     debug,
	}
}

// Start begins periodic statistics logging
func (l *Logger) Run(ctx context.Context) error {
	ticker := time.NewTicker(l.interval)
	defer ticker.Stop()
	for ctx.Err() == nil {
		if _, err := utils.Recv(ctx, ticker.C); err != nil {
			return err
		}
		l.logCurrentStats()
	}
	return ctx.Err()
}

// logCurrentStats logs the current statistics
func (l *Logger) logCurrentStats() {
	stats := l.collector.GetStats()

	// Aggregate metrics for overall summary
	var totalWindowTxs uint64
	var totalTxs uint64
	var totalWindowTPS float64
	var totalCumulativeMaxTPS float64
	var weightedLatencySum time.Duration
	var totalLatencyCount int
	var maxCumulativeLatency time.Duration
	var maxP50, maxP99 time.Duration

	// Log one line per endpoint with concise metrics
	for endpoint, endpointStats := range stats.EndpointStats {
		// Calculate window TPS based on actual window duration
		var windowTPS float64
		if endpointStats.WindowTxCount > 0 {
			// Use the logging interval as the window duration
			windowDuration := l.interval.Seconds()
			windowTPS = float64(endpointStats.WindowTxCount) / windowDuration
		}

		// Calculate window average latency
		var windowAvgLatency time.Duration
		if endpointStats.WindowLatencyCount > 0 {
			windowAvgLatency = endpointStats.WindowLatencySum / time.Duration(endpointStats.WindowLatencyCount)
		}

		// Get total transactions for this endpoint
		totalTxsForEndpoint := uint64(0)
		for _, endpoints := range stats.TxCounts {
			if count, exists := endpoints[endpoint]; exists {
				totalTxsForEndpoint += count
			}
		}

		// Aggregate for overall summary
		totalWindowTxs += endpointStats.WindowTxCount
		totalTxs += totalTxsForEndpoint
		totalWindowTPS += windowTPS
		totalCumulativeMaxTPS += endpointStats.CumulativeMaxTPS
		weightedLatencySum += endpointStats.WindowLatencySum
		totalLatencyCount += endpointStats.WindowLatencyCount
		if endpointStats.CumulativeMaxLatency > maxCumulativeLatency {
			maxCumulativeLatency = endpointStats.CumulativeMaxLatency
		}
		if endpointStats.P50Latency > maxP50 {
			maxP50 = endpointStats.P50Latency
		}
		if endpointStats.P99Latency > maxP99 {
			maxP99 = endpointStats.P99Latency
		}

		if l.debug {
			// Format: [timestamp] endpoint | TXs: total | TPS: window(max) | Latency: avg(max) | P50: x P99: x
			log.Printf("[%s] %s | TXs: %d | TPS: %.1f(%.1f) | Lat: %v(%v) | P50: %v P99: %v",
				time.Now().Format("15:04:05"),
				endpoint,
				totalTxsForEndpoint,
				windowTPS,
				endpointStats.CumulativeMaxTPS,
				windowAvgLatency.Round(time.Millisecond),
				endpointStats.CumulativeMaxLatency.Round(time.Millisecond),
				endpointStats.P50Latency.Round(time.Millisecond),
				endpointStats.P99Latency.Round(time.Millisecond))
		}
	}

	// Calculate overall average latency
	var overallAvgLatency time.Duration
	if totalLatencyCount > 0 {
		overallAvgLatency = weightedLatencySum / time.Duration(totalLatencyCount)
	}

	// Print overall summary line
	log.Printf("throughput tps=%.2f, txs=%d,  latency(avg=%v p50=%v p99=%v max=%v)",
		totalWindowTPS,
		totalTxs,
		overallAvgLatency.Round(time.Millisecond),
		maxP50.Round(time.Millisecond),
		maxP99.Round(time.Millisecond),
		maxCumulativeLatency.Round(time.Millisecond))

	// Print block statistics if available
	if stats.BlockStats != nil && stats.BlockStats.SampleCount > 0 {
		log.Printf("%s",
			stats.BlockStats.FormatBlockStats())
	}

	// Reset window stats for next period
	l.collector.ResetWindowStats()

	// Reset block collector window stats
	if blockCollector := l.collector.GetBlockCollector(); blockCollector != nil {
		blockCollector.ResetWindowStats()
	}
}

// LogFinalStats logs comprehensive final statistics
func (l *Logger) LogFinalStats() {
	stats := l.collector.GetStats()

	// Use fmt.Print for clean output without timestamps
	fmt.Println()
	fmt.Println("=============================")
	fmt.Println("FINAL LOAD TEST RESULTS")
	fmt.Println("=============================")
	fmt.Println()

	// Print load test statistics
	duration := time.Since(stats.StartTime)
	avgTPS := float64(stats.TotalTxs) / duration.Seconds()

	fmt.Println("=== Load Test Statistics ===")
	fmt.Printf("Runtime: %v | Total TXs: %d | Avg TPS: %.2f\n\n",
		duration.Round(time.Second), stats.TotalTxs, avgTPS)

	// Transaction counts by scenario
	fmt.Println("Transaction Counts by Scenario:")
	for scenario, endpoints := range stats.TxCounts {
		fmt.Printf("  %s:\n", scenario)
		for endpoint, count := range endpoints {
			fmt.Printf("    %s: %d\n", endpoint, count)
		}
	}

	// Endpoint performance
	fmt.Println("\nEndpoint Performance:")
	for endpoint, endpointStats := range stats.EndpointStats {
		fmt.Printf("  %s:\n", endpoint)
		fmt.Printf("    Latency P50: %v | P99: %v (samples: %d)\n",
			endpointStats.P50Latency.Round(time.Millisecond),
			endpointStats.P99Latency.Round(time.Millisecond),
			endpointStats.SampleCount)
		fmt.Printf("    TPS Current: %.2f | Max (10s): %.2f\n",
			endpointStats.CurrentTPS, endpointStats.MaxTPS)
		fmt.Printf("    Window TXs: %d | Latency Sum: %v | Latency Count: %d\n",
			endpointStats.WindowTxCount,
			endpointStats.WindowLatencySum.Round(time.Millisecond),
			endpointStats.WindowLatencyCount)
		fmt.Printf("    Window Max Latency: %v | Window Min Latency: %v\n",
			endpointStats.WindowMaxLatency.Round(time.Millisecond),
			endpointStats.WindowMinLatency.Round(time.Millisecond))
		fmt.Printf("    Cumulative Max TPS: %.2f | Cumulative Max Latency: %v\n",
			endpointStats.CumulativeMaxTPS,
			endpointStats.CumulativeMaxLatency.Round(time.Millisecond))
	}

	// Overall TPS
	fmt.Printf("\nOverall TPS: Current: %.2f | Max (10s): %.2f\n",
		stats.OverallCurrentTPS, stats.OverallMaxTPS)

	// Block stats
	if stats.BlockStats != nil && stats.BlockStats.SampleCount > 0 {
		fmt.Printf("\nBlock Statistics:\n")
		fmt.Printf("  Height: %d | Samples: %d\n",
			stats.BlockStats.MaxBlockNumber, stats.BlockStats.SampleCount)
		fmt.Printf("  Block Times: P50=%v | P99=%v | Max=%v\n",
			stats.BlockStats.P50BlockTime.Round(time.Millisecond),
			stats.BlockStats.P99BlockTime.Round(time.Millisecond),
			stats.BlockStats.MaxBlockTime.Round(time.Millisecond))
		fmt.Printf("  Gas Usage: P50=%d | P99=%d | Max=%d\n",
			stats.BlockStats.P50GasUsed,
			stats.BlockStats.P99GasUsed,
			stats.BlockStats.MaxGasUsed)
	}

	// Additional final statistics
	if duration.Seconds() > 0 {
		fmt.Println("\nOverall Performance Summary:")
		fmt.Printf("  Total Runtime: %v\n", duration.Round(time.Second))
		fmt.Printf("  Total Transactions: %d\n", stats.TotalTxs)
		fmt.Printf("  Average TPS: %.2f\n", float64(stats.TotalTxs)/duration.Seconds())
		fmt.Printf("  Max TPS: %.2f\n", stats.OverallMaxTPS)

		// Calculate total transactions per scenario
		scenarioTotals := make(map[string]uint64)
		for scenario, endpoints := range stats.TxCounts {
			total := uint64(0)
			for _, count := range endpoints {
				total += count
			}
			scenarioTotals[scenario] = total
		}

		fmt.Println("\nScenario Distribution:")
		for scenario, total := range scenarioTotals {
			percentage := float64(total) / float64(stats.TotalTxs) * 100
			fmt.Printf("  %s: %d (%.1f%%)\n", scenario, total, percentage)
		}
	}

	// Print overall gas statistics if available (use cumulative data)
	if cumulativeBlockStats := l.collector.GetCumulativeBlockStats(); cumulativeBlockStats != nil && cumulativeBlockStats.SampleCount > 0 {
		fmt.Println("\nOverall Gas Statistics:")
		fmt.Printf("  Max Block Number: %d\n", cumulativeBlockStats.MaxBlockNumber)
		fmt.Printf("  Block Times: p50=%v p99=%v max=%v\n",
			cumulativeBlockStats.P50BlockTime.Round(time.Millisecond),
			cumulativeBlockStats.P99BlockTime.Round(time.Millisecond),
			cumulativeBlockStats.MaxBlockTime.Round(time.Millisecond))
		fmt.Printf("  Gas Usage: p50=%d p99=%d max=%d\n",
			cumulativeBlockStats.P50GasUsed,
			cumulativeBlockStats.P99GasUsed,
			cumulativeBlockStats.MaxGasUsed)
		fmt.Printf("  Block Samples: %d\n", cumulativeBlockStats.SampleCount)
	}

	fmt.Println("==============================")
}
