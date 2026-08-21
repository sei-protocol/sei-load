package stats

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Collector tracks comprehensive statistics for load testing
type Collector struct {
	mu sync.RWMutex

	// Transaction counts by scenario
	txCounts map[string]uint64

	// Per-operation counts and latencies. A run whose scenarios draw from a
	// weighted basket reports one blended percentile without this, which cannot
	// say which operation degraded.
	perOperation map[OperationKey]*operationSamples

	// Latency tracking, pooled across every scenario and operation. Kept as the
	// run's headline percentile; perOperation splits it.
	latencies []time.Duration

	// TPS tracking with 10-second windows
	tpsWindow *TPSWindow

	// Window-based tracking for periodic reporting
	windowStats *WindowStats

	// Block data collector
	blockCollector *BlockCollector

	// Global metrics
	startTime      time.Time
	totalTxs       uint64
	lastWindowTime time.Time

	// Configuration
	maxLatencyHistory int // Limit latency history to prevent memory leaks
}

// TPSWindow tracks transactions in a sliding 10-second window
type TPSWindow struct {
	timestamps []time.Time
	maxTPS     float64
	mu         sync.RWMutex
}

// NewCollector creates a new statistics collector
func NewCollector() *Collector {
	return &Collector{
		txCounts:          make(map[string]uint64),
		perOperation:      make(map[OperationKey]*operationSamples),
		latencies:         make([]time.Duration, 0),
		tpsWindow:         &TPSWindow{timestamps: make([]time.Time, 0)},
		windowStats:       &WindowStats{windowStart: time.Now()},
		startTime:         time.Now(),
		lastWindowTime:    time.Now(),
		maxLatencyHistory: 10000, // Keep last 10k latencies
	}
}

// RecordTransaction records a transaction attempt. The operation names the call
// shape the scenario issued; see types.TxScenario.
func (c *Collector) RecordTransaction(scenario, operation string, latency time.Duration, success bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Record transaction count
	c.txCounts[scenario]++
	c.totalTxs++

	key := OperationKey{Scenario: scenario, Operation: operation}
	samples := c.perOperation[key]
	if samples == nil {
		samples = &operationSamples{}
		c.perOperation[key] = samples
	}
	samples.count++

	// Record latency (only for successful transactions)
	if success {
		c.recordLatency(latency)
		samples.record(latency, c.maxLatencyHistory)
	}

	// Record TPS
	c.recordTPS()

	// Record window stats
	c.recordWindowStats(latency)
}

// recordLatency adds a latency measurement, maintaining history limit
func (c *Collector) recordLatency(latency time.Duration) {
	// Add new latency
	c.latencies = append(c.latencies, latency)

	// Trim if over limit (keep most recent)
	if len(c.latencies) > c.maxLatencyHistory {
		c.latencies = c.latencies[len(c.latencies)-c.maxLatencyHistory:]
	}
}

// recordTPS updates the TPS window
func (c *Collector) recordTPS() {
	window := c.tpsWindow
	window.mu.Lock()
	defer window.mu.Unlock()

	now := time.Now()
	window.timestamps = append(window.timestamps, now)

	// Remove timestamps older than 10 seconds
	cutoff := now.Add(-10 * time.Second)
	validIndex := 0
	for i, ts := range window.timestamps {
		if ts.After(cutoff) {
			validIndex = i
			break
		}
	}
	window.timestamps = window.timestamps[validIndex:]

	// Calculate current TPS and update max
	currentTPS := float64(len(window.timestamps)) / 10.0
	if currentTPS > window.maxTPS {
		window.maxTPS = currentTPS
	}
}

// recordWindowStats updates the window stats
func (c *Collector) recordWindowStats(latency time.Duration) {
	windowStats := c.windowStats

	// Update tx count
	windowStats.txCount++

	// Update latency sum and count
	windowStats.latencySum += latency
	windowStats.latencyCount++

	// Update max and min latency
	if latency > windowStats.maxLatency {
		windowStats.maxLatency = latency
	}
	if latency < windowStats.minLatency || windowStats.minLatency == 0 {
		windowStats.minLatency = latency
	}

	// Update cumulative max TPS and latency
	if windowStats.txCount > 0 {
		currentTPS := float64(windowStats.txCount) / time.Since(windowStats.windowStart).Seconds()
		if currentTPS > windowStats.cumulativeMaxTPS {
			windowStats.cumulativeMaxTPS = currentTPS
		}
	}
	if latency > windowStats.cumulativeMaxLatency {
		windowStats.cumulativeMaxLatency = latency
	}
}

