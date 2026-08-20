package main

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestEndedOnRunContext: only the run context's own expiry turns a context error
// into a clean exit. A deployment that timed out carries the same sentinel and
// must still surface, or a run that deployed nothing reports success.
func TestEndedOnRunContext(t *testing.T) {
	t.Parallel()

	live := context.Background()
	expired, cancel := context.WithTimeout(context.Background(), -time.Second)
	defer cancel()
	canceled, cancelIt := context.WithCancel(context.Background())
	cancelIt()

	// How a deploy timeout reaches the caller: wrapped with %w through three
	// layers, so errors.Is still matches the sentinel.
	deployTimeout := fmt.Errorf("failed to create generator: %w",
		fmt.Errorf("failed to deploy scenarios: %w",
			fmt.Errorf("deploy StorageRW: %w", context.DeadlineExceeded)))

	cases := map[string]struct {
		ctx  context.Context
		err  error
		want bool
	}{
		"deploy timed out, run still live":  {live, deployTimeout, false},
		"run duration elapsed":              {expired, context.DeadlineExceeded, true},
		"operator signalled":                {canceled, context.Canceled, true},
		"real error, run still live":        {live, errors.New("dial tcp: refused"), false},
		"real error while run was ending":   {expired, errors.New("dial tcp: refused"), false},
		"no error at all":                   {live, nil, false},
		"deploy timed out as the run ended": {expired, deployTimeout, true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, endedOnRunContext(tc.ctx, tc.err))
		})
	}
}
