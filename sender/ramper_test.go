package sender

import (
	"context"
	"testing"
	"time"

	"github.com/sei-protocol/sei-load/stats"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func TestRamper_NewStep_LimiterUpdate(t *testing.T) {
	// Create a limiter with non-zero initial rate
	initialRate := rate.Limit(100.0)
	limiter := rate.NewLimiter(initialRate, 1)

	// Verify initial rate is set
	if limiter.Limit() != initialRate {
		t.Fatalf("Expected initial rate %v, got %v", initialRate, limiter.Limit())
	}

	// Create ramper config
	cfg := &RamperConfig{
		IncrementTps: 50.0,
		LoadTime:     2 * time.Second,
		PauseTime:    1 * time.Second,
	}

	blockCollector := stats.NewBlockCollector("loadtest-local")
	// Create ramper - should reset limiter to 0
	ramper := NewRamper(cfg, blockCollector, limiter)

	// Verify limiter was reset to 0
	if limiter.Limit() != 0 {
		t.Fatalf("Expected limiter to be reset to 0, got %v", limiter.Limit())
	}

	// Call NewStep and verify limit is updated correctly
	ramper.NewStep()
	expectedTps := 50.0 // incrementTps * 1 (first step)
	if limiter.Limit() != rate.Limit(expectedTps) {
		t.Fatalf("Expected TPS %v after first step, got %v", expectedTps, limiter.Limit())
	}

	// Call NewStep again and verify increment
	ramper.NewStep()
	expectedTps = 100.0 // incrementTps * 2 (second step)
	if limiter.Limit() != rate.Limit(expectedTps) {
		t.Fatalf("Expected TPS %v after second step, got %v", expectedTps, limiter.Limit())
	}
}

func TestRamper_WatchSLO_ChannelBehavior(t *testing.T) {

	limiter := rate.NewLimiter(0, 1)
	cfg := &RamperConfig{
		IncrementTps: 50.0,
		LoadTime:     2 * time.Second,
		PauseTime:    1 * time.Second,
	}

	mockBlockStats := stats.NewMockBlockStats()
	ramper := NewRamper(cfg, mockBlockStats, limiter)

	mockBlockStats.SetPercentile(90, 1100*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	sloChannel := ramper.WatchSLO(ctx)

	// Since we set 90th percentile to 1100ms (> 1s threshold), we should get an SLO violation
	select {
	case <-sloChannel:
		// This is expected - SLO violation detected due to 1100ms > 1s threshold
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected SLO violation but timeout occurred")
	}

	// Verify the channel is closed after violation
	_, ok := <-sloChannel
	require.False(t, ok, "expected channel to be closed after violation")
}

func TestRamper_Run_StepProgression(t *testing.T) {
	limiter := rate.NewLimiter(0, 1)
	cfg := &RamperConfig{
		IncrementTps: 50.0,
		LoadTime:     2 * time.Second,
		PauseTime:    1 * time.Second,
	}

	mockBlockStats := stats.NewMockBlockStats()
	mockBlockStats.SetPercentile(90, 500*time.Millisecond)
	ramper := NewRamper(cfg, mockBlockStats, limiter)

	// Create a context with timeout to prevent infinite running
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	// Run in goroutine so we can monitor behavior
	done := make(chan error, 1)
	go func() {
		done <- ramper.Run(ctx)
	}()

	// Wait a bit to let first step start
	time.Sleep(100 * time.Millisecond)

	// Check that first step started (TPS should be 50)
	if limiter.Limit() != rate.Limit(50.0) {
		t.Fatalf("Expected first step TPS 50, got %v", limiter.Limit())
	}

	// Wait for load time to pass (2s) plus some buffer
	time.Sleep(2000 * time.Millisecond)

	// Should be in pause phase (limiter set to 1	)
	if limiter.Limit() != rate.Limit(1) {
		t.Fatalf("Expected limiter to be 1 during pause, got %v", limiter.Limit())
	}

	// Wait for pause time (1s) to start next step
	time.Sleep(1000 * time.Millisecond)

	// Should be in second step (TPS should be 100)
	if limiter.Limit() != rate.Limit(100.0) {
		t.Fatalf("Expected second step TPS 100, got %v", limiter.Limit())
	}
	mockBlockStats.SetPercentile(90, 1100*time.Millisecond)

	time.Sleep(100 * time.Millisecond)

	// expect SLO violation - err from done channel
	select {
	case err := <-done:
		require.ErrorIs(t, err, ErrRampTestFailedSLO, "expected SLO violation")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected SLO violation but timeout occurred")
	}
}
