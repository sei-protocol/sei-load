package utils

import (
	"context"
	"fmt"
	"time"
)

// WithinBudget runs work under a deadline of its own, and reports an expiry of
// that deadline without a context sentinel in the error chain.
//
// Startup work bounds itself so a chain that accepts a transaction and never
// mines it cannot hold a run open. But main treats context.Canceled and
// context.DeadlineExceeded as a clean exit — that is how a signalled or
// duration-bounded run reports success — so an internal budget that surfaced its
// own sentinel would be read as a successful run that did nothing at all.
//
// A sentinel from the parent context passes through untouched, because that one
// really is a shutdown.
func WithinBudget(ctx context.Context, budget time.Duration, what string, work func(context.Context) error) error {
	within, cancel := context.WithTimeout(ctx, budget)
	defer cancel()

	err := work(within)
	if err != nil && ctx.Err() == nil && within.Err() != nil {
		return fmt.Errorf("%s exceeded its %s budget: %v", what, budget, err)
	}
	return err
}
