package main

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

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
