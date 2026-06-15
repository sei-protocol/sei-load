package sender

import (
	"context"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/sei-protocol/sei-load/stats"
	"github.com/sei-protocol/sei-load/types"
	"github.com/sei-protocol/sei-load/utils"
	"github.com/sei-protocol/sei-load/utils/scope"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestEthClientSendTx_HTTP(t *testing.T) {
	tel := setupTelemetry(t)
	api := &mockEthAPI{}
	srv := rpc.NewServer()
	require.NoError(t, srv.RegisterName("eth", api))

	var traceparent string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceparent = r.Header.Get("traceparent")
		srv.ServeHTTP(w, r)
	}))
	defer ts.Close()

	tx := testLoadTx(t)
	client := newEthClient(&ethClientConfig{
		ChainID:   "test-chain",
		ID:        7,
		Endpoint:  ts.URL,
		Tasks:     1,
		Collector: stats.NewCollector(),
	})

	err := exerciseClientViaSend(t, client, tx, func() bool {
		return api.CallCount() == 1
	})
	require.NoError(t, err)
	require.NotEmpty(t, traceparent, "otelhttp transport should inject traceparent")
	require.Equal(t, 1, api.CallCount())
	require.NotEmpty(t, api.RawTransactions())

	rm := tel.Collect(t)
	requireHistogramCount(t, rm, "send_latency", map[string]string{
		"scenario": "test-scenario",
		"endpoint": ts.URL,
		"chain_id": "test-chain",
		"status":   "success",
	}, 1)
	requireSumValue(t, rm, "txs_accepted", map[string]string{
		"endpoint": ts.URL,
		"scenario": "test-scenario",
	}, 1)
}

func TestEthClientSendTx_WS(t *testing.T) {
	tel := setupTelemetry(t)
	api := &mockEthAPI{}
	srv := rpc.NewServer()
	require.NoError(t, srv.RegisterName("eth", api))

	ts := httptest.NewServer(srv.WebsocketHandler([]string{"*"}))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	tx := testLoadTx(t)
	client := newEthClient(&ethClientConfig{
		ChainID:   "test-chain",
		ID:        8,
		Endpoint:  wsURL,
		Tasks:     1,
		Collector: stats.NewCollector(),
	})

	err := exerciseClientViaSend(t, client, tx, func() bool {
		return api.CallCount() == 1
	})
	require.NoError(t, err)
	require.Equal(t, 1, api.CallCount())
	require.NotEmpty(t, api.RawTransactions())

	rm := tel.Collect(t)
	requireHistogramCount(t, rm, "send_latency", map[string]string{
		"scenario": "test-scenario",
		"endpoint": wsURL,
		"chain_id": "test-chain",
		"status":   "success",
	}, 1)
	requireSumValue(t, rm, "txs_accepted", map[string]string{
		"endpoint": wsURL,
		"scenario": "test-scenario",
	}, 1)
}

type mockEthAPI struct {
	mu     sync.Mutex
	rawTxs [][]byte
}

func (m *mockEthAPI) SendRawTransaction(_ context.Context, rawTx hexutil.Bytes) (common.Hash, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cp := make([]byte, len(rawTx))
	copy(cp, rawTx)
	m.rawTxs = append(m.rawTxs, cp)
	return common.HexToHash("0x1234"), nil
}

func (m *mockEthAPI) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.rawTxs)
}

func (m *mockEthAPI) RawTransactions() [][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([][]byte, len(m.rawTxs))
	for i, rawTx := range m.rawTxs {
		cp := make([]byte, len(rawTx))
		copy(cp, rawTx)
		out[i] = cp
	}
	return out
}

type testTelemetry struct {
	reader *sdkmetric.ManualReader
}

var (
	testTelemetryOnce sync.Once
	sharedTelemetry   testTelemetry
)

func setupTelemetry(t *testing.T) testTelemetry {
	t.Helper()

	testTelemetryOnce.Do(func() {
		reader := sdkmetric.NewManualReader()
		mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
		tp := sdktrace.NewTracerProvider()
		otel.SetMeterProvider(mp)
		otel.SetTracerProvider(tp)
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		))
		sharedTelemetry = testTelemetry{reader: reader}
	})
	return sharedTelemetry
}

func (tt testTelemetry) Collect(t *testing.T) metricdata.ResourceMetrics {
	t.Helper()

	var rm metricdata.ResourceMetrics
	require.NoError(t, tt.reader.Collect(t.Context(), &rm))
	return rm
}

func exerciseClientViaSend(t *testing.T, client *ethClient, tx *types.LoadTx, sent func() bool) error {
	t.Helper()

	var sendErr error
	err := scope.Run(t.Context(), func(ctx context.Context, s scope.Scope) error {
		s.SpawnBg(func() error {
			return utils.IgnoreAfterCancel(ctx, client.Run(ctx))
		})

		sendCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		sendErr = client.Send(sendCtx, tx)

		require.Eventually(t, func() bool {
			return client.cfg.Collector.GetStats().TotalTxs == 1 && sent()
		}, time.Second, 10*time.Millisecond)
		return nil
	})
	require.NoError(t, err)
	return sendErr
}

func testLoadTx(t *testing.T) *types.LoadTx {
	t.Helper()

	account, err := types.NewAccount()
	require.NoError(t, err)

	to := common.HexToAddress("0x0000000000000000000000000000000000000001")
	tx := ethtypes.NewTx(&ethtypes.LegacyTx{
		Nonce:    1,
		To:       &to,
		Value:    big.NewInt(1),
		Gas:      21_000,
		GasPrice: big.NewInt(1),
	})
	signedTx, err := ethtypes.SignTx(tx, ethtypes.LatestSignerForChainID(big.NewInt(1)), account.PrivKey)
	require.NoError(t, err)

	return types.CreateTxFromEthTx(signedTx, &types.TxScenario{
		Name:   "test-scenario",
		Sender: account,
	})
}

func requireHistogramCount(t *testing.T, rm metricdata.ResourceMetrics, name string, attrs map[string]string, minCount uint64) {
	t.Helper()

	for _, scopeMetric := range rm.ScopeMetrics {
		for _, metric := range scopeMetric.Metrics {
			if metric.Name != name {
				continue
			}
			hist, ok := metric.Data.(metricdata.Histogram[float64])
			require.True(t, ok, "metric %q should be a float64 histogram", name)
			for _, point := range hist.DataPoints {
				if attrsMatch(point.Attributes, attrs) && point.Count >= minCount {
					return
				}
			}
		}
	}
	t.Fatalf("metric %q did not contain attrs=%v with count >= %d", name, attrs, minCount)
}

func requireSumValue(t *testing.T, rm metricdata.ResourceMetrics, name string, attrs map[string]string, minValue int64) {
	t.Helper()

	for _, scopeMetric := range rm.ScopeMetrics {
		for _, metric := range scopeMetric.Metrics {
			if metric.Name != name {
				continue
			}
			sum, ok := metric.Data.(metricdata.Sum[int64])
			require.True(t, ok, "metric %q should be an int64 sum", name)
			for _, point := range sum.DataPoints {
				if attrsMatch(point.Attributes, attrs) && point.Value >= minValue {
					return
				}
			}
		}
	}
	t.Fatalf("metric %q did not contain attrs=%v with value >= %d", name, attrs, minValue)
}

func attrsMatch(set attribute.Set, want map[string]string) bool {
	for k, v := range want {
		got, ok := (&set).Value(attribute.Key(k))
		if !ok || got.Emit() != v {
			return false
		}
	}
	return true
}