// ResetWindowStats resets the window statistics
func (c *Collector) ResetWindowStats() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	// Preserve cumulative maximums
	cumulativeMaxTPS := c.windowStats.cumulativeMaxTPS
	cumulativeMaxLatency := c.windowStats.cumulativeMaxLatency

	// Reset window stats
	c.windowStats = &WindowStats{
		windowStart:          now,
		cumulativeMaxTPS:     cumulativeMaxTPS,
		cumulativeMaxLatency: cumulativeMaxLatency,
	}
	c.lastWindowTime = now
}

// GetStats returns comprehensive statistics
func (c *Collector) GetStats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := Stats{
		StartTime:  c.startTime,
		TotalTxs:   c.totalTxs,
		TxCounts:   make(map[string]uint64),
		Operations: make(map[OperationKey]OperationStats, len(c.perOperation)),
	}

	// Copy transaction counts
	for scenario, count := range c.txCounts {
		stats.TxCounts[scenario] = count
	}

	for key, samples := range c.perOperation {
		op := OperationStats{Count: samples.count, SampleCount: len(samples.latencies)}
		if len(samples.latencies) > 0 {
			sorted := slices.Clone(samples.latencies)
			slices.Sort(sorted)
			op.P50Latency = calculatePercentile(sorted, 50)
			op.P99Latency = calculatePercentile(sorted, 99)
		}
		stats.Operations[key] = op
	}

	// Calculate transaction statistics
	transactionStats := TransactionStats{}
	if len(c.latencies) > 0 {
		sortedLatencies := make([]time.Duration, len(c.latencies))
		copy(sortedLatencies, c.latencies)
		sort.Slice(sortedLatencies, func(i, j int) bool {
			return sortedLatencies[i] < sortedLatencies[j]
		})

		transactionStats.P50Latency = calculatePercentile(sortedLatencies, 50)
		transactionStats.P99Latency = calculatePercentile(sortedLatencies, 99)
		transactionStats.SampleCount = len(sortedLatencies)
	}

	c.tpsWindow.mu.RLock()
	transactionStats.MaxTPS = c.tpsWindow.maxTPS
	now := time.Now()
	cutoff := now.Add(-10 * time.Second)
	currentCount := 0
	for _, ts := range c.tpsWindow.timestamps {
		if ts.After(cutoff) {
			currentCount++
		}
	}
	transactionStats.CurrentTPS = float64(currentCount) / 10.0
	c.tpsWindow.mu.RUnlock()

	transactionStats.WindowTxCount = c.windowStats.txCount
	transactionStats.WindowLatencySum = c.windowStats.latencySum
	transactionStats.WindowLatencyCount = c.windowStats.latencyCount
	transactionStats.WindowMaxLatency = c.windowStats.maxLatency
	transactionStats.WindowMinLatency = c.windowStats.minLatency
	transactionStats.CumulativeMaxTPS = c.windowStats.cumulativeMaxTPS
	transactionStats.CumulativeMaxLatency = c.windowStats.cumulativeMaxLatency
	stats.TransactionStats = transactionStats
	stats.OverallMaxTPS = transactionStats.MaxTPS
	stats.OverallCurrentTPS = transactionStats.CurrentTPS

	// Get block stats
	if c.blockCollector != nil {
		blockStats := c.blockCollector.GetWindowBlockStats()
		stats.BlockStats = &blockStats
	}

	return stats
}

// GetCumulativeBlockStats returns cumulative block stats for final summary
func (c *Collector) GetCumulativeBlockStats() *BlockStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.blockCollector != nil {
		stats := c.blockCollector.GetBlockStats()
		return &stats
	}
	return nil
}

// calculatePercentile calculates the given percentile from sorted durations
func calculatePercentile(sorted []time.Duration, percentile int) time.Duration {
	if len(sorted) == 0 {
		return 0
	}

	index := (len(sorted) * percentile) / 100
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	return sorted[index]
}

// Stats represents comprehensive load test statistics
type Stats struct {
	StartTime         time.Time
	TotalTxs          uint64
	TxCounts          map[string]uint64 // [scenario] -> count
	TransactionStats  TransactionStats
	OverallMaxTPS     float64
	OverallCurrentTPS float64
	BlockStats        *BlockStats                     // Block-related statistics
	Operations        map[OperationKey]OperationStats // [scenario, operation] -> stats
}

// OperationKey identifies one call shape within one scenario. Two scenarios that
// issue the same call share an operation name, so neither field identifies a
// series on its own.
type OperationKey struct {
	Scenario  string
	Operation string
}

// OperationStats is one operation's share of a run.
type OperationStats struct {
	Count       uint64
	P50Latency  time.Duration
	P99Latency  time.Duration
	SampleCount int
}

