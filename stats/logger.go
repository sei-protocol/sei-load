package stats

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/sei-protocol/sei-load/utils"
)

// FinalStats represents the complete final statistics that can be marshaled to JSON
type FinalStats struct {
	LoadTestStatistics   LoadTestStatistics `json:"load_test_statistics"`
	ScenarioDistribution map[string]uint64  `json:"scenario_distribution"`
	TransactionStats     PerformanceStats   `json:"transaction_stats"`
	OverallTPS           OverallTPS         `json:"overall_tps"`
	BlockStatistics      *BlockStats        `json:"block_statistics,omitempty"`
	OverallPerformance   OverallPerformance `json:"overall_performance"`
	GasStatistics        *BlockStats        `json:"gas_statistics,omitempty"`
}

// LoadTestStatistics represents basic load test metrics
type LoadTestStatistics struct {
	Runtime   time.Duration `json:"runtime"`
	TotalTxs  uint64        `json:"total_txs"`
	AvgTPS    float64       `json:"avg_tps"`
	StartTime time.Time     `json:"start_time"`
}

// PerformanceStats represents detailed transaction performance metrics.
type PerformanceStats struct {
	LatencyP50           time.Duration `json:"latency_p50"`
	LatencyP99           time.Duration `json:"latency_p99"`
	SampleCount          int           `json:"sample_count"`
	CurrentTPS           float64       `json:"current_tps"`
	MaxTPS               float64       `json:"max_tps"`
	WindowTxCount        uint64        `json:"window_tx_count"`
	WindowLatencySum     time.Duration `json:"window_latency_sum"`
	WindowLatencyCount   int           `json:"window_latency_count"`
	WindowMaxLatency     time.Duration `json:"window_max_latency"`
	WindowMinLatency     time.Duration `json:"window_min_latency"`
	CumulativeMaxTPS     float64       `json:"cumulative_max_tps"`
	CumulativeMaxLatency time.Duration `json:"cumulative_max_latency"`
}

// OverallTPS represents overall throughput metrics
type OverallTPS struct {
	Current float64 `json:"current"`
	Max     float64 `json:"max"`
}

// OverallPerformance represents comprehensive performance summary
type OverallPerformance struct {
	TotalRuntime      time.Duration `json:"total_runtime"`
	TotalTransactions uint64        `json:"total_transactions"`
	AverageTPS        float64       `json:"average_tps"`
	MaxTPS            float64       `json:"max_tps"`
}

