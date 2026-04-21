// Package observability configures OpenTelemetry for sei-load.
//
// Setup wires a MeterProvider + TracerProvider backed by a Resource populated
// from the SEILOAD_* run-scope env vars, exports metrics via a Prometheus
// reader (always) and an OTLP gRPC exporter (when OTEL_EXPORTER_OTLP_ENDPOINT
// is set), and installs the composite TraceContext+Baggage propagator for
// W3C context propagation on outbound HTTP.
//
// See docs/designs/sei-load-observability.md (in the platform repo) for the
// design rationale — run scope rides as Resource attributes (never per-sample
// labels), exemplars are enabled via trace_based filter by default, and the
// package supports future cluster-of-seiload deployments via service.instance.id.
package observability

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// RunScope is the set of attributes that identify a single sei-load process
// invocation. The values travel on the OTel Resource (one series per process)
// rather than on per-sample metric labels — see the design doc for the
// cardinality rationale.
type RunScope struct {
	// ServiceVersion is sei-load's own build version, separate from CommitID
	// below (which names the commit under test). Built via -ldflags or pulled
	// from runtime/debug.ReadBuildInfo. Optional.
	ServiceVersion string

	// RunID uniquely identifies a single run (e.g., GHA run id for autobake,
	// benchmark job id for the benchmarks namespace).
	RunID string

	// ChainID names the ephemeral test chain for this run (e.g.
	// "autobake-24702101866-1").
	ChainID string

	// CommitID is the full sei-chain commit SHA being exercised. Lives on the
	// Resource as seiload.commit_id and (truncated to 8 chars) as
	// seiload.commit_id_short for label-friendly uses.
	CommitID string

	// Workload names the invocation's purpose: "autobake", "benchmark",
	// "loadtest". Used by alert rules to scope matchers.
	Workload string

	// InstanceID uniquely identifies a single sei-load process within a run —
	// typically the pod name via the downward API. Populated for cluster-of-
	// seiload deployments where multiple pods share a RunID/ChainID but each
	// emits its own metrics stream. Exported as service.instance.id
	// (OTel semconv) so dashboards can aggregate or disaggregate naturally.
	InstanceID string
}

// RunScopeFromEnv reads SEILOAD_RUN_ID / SEILOAD_CHAIN_ID / SEILOAD_COMMIT_ID
// / SEILOAD_WORKLOAD / SEILOAD_INSTANCE_ID and OTEL_SERVICE_VERSION. Missing
// values remain empty strings — the caller decides whether that's fatal.
func RunScopeFromEnv() RunScope {
	return RunScope{
		ServiceVersion: os.Getenv("OTEL_SERVICE_VERSION"),
		RunID:          os.Getenv("SEILOAD_RUN_ID"),
		ChainID:        os.Getenv("SEILOAD_CHAIN_ID"),
		CommitID:       os.Getenv("SEILOAD_COMMIT_ID"),
		Workload:       os.Getenv("SEILOAD_WORKLOAD"),
		InstanceID:     os.Getenv("SEILOAD_INSTANCE_ID"),
	}
}

// shortCommit returns the first 8 chars of a commit SHA for label-friendly
// use (Alertmanager grouping, dashboard variable values). Empty in / empty out.
func shortCommit(full string) string {
	if len(full) <= 8 {
		return full
	}
	return full[:8]
}

// buildResource composes the OTel Resource from the run scope plus standard
// service identity. service.instance.id defaults to the hostname when
// InstanceID is empty — this keeps single-pod deployments behaving sensibly
// without requiring explicit wiring.
func buildResource(rs RunScope) (*resource.Resource, error) {
	instanceID := rs.InstanceID
	if instanceID == "" {
		host, err := os.Hostname()
		if err == nil {
			instanceID = host
		}
	}

	attrs := []attribute.KeyValue{
		semconv.ServiceName("seiload"),
		semconv.ServiceInstanceID(instanceID),
	}
	if rs.ServiceVersion != "" {
		attrs = append(attrs, semconv.ServiceVersion(rs.ServiceVersion))
	}
	if rs.RunID != "" {
		attrs = append(attrs, attribute.String("seiload.run_id", rs.RunID))
	}
	if rs.ChainID != "" {
		attrs = append(attrs, attribute.String("seiload.chain_id", rs.ChainID))
	}
	if rs.CommitID != "" {
		attrs = append(attrs,
			attribute.String("seiload.commit_id", rs.CommitID),
			attribute.String("seiload.commit_id_short", shortCommit(rs.CommitID)),
		)
	}
	if rs.Workload != "" {
		attrs = append(attrs, attribute.String("seiload.workload", rs.Workload))
	}

	// NewSchemaless lets us merge with resource.Default() (which tracks its
	// own schema URL) without the "conflicting Schema URL" error that occurs
	// when both sides declare a schema. App-specific attributes are schema-
	// agnostic; the SDK Default covers the OS/process identity schema.
	return resource.Merge(
		resource.Default(),
		resource.NewSchemaless(attrs...),
	)
}

