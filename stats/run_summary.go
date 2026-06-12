package stats

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// RunSummary carries the arrival-model accounting the collector does not track
// itself (the dispatcher owns it) into the run-summary gauges.
type RunSummary struct {
	// ArrivalModel is "open_loop" or "closed_loop"; tags the dropped gauge so a
	// nonzero drop count is attributable to the model that produced it.
	ArrivalModel string
	// Dropped is the count of open-loop txs shed on in-flight saturation. These
	// never reach the inclusion tracker, so they carry a zero InclusionTime;
	// keep them out of later inclusion-rate denominators (PLT-463 forward-note).
	Dropped uint64
}

// EmitRunSummary records the run-summary gauges. Call once at shutdown.
func (c *Collector) EmitRunSummary(ctx context.Context, summary RunSummary) {
	c.mu.RLock()
	duration := time.Since(c.startTime)
	totalTxs := c.totalTxs
	finalTPS := c.overallTpsWindow.maxTPS
	c.mu.RUnlock()

	runDurationSeconds.Record(ctx, duration.Seconds())
	runTPSFinal.Record(ctx, finalTPS)
	runTxsAcceptedTotal.Record(ctx, int64(totalTxs))
	runTxsDroppedTotal.Record(ctx, int64(summary.Dropped),
		metric.WithAttributes(attribute.String("arrival_model", summary.ArrivalModel)))
}