// String returns a formatted string representation of the final statistics
func (fs *FinalStats) String() string {
	var result string

	result += "\n=============================\n"
	result += "FINAL LOAD TEST RESULTS\n"
	result += "=============================\n\n"

	// Load test statistics
	result += "=== Load Test Statistics ===\n"
	result += fmt.Sprintf("Runtime: %v | Total TXs: %d | Avg TPS: %.2f\n\n",
		fs.LoadTestStatistics.Runtime.Round(time.Second),
		fs.LoadTestStatistics.TotalTxs,
		fs.LoadTestStatistics.AvgTPS)

	// Transaction counts by scenario
	result += "Transaction Counts by Scenario:\n"
	for scenario, total := range fs.ScenarioDistribution {
		result += fmt.Sprintf("  %s: %d\n", scenario, total)
	}

	// Transaction performance
	result += "\nTransaction Performance:\n"
	result += fmt.Sprintf("  Latency P50: %v | P99: %v (samples: %d)\n",
		fs.TransactionStats.LatencyP50.Round(time.Millisecond),
		fs.TransactionStats.LatencyP99.Round(time.Millisecond),
		fs.TransactionStats.SampleCount)
	result += fmt.Sprintf("  TPS Current: %.2f | Max (10s): %.2f\n",
		fs.TransactionStats.CurrentTPS, fs.TransactionStats.MaxTPS)
	result += fmt.Sprintf("  Window TXs: %d | Latency Sum: %v | Latency Count: %d\n",
		fs.TransactionStats.WindowTxCount,
		fs.TransactionStats.WindowLatencySum.Round(time.Millisecond),
		fs.TransactionStats.WindowLatencyCount)
	result += fmt.Sprintf("  Window Max Latency: %v | Window Min Latency: %v\n",
		fs.TransactionStats.WindowMaxLatency.Round(time.Millisecond),
		fs.TransactionStats.WindowMinLatency.Round(time.Millisecond))
	result += fmt.Sprintf("  Cumulative Max TPS: %.2f | Cumulative Max Latency: %v\n",
		fs.TransactionStats.CumulativeMaxTPS,
		fs.TransactionStats.CumulativeMaxLatency.Round(time.Millisecond))

	// Overall TPS
	result += fmt.Sprintf("\nOverall TPS: Current: %.2f | Max (10s): %.2f\n",
		fs.OverallTPS.Current, fs.OverallTPS.Max)

	// Block stats
	if fs.BlockStatistics != nil && fs.BlockStatistics.SampleCount > 0 {
		result += "\nBlock Statistics:\n"
		result += fmt.Sprintf("  Height: %d | Samples: %d\n",
			fs.BlockStatistics.MaxBlockNumber, fs.BlockStatistics.SampleCount)
		result += fmt.Sprintf("  Block Times: P50=%v | P99=%v | Max=%v\n",
			fs.BlockStatistics.P50BlockTime.Round(time.Millisecond),
			fs.BlockStatistics.P99BlockTime.Round(time.Millisecond),
			fs.BlockStatistics.MaxBlockTime.Round(time.Millisecond))
		result += fmt.Sprintf("  Gas Usage: P50=%d | P99=%d | Max=%d\n",
			fs.BlockStatistics.P50GasUsed,
			fs.BlockStatistics.P99GasUsed,
			fs.BlockStatistics.MaxGasUsed)
	}

	// Overall performance summary
	result += "\nOverall Performance Summary:\n"
	result += fmt.Sprintf("  Total Runtime: %v\n", fs.OverallPerformance.TotalRuntime.Round(time.Second))
	result += fmt.Sprintf("  Total Transactions: %d\n", fs.OverallPerformance.TotalTransactions)
	result += fmt.Sprintf("  Average TPS: %.2f\n", fs.OverallPerformance.AverageTPS)
	result += fmt.Sprintf("  Max TPS: %.2f\n", fs.OverallPerformance.MaxTPS)

	// Scenario distribution
	result += "\nScenario Distribution:\n"
	for scenario, total := range fs.ScenarioDistribution {
		percentage := float64(total) / float64(fs.LoadTestStatistics.TotalTxs) * 100
		result += fmt.Sprintf("  %s: %d (%.1f%%)\n", scenario, total, percentage)
	}

	// Gas statistics
	if fs.GasStatistics != nil && fs.GasStatistics.SampleCount > 0 {
		result += "\nOverall Gas Statistics:\n"
		result += fmt.Sprintf("  Max Block Number: %d\n", fs.GasStatistics.MaxBlockNumber)
		result += fmt.Sprintf("  Block Times: p50=%v p99=%v max=%v\n",
			fs.GasStatistics.P50BlockTime.Round(time.Millisecond),
			fs.GasStatistics.P99BlockTime.Round(time.Millisecond),
			fs.GasStatistics.MaxBlockTime.Round(time.Millisecond))
		result += fmt.Sprintf("  Gas Usage: p50=%d p99=%d max=%d\n",
			fs.GasStatistics.P50GasUsed,
			fs.GasStatistics.P99GasUsed,
			fs.GasStatistics.MaxGasUsed)
		result += fmt.Sprintf("  Block Samples: %d\n", fs.GasStatistics.SampleCount)
	}

	result += "==============================\n"
	return result
}