// Config is the knob set for Setup. Zero-value Config is usable: it produces a
// Prometheus-only metrics pipeline, no OTLP, no tracing exporter. Populate
// PrometheusNamespace to set the exporter's metric name prefix.
type Config struct {
	// RunScope populates the Resource. Typically built via RunScopeFromEnv().
	RunScope RunScope

	// PrometheusNamespace prefixes all exported metric names. Defaults to
	// "seiload" when empty — matches the existing convention.
	PrometheusNamespace string

	// OTLPEndpoint, when non-empty, activates the OTLP gRPC exporter for
	// BOTH metrics and traces, wired to this endpoint. Use the standard
	// OTEL_EXPORTER_OTLP_ENDPOINT env var to populate this — see Setup.
	OTLPEndpoint string
}

// Setup installs the global MeterProvider and TracerProvider, wires the
// Prometheus reader (always) and the OTLP gRPC exporter (when OTLPEndpoint is
// non-empty), attaches the composite W3C TraceContext+Baggage propagator, and
// returns a shutdown function that flushes and stops all providers.
//
// Call once from main, before any code path that emits telemetry. The
// Meter/Tracer accessors in this package respect the configured providers;
// code that still uses otel.Meter / otel.Tracer at package init will silently
// capture the NoOp provider and drop its emissions — migrate those sites to
// the lazy accessors (see observability.Meter / observability.Tracer).
func Setup(ctx context.Context, cfg Config) (shutdown func(context.Context) error, err error) {
	res, err := buildResource(cfg.RunScope)
	if err != nil {
		return nil, fmt.Errorf("build resource: %w", err)
	}

	ns := cfg.PrometheusNamespace
	if ns == "" {
		ns = "seiload"
	}
	promExporter, err := prometheus.New(prometheus.WithNamespace(ns))
	if err != nil {
		return nil, fmt.Errorf("prometheus exporter: %w", err)
	}

	meterOpts := []sdkmetric.Option{
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(promExporter),
	}
	tracerOpts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
	}

	shutdowns := []func(context.Context) error{
		promExporter.Shutdown,
	}

	if cfg.OTLPEndpoint != "" {
		metricExp, mErr := otlpmetricgrpc.New(ctx,
			otlpmetricgrpc.WithEndpoint(stripScheme(cfg.OTLPEndpoint)),
			otlpmetricgrpc.WithInsecure(),
		)
		if mErr != nil {
			return nil, fmt.Errorf("otlp metric exporter: %w", mErr)
		}
		meterOpts = append(meterOpts, sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp)))
		shutdowns = append(shutdowns, metricExp.Shutdown)

		traceExp, tErr := otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(stripScheme(cfg.OTLPEndpoint)),
			otlptracegrpc.WithInsecure(),
		)
		if tErr != nil {
			return nil, fmt.Errorf("otlp trace exporter: %w", tErr)
		}
		tracerOpts = append(tracerOpts, sdktrace.WithBatcher(traceExp))
		shutdowns = append(shutdowns, traceExp.Shutdown)
	}

	mp := sdkmetric.NewMeterProvider(meterOpts...)
	otel.SetMeterProvider(mp)
	shutdowns = append(shutdowns, mp.Shutdown)

	tp := sdktrace.NewTracerProvider(tracerOpts...)
	otel.SetTracerProvider(tp)
	shutdowns = append(shutdowns, tp.Shutdown)

	// Composite propagator: traceparent for trace context, baggage for
	// arbitrary cross-process K/V. Without this, otelhttp will create spans
	// but NOT inject traceparent on outbound requests — silent propagation
	// failure, a well-known OTel footgun.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return func(shutdownCtx context.Context) error {
		var errs []error
		for _, fn := range shutdowns {
			if sErr := fn(shutdownCtx); sErr != nil {
				errs = append(errs, sErr)
			}
		}
		return errors.Join(errs...)
	}, nil
}

// stripScheme removes a leading scheme like "http://" or "grpc://" from an
// endpoint — otlpmetricgrpc.WithEndpoint expects host:port without a scheme.
func stripScheme(endpoint string) string {
	for _, prefix := range []string{"http://", "https://", "grpc://", "dns:///"} {
		if strings.HasPrefix(endpoint, prefix) {
			return strings.TrimPrefix(endpoint, prefix)
		}
	}
	return endpoint
}

