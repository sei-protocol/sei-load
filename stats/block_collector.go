package stats

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sei-protocol/sei-load/utils"
	"github.com/sei-protocol/sei-load/utils/service"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// pendingBlock holds a block header and the time it was received for async processing
type pendingBlock struct {
	header       *types.Header
	receivedTime time.Time
}

type blockCollectorStats struct {
	// Cumulative data (for final stats)
	allBlockTimes []time.Duration // All block times
	allGasUsed    []uint64        // All gas used values
	allTxCounts   []int           // Transaction counts per block
	allTPS        []float64       // TPS per block (txCount / blockTime)
	totalTxCount  uint64          // Total transactions across all blocks
	maxBlockNum   uint64          // Highest block number seen
	lastBlockTime time.Time       // Timestamp of last block

	// Window-based data (for periodic reporting)
	windowBlockTimes []time.Duration // Block times in current window
	windowGasUsed    []uint64        // Gas used in current window
	windowTxCounts   []int           // Transaction counts in current window
	windowTPS        []float64       // TPS values in current window
	windowTotalTxs   uint64          // Total txs in current window
	windowStart      time.Time       // Start of current window
}

// BlockCollector subscribes to new blocks and tracks block metrics
type BlockCollector struct {
	seiChainID string
	stats      utils.Mutex[*blockCollectorStats]
}

type BlockStatsProvider interface {
	GetBlockStats() BlockStats
	GetWindowBlockStats() BlockStats
	GetWindowBlockTimePercentile(percentile int) time.Duration
	ResetWindowStats()
}

// NewBlockCollector creates a new block data collector
func NewBlockCollector(seiChainID string) *BlockCollector {
	return &BlockCollector{
		seiChainID: seiChainID,
		stats: utils.NewMutex(&blockCollectorStats{
			allBlockTimes:    make([]time.Duration, 0),
			allGasUsed:       make([]uint64, 0),
			allTxCounts:      make([]int, 0),
			allTPS:           make([]float64, 0),
			windowBlockTimes: make([]time.Duration, 0),
			windowGasUsed:    make([]uint64, 0),
			windowTxCounts:   make([]int, 0),
			windowTPS:        make([]float64, 0),
		}),
	}
}

// Start begins block subscription and data collection
func (bc *BlockCollector) Run(ctx context.Context, firstEndpoint string) error {
	wsEndpoint := utils.GetWSEndpoint(firstEndpoint)
	return service.Run(ctx, func(ctx context.Context, s service.Scope) error {
		// Connect to WebSocket endpoint
		client, err := ethclient.Dial(wsEndpoint)
		if err != nil {
			return fmt.Errorf("failed to connect to WebSocket endpoint %s: %w", wsEndpoint, err)
		}
		headers := make(chan *types.Header)
		sub, err := client.SubscribeNewHead(ctx, headers)
		if err != nil {
			return fmt.Errorf("❌ Failed to subscribe to new blocks: %w", err)
		}
		defer sub.Unsubscribe()

		// Channel for pending blocks to be processed asynchronously
		pendingBlocks := make(chan pendingBlock, 100)

		// Background error handler for subscription
		s.SpawnBg(func() error {
			subErr, err := utils.Recv(ctx, sub.Err())
			if err != nil {
				return err
			}
			return subErr
		})

		// Background worker to fetch full blocks with retries
		s.Spawn(func() error {
			return bc.processBlocksWorker(ctx, client, pendingBlocks)
		})

		log.Printf("📡 Subscribed to new blocks on %s", wsEndpoint)
		for ctx.Err() == nil {
			header, err := utils.Recv(ctx, headers)
			if err != nil {
				return err
			}
			// Queue block for async processing (non-blocking)
			select {
			case pendingBlocks <- pendingBlock{header: header, receivedTime: time.Now()}:
			default:
				log.Printf("⚠️ Block processing queue full, dropping block %d", header.Number.Uint64())
			}
		}
		return ctx.Err()
	})
}

