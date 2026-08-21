package sender

import (
	"testing"

	"github.com/sei-protocol/sei-load/stats"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// TestSendPathLabelsEveryTransaction reads back what the SDK collected, which is
// the only place a missing or misspelled attribute shows. An emit that leaves
// operation empty produces a stream the Prometheus exporter writes as an empty
// label; that line collides with the omitted-attribute stream at ingest and one
// of the two is discarded, with no scrape error and the target still up. Nothing
// downstream reports that loss, so the assertion belongs here.
func TestSendPathLabelsEveryTransaction(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	otel.SetMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))

	client, err := newEthClient(t.Context(), &ethClientConfig{
		DryRun:    true,
		ChainID:   "test-chain",
		Collector: stats.NewCollector(),
	})
	require.NoError(t, err)
	defer client.Close()

	tx := testLoadTx(t)
	tx.Scenario.Operation = "transfer"
	require.NoError(t, client.Send(t.Context(), tx))

	var collected metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &collected))

	recorded := map[string]int{}
	for _, scope := range collected.ScopeMetrics {
		for _, m := range scope.Metrics {
			if m.Name != "send_latency" && m.Name != "txs_accepted" && m.Name != "txs_rejected" {
				continue
			}
			for _, attrs := range dataPointAttributes(m.Data) {
				for _, key := range []string{"scenario", "operation"} {
					value, found := attrs.Value(attribute.Key(key))
					require.True(t, found, "%s carries no %s", m.Name, key)
					require.NotEmpty(t, value.AsString(), "%s carries an empty %s", m.Name, key)
				}
				recorded[m.Name]++
			}
		}
	}
	require.Positive(t, recorded["send_latency"])
	require.Positive(t, recorded["txs_accepted"])
}

// dataPointAttributes returns the attribute set of every data point in an
// aggregation, across the shapes the send path emits.
func dataPointAttributes(data metricdata.Aggregation) []attribute.Set {
	var sets []attribute.Set
	switch a := data.(type) {
	case metricdata.Histogram[float64]:
		for _, dp := range a.DataPoints {
			sets = append(sets, dp.Attributes)
		}
	case metricdata.Sum[int64]:
		for _, dp := range a.DataPoints {
			sets = append(sets, dp.Attributes)
		}
	}
	return sets
}
