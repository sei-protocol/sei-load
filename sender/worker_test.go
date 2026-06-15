package sender

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"

	"github.com/sei-protocol/sei-load/stats"
	"github.com/sei-protocol/sei-load/types"
)

// drainWorkerWithLimiter runs runTxSender (DryRun: no RPC) over txCount queued
// txs gated by a tight limiter and returns how long the drain took. cancel fires
// once all txs are recorded, so the elapsed time reflects limiter pacing alone.
func drainWorkerWithLimiter(t *testing.T, skipRateLimit bool, txCount int, rps float64) time.Duration {
	t.Helper()
	collector := stats.NewCollector()
	w := NewWorker(&WorkerConfig{
		ID:            0,
		Endpoint:      "dryrun",
		BufferSize:    txCount,
		Tasks:         1,
		DryRun:        true,
		Collector:     collector,
		Limiter:       rate.NewLimiter(rate.Limit(rps), 1),
		SkipRateLimit: skipRateLimit,
	})
	for range txCount {
		w.txChan <- &types.LoadTx{Scenario: &types.TxScenario{Name: "gate"}}
	}

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	go func() {
		for collector.GetStats().TotalTxs < uint64(txCount) {
			time.Sleep(time.Millisecond)
		}
		cancel()
	}()

	start := time.Now()
	_ = w.runTxSender(ctx, nil) // DryRun never touches the client
	return time.Since(start)
}

// TestRunTxSender_RateLimitedByDefault is the SkipRateLimit-flip guard: the
// zero-value config (SkipRateLimit=false) with a non-nil Limiter must gate, so
// an omitted flag can never silently drop rate limiting. With burst=1 at `rps`,
// draining txCount txs cannot finish faster than (txCount-1)/rps.
func TestRunTxSender_RateLimitedByDefault(t *testing.T) {
	const txCount = 10
	const rps = 50.0 // floor: (10-1)/50 = 180ms
	elapsed := drainWorkerWithLimiter(t, false, txCount, rps)
	require.GreaterOrEqual(t, elapsed, 150*time.Millisecond,
		"default config must rate-limit (safe zero value)")
}

// TestRunTxSender_SkipRateLimitBypassesLimiter confirms the open-loop opt-out:
// SkipRateLimit=true ignores the limiter entirely, so the same drain finishes
// far under the gated floor.
func TestRunTxSender_SkipRateLimitBypassesLimiter(t *testing.T) {
	const txCount = 10
	const rps = 50.0
	elapsed := drainWorkerWithLimiter(t, true, txCount, rps)
	require.Less(t, elapsed, 100*time.Millisecond,
		"SkipRateLimit must bypass the limiter")
}

func TestNewHttpTransport_Defaults(t *testing.T) {
	tr := newHttpTransport()

	require.Equal(t, 500, tr.MaxIdleConns)
	require.Equal(t, 50, tr.MaxIdleConnsPerHost)
	require.Equal(t, 90*time.Second, tr.IdleConnTimeout)
	require.False(t, tr.DisableKeepAlives)
}

func TestNewHttpTransport_WithMaxIdleConns(t *testing.T) {
	tr := newHttpTransport(WithMaxIdleConns(2048))

	require.Equal(t, 2048, tr.MaxIdleConns)
	require.Equal(t, 50, tr.MaxIdleConnsPerHost, "per-host default preserved")
}

func TestNewHttpTransport_WithMaxIdleConnsPerHost(t *testing.T) {
	tr := newHttpTransport(WithMaxIdleConnsPerHost(1024))

	require.Equal(t, 1024, tr.MaxIdleConnsPerHost)
	require.Equal(t, 500, tr.MaxIdleConns, "global default preserved")
}

func TestNewHttpTransport_MultipleOptions(t *testing.T) {
	tr := newHttpTransport(
		WithMaxIdleConns(4096),
		WithMaxIdleConnsPerHost(1024),
	)

	require.Equal(t, 4096, tr.MaxIdleConns)
	require.Equal(t, 1024, tr.MaxIdleConnsPerHost)
}

func TestNewHttpClient_Smoke(t *testing.T) {
	c := newHttpClient()
	require.Equal(t, 30*time.Second, c.Timeout)
	require.NotNil(t, c.Transport, "Transport must be set")
	_, isBareTransport := c.Transport.(*http.Transport)
	require.False(t, isBareTransport, "Transport should be wrapped by otelhttp, not bare *http.Transport")
}

func TestNewRPCClient_HTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client, err := newRPCClient(context.Background(), srv.URL)
	require.NoError(t, err)
	require.NotNil(t, client)
	client.Close()
}

func TestNewRPCClient_WS(t *testing.T) {
	srv := rpc.NewServer()
	ts := httptest.NewServer(srv.WebsocketHandler([]string{"*"}))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	client, err := newRPCClient(context.Background(), wsURL)
	require.NoError(t, err)
	require.NotNil(t, client)
	client.Close()
}

func TestNewRPCClient_UnsupportedScheme(t *testing.T) {
	client, err := newRPCClient(context.Background(), "ftp://example.com")
	require.Error(t, err)
	require.Nil(t, client)
}