// processBlocksWorker processes pending blocks in the background with retries
func (bc *BlockCollector) processBlocksWorker(ctx context.Context, client *ethclient.Client, pendingBlocks <-chan pendingBlock) error {
	const (
		maxRetries    = 5
		retryInterval = 200 * time.Millisecond
		fetchTimeout  = 5 * time.Second
	)

	for ctx.Err() == nil {
		pending, err := utils.Recv(ctx, pendingBlocks)
		if err != nil {
			return err
		}

		// Fetch full block with retries
		txCount, err := bc.fetchBlockWithRetries(ctx, client, pending.header, maxRetries, retryInterval, fetchTimeout)
		if err != nil {
			log.Printf("⚠️ Failed to fetch block %d after %d retries: %v (using 0 tx count)",
				pending.header.Number.Uint64(), maxRetries, err)
			txCount = 0
		}

		bc.recordBlockMetrics(ctx, pending.header, pending.receivedTime, txCount)
	}
	return ctx.Err()
}

// fetchBlockWithRetries attempts to fetch a full block with exponential backoff
func (bc *BlockCollector) fetchBlockWithRetries(ctx context.Context, client *ethclient.Client, header *types.Header, maxRetries int, retryInterval, timeout time.Duration) (int, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry with exponential backoff
			backoff := retryInterval * time.Duration(1<<(attempt-1))
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			case <-time.After(backoff):
			}
		}

		// Create a timeout context for this fetch attempt
		fetchCtx, cancel := context.WithTimeout(ctx, timeout)
		// Use BlockByNumber instead of BlockByHash to handle potential reorgs
		block, err := client.BlockByNumber(fetchCtx, header.Number)
		cancel()

		if err == nil {
			return len(block.Transactions()), nil
		}

		lastErr = err

		// If it's a "not found" error, retry (block might not be propagated yet)
		if errors.Is(err, ethereum.NotFound) {
			continue
		}

		// For other errors, also retry
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
	}

	return 0, lastErr
}

// recordBlockMetrics records the metrics for a processed block
func (bc *BlockCollector) recordBlockMetrics(ctx context.Context, header *types.Header, receivedTime time.Time, txCount int) {
	for stats := range bc.stats.Lock() {
		blockNum := header.Number.Uint64()
		gasUsed := header.GasUsed
		chainAttr := metric.WithAttributes(attribute.String("chain_id", bc.seiChainID))

		metrics.gasUsed.Record(ctx, int64(gasUsed), chainAttr)
		metrics.blockTxCount.Record(ctx, int64(txCount), chainAttr)

		// Update max block number
		if blockNum > stats.maxBlockNum {
			metrics.blockNumber.Record(ctx, int64(blockNum), chainAttr)
			stats.maxBlockNum = blockNum
		}

		// Track gas used and tx counts
		stats.allGasUsed = append(stats.allGasUsed, gasUsed)
		stats.windowGasUsed = append(stats.windowGasUsed, gasUsed)
		stats.allTxCounts = append(stats.allTxCounts, txCount)
		stats.windowTxCounts = append(stats.windowTxCounts, txCount)
		stats.totalTxCount += uint64(txCount)
		stats.windowTotalTxs += uint64(txCount)

		// Calculate time between blocks and TPS
		if !stats.lastBlockTime.IsZero() {
			timeBetween := receivedTime.Sub(stats.lastBlockTime)
			metrics.blockTime.Record(ctx, timeBetween.Seconds(), chainAttr)
			stats.allBlockTimes = append(stats.allBlockTimes, timeBetween)
			stats.windowBlockTimes = append(stats.windowBlockTimes, timeBetween)

			// Calculate TPS for this block
			if timeBetween.Seconds() > 0 {
				tps := float64(txCount) / timeBetween.Seconds()
				metrics.blockTPS.Record(ctx, tps, chainAttr)
				stats.allTPS = append(stats.allTPS, tps)
				stats.windowTPS = append(stats.windowTPS, tps)
			}
		}

		stats.lastBlockTime = receivedTime

		// Limit history to prevent memory growth (keep last 1000 entries)
		if len(stats.allBlockTimes) > 1000 {
			stats.allBlockTimes = stats.allBlockTimes[len(stats.allBlockTimes)-1000:]
		}
		if len(stats.allGasUsed) > 1000 {
			stats.allGasUsed = stats.allGasUsed[len(stats.allGasUsed)-1000:]
		}
		if len(stats.allTxCounts) > 1000 {
			stats.allTxCounts = stats.allTxCounts[len(stats.allTxCounts)-1000:]
		}
		if len(stats.allTPS) > 1000 {
			stats.allTPS = stats.allTPS[len(stats.allTPS)-1000:]
		}
		if len(stats.windowBlockTimes) > 1000 {
			stats.windowBlockTimes = stats.windowBlockTimes[len(stats.windowBlockTimes)-1000:]
		}
		if len(stats.windowGasUsed) > 1000 {
			stats.windowGasUsed = stats.windowGasUsed[len(stats.windowGasUsed)-1000:]
		}
		if len(stats.windowTxCounts) > 1000 {
			stats.windowTxCounts = stats.windowTxCounts[len(stats.windowTxCounts)-1000:]
		}
		if len(stats.windowTPS) > 1000 {
			stats.windowTPS = stats.windowTPS[len(stats.windowTPS)-1000:]
		}
	}
}

