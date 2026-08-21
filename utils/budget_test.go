package utils_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/sei-protocol/sei-load/utils"
)

// TestWithinBudgetHidesItsOwnDeadline: an expiry of the budget must not reach
// the caller as a context sentinel. main reads those as a clean exit, so an
// internal timeout that surfaced one would report a run that did nothing as a
// success.
func TestWithinBudgetHidesItsOwnDeadline(t *testing.T) {
	t.Parallel()

	err := utils.WithinBudget(context.Background(), time.Millisecond, "work", func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})
	require.Error(t, err)
	require.NotErrorIs(t, err, context.DeadlineExceeded)
	require.Contains(t, err.Error(), "work exceeded its 1ms budget")
}

// TestWithinBudgetPassesParentCancellation: a sentinel from the caller's context
// is a real shutdown and must reach main unchanged, or a signalled run would
// report failure.
func TestWithinBudgetPassesParentCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := utils.WithinBudget(ctx, time.Hour, "work", func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})
	require.ErrorIs(t, err, context.Canceled)
}

// TestWithinBudgetLeavesOtherErrorsAlone: only a budget expiry is reworded, so a
// real failure keeps its chain and stays matchable with errors.Is.
func TestWithinBudgetLeavesOtherErrorsAlone(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("dial tcp: refused")
	err := utils.WithinBudget(context.Background(), time.Hour, "work", func(context.Context) error {
		return sentinel
	})
	require.ErrorIs(t, err, sentinel)

	require.NoError(t, utils.WithinBudget(context.Background(), time.Hour, "work",
		func(context.Context) error { return nil }))
}
