package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestLoadConfigParsesStrictly asserts that the run entrypoint rejects a key the
// config does not define. config.TestParseLoadConfigRejectsUnknownKey covers
// which keys and which nesting levels; this covers only that loadConfig routes
// through that parser rather than a lenient unmarshal.
func TestLoadConfigParsesStrictly(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "profile.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"chainId":1,"endpoints":["http://127.0.0.1:8545"],
		"scenarios":[{"name":"EVMTransfer","weight":1}],
		"settings":{"tps":10,"workers":50}}`), 0o600))

	_, err := loadConfig(path)
	require.ErrorContains(t, err, `"workers"`)
}

// TestEndedOnRunContext: a signalled or duration-bounded run exits clean, and a
// real error does not. A signalled run leaves the outer context uncancelled —
// cobra runs uncancelled and the handler reads the signal off a channel — so the
// context error arrives from a background task that scope cancelled on the way
// out, and the predicate has to recognise it without consulting ctx.
func TestEndedOnRunContext(t *testing.T) {
	t.Parallel()

	live := context.Background()
	// What a signalled shutdown actually produces: scope cancels its own context
	// and a background task reports it, while the outer context stays live.
	backgroundTaskCanceled := fmt.Errorf("sender: %w", context.Canceled)

	cases := map[string]struct {
		err  error
		want bool
	}{
		"operator signalled, outer context still live": {backgroundTaskCanceled, true},
		"run duration elapsed":                         {context.DeadlineExceeded, true},
		"real error":                                   {errors.New("dial tcp: refused"), false},
		"no error":                                     {nil, false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, endedOnRunContext(live, tc.err))
		})
	}
}