// GetBlockStats returns current block statistics
func (bc *BlockCollector) GetBlockStats() BlockStats {
	for bc := range bc.stats.Lock() {
		stats := BlockStats{
			MaxBlockNumber: bc.maxBlockNum,
			SampleCount:    len(bc.allBlockTimes),
			TotalTxCount:   bc.totalTxCount,
		}

		// Calculate block time percentiles
		if len(bc.allBlockTimes) > 0 {
			sortedTimes := make([]time.Duration, len(bc.allBlockTimes))
			copy(sortedTimes, bc.allBlockTimes)
			sort.Slice(sortedTimes, func(i, j int) bool {
				return sortedTimes[i] < sortedTimes[j]
			})

			stats.P50BlockTime = calculatePercentile(sortedTimes, 50)
			stats.P99BlockTime = calculatePercentile(sortedTimes, 99)
			stats.MaxBlockTime = sortedTimes[len(sortedTimes)-1]
		}

		// Calculate gas used percentiles
		if len(bc.allGasUsed) > 0 {
			sortedGas := make([]uint64, len(bc.allGasUsed))
			copy(sortedGas, bc.allGasUsed)
			sort.Slice(sortedGas, func(i, j int) bool {
				return sortedGas[i] < sortedGas[j]
			})

			stats.P50GasUsed = calculateGasPercentile(sortedGas, 50)
			stats.P99GasUsed = calculateGasPercentile(sortedGas, 99)
			stats.MaxGasUsed = sortedGas[len(sortedGas)-1]
		}

		// Calculate TPS percentiles
		if len(bc.allTPS) > 0 {
			sortedTPS := make([]float64, len(bc.allTPS))
			copy(sortedTPS, bc.allTPS)
			sort.Float64s(sortedTPS)

			stats.P50TPS = calculateTPSPercentile(sortedTPS, 50)
			stats.P99TPS = calculateTPSPercentile(sortedTPS, 99)
			stats.MaxTPS = sortedTPS[len(sortedTPS)-1]

			// Calculate average TPS
			var sum float64
			for _, tps := range sortedTPS {
				sum += tps
			}
			stats.AvgTPS = sum / float64(len(sortedTPS))
		}

		return stats
	}
	panic("unreachable")
}

// GetWindowBlockStats returns current window-based block statistics
func (bc *BlockCollector) GetWindowBlockStats() BlockStats {
	for bc := range bc.stats.Lock() {
		stats := BlockStats{
			MaxBlockNumber: bc.maxBlockNum,
			SampleCount:    len(bc.windowBlockTimes),
			TotalTxCount:   bc.windowTotalTxs,
		}

		// Calculate block time percentiles for current window
		if len(bc.windowBlockTimes) > 0 {
			sortedTimes := make([]time.Duration, len(bc.windowBlockTimes))
			copy(sortedTimes, bc.windowBlockTimes)
			sort.Slice(sortedTimes, func(i, j int) bool {
				return sortedTimes[i] < sortedTimes[j]
			})

			stats.P50BlockTime = calculatePercentile(sortedTimes, 50)
			stats.P99BlockTime = calculatePercentile(sortedTimes, 99)
			stats.MaxBlockTime = sortedTimes[len(sortedTimes)-1]
		}

		// Calculate gas used percentiles for current window
		if len(bc.windowGasUsed) > 0 {
			sortedGas := make([]uint64, len(bc.windowGasUsed))
			copy(sortedGas, bc.windowGasUsed)
			sort.Slice(sortedGas, func(i, j int) bool {
				return sortedGas[i] < sortedGas[j]
			})

			stats.P50GasUsed = calculateGasPercentile(sortedGas, 50)
			stats.P99GasUsed = calculateGasPercentile(sortedGas, 99)
			stats.MaxGasUsed = sortedGas[len(sortedGas)-1]
		}

		// Calculate TPS percentiles for current window
		if len(bc.windowTPS) > 0 {
			sortedTPS := make([]float64, len(bc.windowTPS))
			copy(sortedTPS, bc.windowTPS)
			sort.Float64s(sortedTPS)

			stats.P50TPS = calculateTPSPercentile(sortedTPS, 50)
			stats.P99TPS = calculateTPSPercentile(sortedTPS, 99)
			stats.MaxTPS = sortedTPS[len(sortedTPS)-1]

			// Calculate average TPS
			var sum float64
			for _, tps := range sortedTPS {
				sum += tps
			}
			stats.AvgTPS = sum / float64(len(sortedTPS))
		}

		return stats
	}
	panic("unreachable")
}

