package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
)

func TestShortCommit(t *testing.T) {
	require.Equal(t, "", shortCommit(""))
	require.Equal(t, "abc", shortCommit("abc"))
	require.Equal(t, "abcdef12", shortCommit("abcdef1234567890"))
	require.Equal(t, "12345678", shortCommit("12345678"))
}

func TestRunScopeFromEnv(t *testing.T) {
	t.Setenv("SEILOAD_RUN_ID", "run-42")
	t.Setenv("SEILOAD_CHAIN_ID", "autobake-42-1")
	t.Setenv("SEILOAD_COMMIT_ID", "deadbeefcafef00d")
	t.Setenv("SEILOAD_WORKLOAD", "autobake")
	t.Setenv("SEILOAD_INSTANCE_ID", "seiload-abc-0")
	t.Setenv("OTEL_SERVICE_VERSION", "v1.2.3")

	rs := RunScopeFromEnv()
	require.Equal(t, "run-42", rs.RunID)
	require.Equal(t, "autobake-42-1", rs.ChainID)
	require.Equal(t, "deadbeefcafef00d", rs.CommitID)
	require.Equal(t, "autobake", rs.Workload)
	require.Equal(t, "seiload-abc-0", rs.InstanceID)
	require.Equal(t, "v1.2.3", rs.ServiceVersion)
}

func TestBuildResource_FullScope(t *testing.T) {
	rs := RunScope{
		ServiceVersion: "v1.2.3",
		RunID:          "run-42",
		ChainID:        "autobake-42-1",
		CommitID:       "deadbeefcafef00d",
		Workload:       "autobake",
		InstanceID:     "seiload-abc-0",
	}
	res, err := buildResource(rs)
	require.NoError(t, err)

	want := map[string]string{
		"service.name":             "seiload",
		"service.version":          "v1.2.3",
		"service.instance.id":      "seiload-abc-0",
		"seiload.run_id":           "run-42",
		"seiload.chain_id":         "autobake-42-1",
		"seiload.commit_id":        "deadbeefcafef00d",
		"seiload.commit_id_short": "deadbeef",
		"seiload.workload":         "autobake",
	}
	got := resourceAttrs(res.Attributes())
	for k, v := range want {
		require.Equal(t, v, got[k], "attr %q", k)
	}
}

func TestBuildResource_EmptyScope(t *testing.T) {
	// Empty RunScope should still produce a Resource with service.name.
	// service.instance.id falls back to os.Hostname(); just ensure it's not
	// empty so downstream doesn't see a missing label.
	res, err := buildResource(RunScope{})
	require.NoError(t, err)
	got := resourceAttrs(res.Attributes())
	require.Equal(t, "seiload", got["service.name"])
	require.NotEmpty(t, got["service.instance.id"], "service.instance.id should fall back to hostname")
	require.Empty(t, got["seiload.run_id"])
	require.Empty(t, got["seiload.commit_id_short"])
}

func TestSetup_PrometheusOnly(t *testing.T) {
	// Ensure OTLP is NOT activated when endpoint is empty.
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "") // belt-and-suspenders

	ctx := context.Background()
	shutdown, err := Setup(ctx, Config{
		RunScope:            RunScope{RunID: "test-run"},
		PrometheusNamespace: "seiload_test",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = shutdown(ctx) })

	// Global providers should be populated (not NoOp).
	require.NotNil(t, otel.GetMeterProvider())
	require.NotNil(t, otel.GetTracerProvider())

	// Propagator should include TraceContext + Baggage fields.
	fields := otel.GetTextMapPropagator().Fields()
	require.Contains(t, fields, "traceparent", "traceparent field must be present for W3C trace propagation")
	require.Contains(t, fields, "baggage", "baggage field must be present for cross-process attribute propagation")
}

func TestSetup_PropagatorInstalled(t *testing.T) {
	// After Setup, the global propagator should be a CompositeTextMapPropagator
	// with TraceContext + Baggage — without this, otelhttp creates spans but
	// never injects traceparent, and cross-process traces silently break.
	ctx := context.Background()
	shutdown, err := Setup(ctx, Config{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = shutdown(ctx) })

	p := otel.GetTextMapPropagator()
	require.NotNil(t, p)

	// TraceContext registers "traceparent" + "tracestate"; Baggage registers "baggage".
	fields := p.Fields()
	require.Contains(t, fields, "traceparent")
	require.Contains(t, fields, "tracestate")
	require.Contains(t, fields, "baggage")

	// Sanity check with a sample carrier: injection shouldn't panic.
	p.Inject(ctx, propagation.MapCarrier{})
}

func TestStripScheme(t *testing.T) {
	cases := map[string]string{
		"":                        "",
		"otel-collector:4317":     "otel-collector:4317",
		"http://host:4317":        "host:4317",
		"https://host:4317":       "host:4317",
		"grpc://host:4317":        "host:4317",
		"dns:///host:4317":        "host:4317",
		"host:4317/path":          "host:4317/path",
		"grpc://already/stripped": "already/stripped",
	}
	for in, want := range cases {
		require.Equal(t, want, stripScheme(in), "input=%q", in)
	}
}

func resourceAttrs(kvs []attribute.KeyValue) map[string]string {
	out := make(map[string]string, len(kvs))
	for _, kv := range kvs {
		out[string(kv.Key)] = kv.Value.Emit()
	}
	return out
}
