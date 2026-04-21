package sender

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func transport(t *testing.T, c *http.Client) *http.Transport {
	t.Helper()
	tr, ok := c.Transport.(*http.Transport)
	require.True(t, ok, "expected *http.Transport, got %T", c.Transport)
	return tr
}

func TestNewHttpClient_Defaults(t *testing.T) {
	c := newHttpClient()
	tr := transport(t, c)

	require.Equal(t, 500, tr.MaxIdleConns)
	require.Equal(t, 50, tr.MaxIdleConnsPerHost)
	require.Equal(t, 30*time.Second, c.Timeout)
	require.Equal(t, 90*time.Second, tr.IdleConnTimeout)
	require.False(t, tr.DisableKeepAlives)
}

func TestNewHttpClient_WithMaxIdleConns(t *testing.T) {
	c := newHttpClient(WithMaxIdleConns(2048))
	tr := transport(t, c)

	require.Equal(t, 2048, tr.MaxIdleConns)
	require.Equal(t, 50, tr.MaxIdleConnsPerHost, "per-host default preserved")
}

func TestNewHttpClient_WithMaxIdleConnsPerHost(t *testing.T) {
	c := newHttpClient(WithMaxIdleConnsPerHost(1024))
	tr := transport(t, c)

	require.Equal(t, 1024, tr.MaxIdleConnsPerHost)
	require.Equal(t, 500, tr.MaxIdleConns, "global default preserved")
}

func TestNewHttpClient_MultipleOptions(t *testing.T) {
	c := newHttpClient(
		WithMaxIdleConns(4096),
		WithMaxIdleConnsPerHost(1024),
	)
	tr := transport(t, c)

	require.Equal(t, 4096, tr.MaxIdleConns)
	require.Equal(t, 1024, tr.MaxIdleConnsPerHost)
}