func (bc *BlockCollector) GetWindowBlockTimePercentile(percentile int) time.Duration {
	for bc := range bc.stats.Lock() {
		sortedTimes := make([]time.Duration, len(bc.windowBlockTimes))
		copy(sortedTimes, bc.windowBlockTimes)
		sort.Slice(sortedTimes, func(i, j int) bool {
			return sortedTimes[i] < sortedTimes[j]
		})
		return calculatePercentile(sortedTimes, percentile)
	}
	panic("unreachable")
}

// ResetWindowStats resets the window-based statistics for the next reporting period
func (bc *BlockCollector) ResetWindowStats() {
	for bc := range bc.stats.Lock() {
		bc.windowBlockTimes = make([]time.Duration, 0)
		bc.windowGasUsed = make([]uint64, 0)
		bc.windowTxCounts = make([]int, 0)
		bc.windowTPS = make([]float64, 0)
		bc.windowTotalTxs = 0
		bc.windowStart = time.Now()
	}
}

// calculateGasPercentile calculates the given percentile from sorted gas values
func calculateGasPercentile(sorted []uint64, percentile int) uint64 {
	if len(sorted) == 0 {
		return 0
	}

	index := (percentile * (len(sorted) - 1)) / 100
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	return sorted[index]
}

// calculateTPSPercentile calculates the given percentile from sorted TPS values
func calculateTPSPercentile(sorted []float64, percentile int) float64 {
	if len(sorted) == 0 {
		return 0
	}

	index := (percentile * (len(sorted) - 1)) / 100
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	return sorted[index]
}

// BlockStats represents block-related statistics
type BlockStats struct {
	MaxBlockNumber uint64        `json:"max_block_number"`
	P50BlockTime   time.Duration `json:"p50_block_time"`
	P99BlockTime   time.Duration `json:"p99_block_time"`
	MaxBlockTime   time.Duration `json:"max_block_time"`
	P50GasUsed     uint64        `json:"p50_gas_used"`
	P99GasUsed     uint64        `json:"p99_gas_used"`
	MaxGasUsed     uint64        `json:"max_gas_used"`
	SampleCount    int           `json:"sample_count"`
	// TPS metrics (actual on-chain throughput)
	TotalTxCount uint64  `json:"total_tx_count"`
	P50TPS       float64 `json:"p50_tps"`
	P99TPS       float64 `json:"p99_tps"`
	MaxTPS       float64 `json:"max_tps"`
	AvgTPS       float64 `json:"avg_tps"`
}

// FormatBlockStats returns a formatted string representation of block statistics
func (bs BlockStats) FormatBlockStats() string {
	if bs.SampleCount == 0 {
		return "block stats: no data available"
	}

	return fmt.Sprintf("block height=%d, times(p50=%v p99=%v max=%v), gas(p50=%d p99=%d max=%d), tps(avg=%.1f p50=%.1f p99=%.1f max=%.1f) txs=%d blocks=%d",
		bs.MaxBlockNumber,
		bs.P50BlockTime.Round(time.Millisecond),
		bs.P99BlockTime.Round(time.Millisecond),
		bs.MaxBlockTime.Round(time.Millisecond),
		bs.P50GasUsed,
		bs.P99GasUsed,
		bs.MaxGasUsed,
		bs.AvgTPS,
		bs.P50TPS,
		bs.P99TPS,
		bs.MaxTPS,
		bs.TotalTxCount,
		bs.SampleCount,
	)
}