// operationSamples accumulates one operation's counts and latencies. Each
// operation keeps its own bounded window, so a low-rate operation stays
// represented instead of being evicted by a high-rate one.
type operationSamples struct {
	count     uint64
	latencies []time.Duration
}

func (o *operationSamples) record(latency time.Duration, limit int) {
	o.latencies = append(o.latencies, latency)
	if len(o.latencies) > limit {
		o.latencies = o.latencies[len(o.latencies)-limit:]
	}
}

// TransactionStats represents transaction performance statistics.
type TransactionStats struct {
	P50Latency  time.Duration
	P99Latency  time.Duration
	MaxTPS      float64
	CurrentTPS  float64
	SampleCount int
	QueueDepth  int // Current queue depth for monitoring backpressure

	// Window stats
	WindowTxCount        uint64
	WindowLatencySum     time.Duration
	WindowLatencyCount   int
	WindowMaxLatency     time.Duration
	WindowMinLatency     time.Duration
	CumulativeMaxTPS     float64
	CumulativeMaxLatency time.Duration
}

// WindowStats tracks metrics for the current reporting window
type WindowStats struct {
	windowStart          time.Time
	txCount              uint64
	latencySum           time.Duration
	latencyCount         int
	maxLatency           time.Duration
	minLatency           time.Duration
	cumulativeMaxTPS     float64
	cumulativeMaxLatency time.Duration
}

// SetBlockCollector sets the block collector for this stats collector
func (c *Collector) SetBlockCollector(bc *BlockCollector) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.blockCollector = bc
}

// GetBlockCollector returns the block collector
func (c *Collector) GetBlockCollector() *BlockCollector {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.blockCollector
}

// FormatStats returns a formatted string representation of the statistics
func (s *Stats) FormatStats() string {
	duration := time.Since(s.StartTime)
	avgTPS := float64(s.TotalTxs) / duration.Seconds()

	result := "\n=== Load Test Statistics ===\n"
	result += fmt.Sprintf("Runtime: %v | Total TXs: %d | Avg TPS: %.2f\n\n",
		duration.Round(time.Second), s.TotalTxs, avgTPS)

	// Transaction counts by scenario
	result += "Transaction Counts by Scenario:\n"
	for scenario, count := range s.TxCounts {
		result += fmt.Sprintf("  %s: %d\n", scenario, count)
	}

	if len(s.Operations) > 0 {
		keys := slices.SortedFunc(maps.Keys(s.Operations), func(a, b OperationKey) int {
			if a.Scenario != b.Scenario {
				return strings.Compare(a.Scenario, b.Scenario)
			}
			return strings.Compare(a.Operation, b.Operation)
		})
		result += "\nPer Operation:\n"
		for _, key := range keys {
			op := s.Operations[key]
			result += fmt.Sprintf("  %s/%s: %d txs | P50: %v | P99: %v (samples: %d)\n",
				key.Scenario, key.Operation, op.Count,
				op.P50Latency.Round(time.Millisecond),
				op.P99Latency.Round(time.Millisecond),
				op.SampleCount)
		}
	}

	result += "\nTransaction Performance:\n"
	result += "  (pooled across every scenario and operation)\n"
	result += fmt.Sprintf("  Latency P50: %v | P99: %v (samples: %d)\n",
		s.TransactionStats.P50Latency.Round(time.Millisecond),
		s.TransactionStats.P99Latency.Round(time.Millisecond),
		s.TransactionStats.SampleCount)
	result += fmt.Sprintf("  TPS Current: %.2f | Max (10s): %.2f\n",
		s.TransactionStats.CurrentTPS, s.TransactionStats.MaxTPS)
	result += fmt.Sprintf("  Window TXs: %d | Latency Sum: %v | Latency Count: %d\n",
		s.TransactionStats.WindowTxCount,
		s.TransactionStats.WindowLatencySum.Round(time.Millisecond),
		s.TransactionStats.WindowLatencyCount)
	result += fmt.Sprintf("  Window Max Latency: %v | Window Min Latency: %v\n",
		s.TransactionStats.WindowMaxLatency.Round(time.Millisecond),
		s.TransactionStats.WindowMinLatency.Round(time.Millisecond))
	result += fmt.Sprintf("  Cumulative Max TPS: %.2f | Cumulative Max Latency: %v\n",
		s.TransactionStats.CumulativeMaxTPS,
		s.TransactionStats.CumulativeMaxLatency.Round(time.Millisecond))

	// Overall TPS
	result += fmt.Sprintf("\nOverall TPS: Current: %.2f | Max (10s): %.2f\n",
		s.OverallCurrentTPS, s.OverallMaxTPS)

	// Block stats
	if s.BlockStats != nil {
		result += "\n" + s.BlockStats.FormatBlockStats()
	}

	return result
}