// BuildFinalStats creates a FinalStats struct from the current collector data
func (l *Logger) BuildFinalStats() *FinalStats {
	stats := l.collector.GetStats()
	duration := time.Since(stats.StartTime)
	avgTPS := float64(stats.TotalTxs) / duration.Seconds()

	// Build scenario distribution (aggregate by scenario)
	scenarioDistribution := make(map[string]uint64)
	for scenario, count := range stats.TxCounts {
		scenarioDistribution[scenario] = count
	}

	transactionStats := PerformanceStats{
		LatencyP50:           stats.TransactionStats.P50Latency,
		LatencyP99:           stats.TransactionStats.P99Latency,
		SampleCount:          stats.TransactionStats.SampleCount,
		CurrentTPS:           stats.TransactionStats.CurrentTPS,
		MaxTPS:               stats.TransactionStats.MaxTPS,
		WindowTxCount:        stats.TransactionStats.WindowTxCount,
		WindowLatencySum:     stats.TransactionStats.WindowLatencySum,
		WindowLatencyCount:   stats.TransactionStats.WindowLatencyCount,
		WindowMaxLatency:     stats.TransactionStats.WindowMaxLatency,
		WindowMinLatency:     stats.TransactionStats.WindowMinLatency,
		CumulativeMaxTPS:     stats.TransactionStats.CumulativeMaxTPS,
		CumulativeMaxLatency: stats.TransactionStats.CumulativeMaxLatency,
	}

	// Build overall TPS
	overallTPS := OverallTPS{
		Current: stats.OverallCurrentTPS,
		Max:     stats.OverallMaxTPS,
	}

	// Build overall performance
	overallPerformance := OverallPerformance{
		TotalRuntime:      duration,
		TotalTransactions: stats.TotalTxs,
		AverageTPS:        avgTPS,
		MaxTPS:            stats.OverallMaxTPS,
	}

	// Build load test statistics
	loadTestStats := LoadTestStatistics{
		Runtime:   duration,
		TotalTxs:  stats.TotalTxs,
		AvgTPS:    avgTPS,
		StartTime: stats.StartTime,
	}

	return &FinalStats{
		LoadTestStatistics:   loadTestStats,
		ScenarioDistribution: scenarioDistribution,
		TransactionStats:     transactionStats,
		OverallTPS:           overallTPS,
		BlockStatistics:      stats.BlockStats,
		OverallPerformance:   overallPerformance,
		GasStatistics:        l.collector.GetCumulativeBlockStats(),
	}
}

// Logger handles periodic statistics logging and dry-run transaction printing
type Logger struct {
	collector  *Collector
	interval   time.Duration
	debug      bool
	reportPath string
}

// NewLogger creates a new statistics logger
func NewLogger(collector *Collector, interval time.Duration, reportPath string, debug bool) *Logger {
	return &Logger{
		collector:  collector,
		interval:   interval,
		reportPath: reportPath,
		debug:      debug,
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

	txStats := stats.TransactionStats
	windowTPS := 0.0
	if txStats.WindowTxCount > 0 {
		windowTPS = float64(txStats.WindowTxCount) / l.interval.Seconds()
	}
	var overallAvgLatency time.Duration
	if txStats.WindowLatencyCount > 0 {
		overallAvgLatency = txStats.WindowLatencySum / time.Duration(txStats.WindowLatencyCount)
	}

	if l.debug {
		log.Printf("[%s] TXs: %d | TPS: %.1f(%.1f) | Lat: %v(%v) | P50: %v P99: %v",
			time.Now().Format("15:04:05"),
			stats.TotalTxs,
			windowTPS,
			txStats.CumulativeMaxTPS,
			overallAvgLatency.Round(time.Millisecond),
			txStats.CumulativeMaxLatency.Round(time.Millisecond),
			txStats.P50Latency.Round(time.Millisecond),
			txStats.P99Latency.Round(time.Millisecond))
	}

	// Print overall summary line
	log.Printf("throughput tps=%.2f, txs=%d,  latency(avg=%v p50=%v p99=%v max=%v)",
		windowTPS,
		stats.TotalTxs,
		overallAvgLatency.Round(time.Millisecond),
		txStats.P50Latency.Round(time.Millisecond),
		txStats.P99Latency.Round(time.Millisecond),
		txStats.CumulativeMaxLatency.Round(time.Millisecond))

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
	finalStats := l.BuildFinalStats()
	fmt.Print(finalStats.String())

	if l.reportPath != "" {
		// just write the string to file
		reportFile, err := os.Create(l.reportPath)
		if err != nil {
			log.Printf("Error creating report file: %v", err)
			return
		}
		defer func() { _ = reportFile.Close() }()
		_, err = reportFile.WriteString(finalStats.String())
		if err != nil {
			log.Printf("Error writing report file: %v", err)
			return
		}
	}
}
