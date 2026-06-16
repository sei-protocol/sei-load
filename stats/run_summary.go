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
	// Dropped is the count of open-loop txs shed on in-flight saturation. A
	// dropped tx never reaches the inclusion tracker and carries no
	// InclusionTime, so it must stay out of inclusion-rate denominators: the
	// denominator is succeeded txs (those that reached a sender), never the full
	// scheduled count.
	Dropped uint64
	// Failed is the count of admitted open-loop txs whose send completed with an
	// error. Like Dropped, a failed tx reached no inclusion tracker; it is
	// reported so the conservation invariant (see sender package doc:
	// scheduled = dropped + succeeded + failed) is auditable from the run summary.
	Failed uint64

	// Inclusion-stage tally (see sender/doc.go). The conservation identity is
	// registered == Included + Expired + InflightAtShutdown, with
	// registered ⊆ succeeded. InclusionTracked disambiguates a not-tracked run
	// (all zero, flag false) from a tracked run with no inclusions yet.
	// TODO(PLT-467): owns run-summary schema versioning for these fields.
	InclusionTracked bool
	// Included is the count of txs the tracker observed on-chain (stamped).
	Included uint64
	// Expired is the count of registered txs reaped un-included after reapAfter.
	Expired uint64
	// DroppedAtCap is the count of successful sends rejected at the in-flight cap;
	// excluded from the inclusion denominator (they were never registered).
	DroppedAtCap uint64
	// InflightAtShutdown is len(inflight) read after workers and tracker joined.
	InflightAtShutdown uint64
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
	runTxsFailedTotal.Record(ctx, int64(summary.Failed),
		metric.WithAttributes(attribute.String("arrival_model", summary.ArrivalModel)))
	if summary.InclusionTracked {
		runInflightAtShutdown.Record(ctx, int64(summary.InflightAtShutdown))
	}
}
