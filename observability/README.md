# observability

OTel SDK bootstrap for sei-load. `Setup` installs MeterProvider + TracerProvider, Resource, Prometheus + optional OTLP exporters, and the W3C propagator.

Full design: [`docs/designs/sei-load-observability.md`](https://github.com/sei-protocol/platform/blob/main/docs/designs/sei-load-observability.md) in `sei-protocol/platform`.

## Invariants you won't want to break

- **Run scope rides on the Resource, never per-sample labels.** `run_id`, `commit_id`, `chain_id`, `workload`, `service.instance.id` are process-lifetime constants. Putting them on every metric sample would multiply cardinality by (runs × endpoints × workers × buckets). Prometheus joins them in via `target_info{run_id=...}`.
- **`newSchemaless` merging with `resource.Default()`.** `resource.NewWithAttributes(semconv.SchemaURL, ...)` collides with `Default()`'s own schema URL and errors. Schemaless app attributes + schema-stamped Default is the supported pattern.
- **Shutdown providers before exporters.** `MeterProvider.Shutdown` / `TracerProvider.Shutdown` cascade a final flush into their readers and exporters. Explicit exporter shutdown before provider shutdown drops the last OTLP batch (PeriodicReader + BatchSpanProcessor buffer in memory). Prometheus pull-readers are immune but we keep the invariant uniform.
- **Composite propagator is not optional.** Without `SetTextMapPropagator(TraceContext + Baggage)`, `otelhttp` creates spans but silently omits `traceparent` on outbound requests.

## Exemplar requirements

Trace-ID exemplars on histograms need all three:

1. OTel SDK ≥ v1.28 (we pin v1.43).
2. `promhttp.HandlerFor(DefaultGatherer, HandlerOpts{EnableOpenMetrics: true})` — the default `promhttp.Handler()` never negotiates OpenMetrics and silently strips exemplars regardless of the scraper's `Accept` header.
3. Prometheus server with `enableFeatures: [exemplar-storage]` (set in `clusters/harbor/monitoring/prometheus-operator.yaml`).

## Cluster-of-seiload

`service.instance.id` defaults to `os.Hostname()` when `SEILOAD_INSTANCE_ID` is unset, so multi-pod deployments disambiguate by pod name automatically. In Kubernetes, wire the env var via downward API (`fieldRef: metadata.name`) for a human-readable attribute.
