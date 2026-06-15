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
)

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
