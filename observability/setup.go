// Package observability configures OpenTelemetry for sei-load.
// See README.md for invariants and exemplar requirements.
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

// RunScope identifies a single sei-load invocation. Values ride on the OTel
// Resource, not per-sample metric labels (see README cardinality rationale).
type RunScope struct {
	ServiceVersion string // sei-load's own build version (distinct from CommitID, which names what's under test)
	RunID          string // e.g. GHA run id for autobake, benchmark job id elsewhere
	ChainID        string // ephemeral test chain for this run
	CommitID       string // sei-chain commit under test; exported as seiload.commit_id + 8-char seiload.commit_id_short
	Workload       string // "autobake" | "benchmark" | "loadtest"; alert rules match on this
	InstanceID     string // unique per process; falls back to hostname. Disambiguates cluster-of-seiload pods.
}

// RunScopeFromEnv reads SEILOAD_* and OTEL_SERVICE_VERSION. Missing values stay empty.
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

func shortCommit(full string) string {
	if len(full) <= 8 {
		return full
	}
	return full[:8]
}

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

	// NewSchemaless avoids "conflicting Schema URL" on merge with Default().
	return resource.Merge(
		resource.Default(),
		resource.NewSchemaless(attrs...),
	)
}

// Config knobs for Setup. Zero value is usable (Prometheus-only, no OTLP).
type Config struct {
	RunScope            RunScope
	PrometheusNamespace string // defaults to "seiload"
	OTLPEndpoint        string // non-empty activates OTLP gRPC for metrics + traces
}

// Setup installs MeterProvider, TracerProvider, W3C propagator, and returns a
// shutdown func. Call once from main before any telemetry emits.
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

	if cfg.OTLPEndpoint != "" {
		metricExp, mErr := otlpmetricgrpc.New(ctx,
			otlpmetricgrpc.WithEndpoint(stripScheme(cfg.OTLPEndpoint)),
			otlpmetricgrpc.WithInsecure(),
		)
		if mErr != nil {
			return nil, fmt.Errorf("otlp metric exporter: %w", mErr)
		}
		meterOpts = append(meterOpts, sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp)))

		traceExp, tErr := otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(stripScheme(cfg.OTLPEndpoint)),
			otlptracegrpc.WithInsecure(),
		)
		if tErr != nil {
			return nil, fmt.Errorf("otlp trace exporter: %w", tErr)
		}
		tracerOpts = append(tracerOpts, sdktrace.WithBatcher(traceExp))
	}

	mp := sdkmetric.NewMeterProvider(meterOpts...)
	otel.SetMeterProvider(mp)
	tp := sdktrace.NewTracerProvider(tracerOpts...)
	otel.SetTracerProvider(tp)

	// Provider.Shutdown cascades flush into exporters; see README.
	shutdowns := []func(context.Context) error{mp.Shutdown, tp.Shutdown}

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

// stripScheme trims http://, https://, grpc://, dns:/// so the value fits
// otlpmetricgrpc.WithEndpoint (which wants bare host:port).
func stripScheme(endpoint string) string {
	for _, prefix := range []string{"http://", "https://", "grpc://", "dns:///"} {
		if strings.HasPrefix(endpoint, prefix) {
			return strings.TrimPrefix(endpoint, prefix)
		}
	}
	return endpoint
}
