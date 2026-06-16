package sender

import (
	"context"
	"crypto/sha256"
	"math/big"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
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
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestEthClientSendTx_HTTP(t *testing.T) {
	api := newMockEthAPI()
	srv := rpc.NewServer()
	require.NoError(t, srv.RegisterName("eth", api))

	// We check the TraceID as a proof that otel Transport was used.
	var traceparent string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceparent = r.Header.Get("traceparent")
		srv.ServeHTTP(w, r)
	}))
	defer ts.Close()
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	otel.SetTracerProvider(sdktrace.NewTracerProvider())
	ctx, span := otel.Tracer("sender-test").Start(t.Context(), "parent")
	defer span.End()

	tx := testLoadTx(t)
	client := newEthClient(&ethClientConfig{
		ChainID:   "test-chain",
		ID:        7,
		Endpoint:  ts.URL,
		Tasks:     1,
		Collector: stats.NewCollector(),
	})

	err := scope.Run(ctx, func(ctx context.Context, s scope.Scope) error {
		s.SpawnBg(func() error { return utils.IgnoreCancel(client.Run(ctx)) })
		return client.Send(ctx, tx)
	})
	require.NoError(t, err)
	require.Equal(t, [][]byte{tx.Payload}, api.RawTransactions())
	require.Contains(t, traceparent, span.SpanContext().TraceID().String())
}

func TestEthClientSendTx_WS(t *testing.T) {
	api := newMockEthAPI()
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

	err := scope.Run(t.Context(), func(ctx context.Context, s scope.Scope) error {
		s.SpawnBg(func() error { return utils.IgnoreCancel(client.Run(ctx)) })
		return client.Send(ctx, tx)
	})
	require.NoError(t, err)
	require.Equal(t, [][]byte{tx.Payload}, api.RawTransactions())
}

func TestEthClientRunSender_RegistersSuccessfulSendAfterOnComplete(t *testing.T) {
	tracker := stats.NewInclusionTracker("test-chain", time.Hour, 100, true)
	client := newEthClient(&ethClientConfig{
		ChainID:   "test-chain",
		ID:        9,
		Endpoint:  "dryrun",
		Tasks:     1,
		DryRun:    true,
		Collector: stats.NewCollector(),
		Inclusion: utils.Some(tracker),
	})

	tx := testLoadTx(t)
	var inflightAtComplete atomic.Uint64
	tx.OnComplete = func(error) {
		inflightAtComplete.Store(tracker.Summary().InflightAtShutdown)
	}

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() { errCh <- client.runSender(ctx, nil) }()

	require.NoError(t, client.Send(ctx, tx))
	cancel()
	require.ErrorIs(t, <-errCh, context.Canceled)
	require.Zero(t, inflightAtComplete.Load(), "inclusion must register after OnComplete")
	require.Equal(t, uint64(1), tracker.Summary().InflightAtShutdown, "successful send must register exactly once")
}

type mockEthAPI struct {
	rawTxs utils.Mutex[*[][]byte]
}

func newMockEthAPI() *mockEthAPI {
	rawTxs := [][]byte{}
	return &mockEthAPI{rawTxs: utils.NewMutex(&rawTxs)}
}

func (m *mockEthAPI) SendRawTransaction(_ context.Context, rawTx hexutil.Bytes) (common.Hash, error) {
	for rawTxs := range m.rawTxs.Lock() {
		*rawTxs = append(*rawTxs, slices.Clone(rawTx))
	}
	sum := sha256.Sum256(rawTx)
	return common.BytesToHash(sum[:]), nil
}

func (m *mockEthAPI) RawTransactions() [][]byte {
	for rawTxs := range m.rawTxs.Lock() {
		return slices.Clone(*rawTxs)
	}
	panic("unreachable")
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
