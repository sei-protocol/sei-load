package stats

import (
	"context"
	"time"
)

// EmitRunSummary records the three run-summary gauges (duration, final TPS,
// accepted-txs total) for this Collector's run. Call once at shutdown, after
// the collector has stopped accepting new samples.
//
// The run-summary metrics carry no per-sample attributes — they rely on the
// Resource attributes installed by observability.Setup (run_id, chain_id,
// commit_id_short, workload) to identify the run. This produces exactly
// one series per metric per run after the Resource join in Prometheus
// (target_info), the shape needed for cross-run benchmark dashboards.
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
