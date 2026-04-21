package observability

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Meter returns a Meter for the given instrumentation name. The contract is
// identical to otel.Meter; the indirection exists so call sites can wrap with
// sync.OnceValue at package scope for lazy acquisition without capturing the
// NoOp provider at init time:
//
//	var senderMeter = sync.OnceValue(func() metric.Meter {
//	    return observability.Meter("github.com/sei-protocol/sei-load/sender")
//	})
//
//	// inside functions: senderMeter().Float64Histogram(...)
//
// Calling observability.Meter (or otel.Meter) at package init is a bug — the
// SDK setup hasn't run yet. The sync.OnceValue wrapper defers acquisition
// until the first call site, by which time main has invoked Setup.
func Meter(name string) metric.Meter {
	return otel.Meter(name)
}

// Tracer returns a Tracer for the given instrumentation name. Same lazy-
// acquisition contract as Meter above.
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}
