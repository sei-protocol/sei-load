package sender

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Tests target newHttpTransport (the inner *http.Transport factory) directly
// so they stay independent of the otelhttp wrapping that newHttpClient adds.
// The otelhttp wrap is validated separately in integration contexts because
// otelhttp.Transport does not expose its wrapped RoundTripper.

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

// Smoke-test the full newHttpClient: Timeout is set on the outer client, and
// the Transport exists (opaque otelhttp wrapper). Inner-transport field
// assertions belong in the TestNewHttpTransport_* tests above.
func TestNewHttpClient_Smoke(t *testing.T) {
	c := newHttpClient()
	require.Equal(t, 30*time.Second, c.Timeout)
	require.NotNil(t, c.Transport, "Transport must be set")
	_, isBareTransport := c.Transport.(*http.Transport)
	require.False(t, isBareTransport, "Transport should be wrapped by otelhttp, not bare *http.Transport")
}
