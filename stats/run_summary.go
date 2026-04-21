package stats

import (
	"context"
	"time"
)

// EmitRunSummary records the run-summary gauges. Call once at shutdown.
func (c *Collector) EmitRunSummary(ctx context.Context) {
	c.mu.RLock()
	duration := time.Since(c.startTime)
	totalTxs := c.totalTxs
	finalTPS := c.overallTpsWindow.maxTPS
	c.mu.RUnlock()

	runDurationSeconds.Record(ctx, duration.Seconds())
	runTPSFinal.Record(ctx, finalTPS)
	runTxsAcceptedTotal.Record(ctx, int64(totalTxs))
}
