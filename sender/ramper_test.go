package sender

import (
	"context"
	"testing"
	"time"

	"github.com/sei-protocol/sei-load/stats"
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
	err := ramper.NewStep()
	if err != nil {
		t.Fatalf("NewStep failed: %v", err)
	}

	expectedTps := 50.0 // incrementTps * 1 (first step)
	if limiter.Limit() != rate.Limit(expectedTps) {
		t.Fatalf("Expected TPS %v after first step, got %v", expectedTps, limiter.Limit())
	}

	// Call NewStep again and verify increment
	err = ramper.NewStep()
	if err != nil {
		t.Fatalf("Second NewStep failed: %v", err)
	}

	expectedTps = 100.0 // incrementTps * 2 (second step)
	if limiter.Limit() != rate.Limit(expectedTps) {
		t.Fatalf("Expected TPS %v after second step, got %v", expectedTps, limiter.Limit())
	}
}

// func TestRamper_WatchSLO_ChannelBehavior(t *testing.T) {
// 	limiter := rate.NewLimiter(0, 1)
// 	cfg := &RamperConfig{
// 		IncrementTps: 50.0,
// 		LoadTime:     2 * time.Second,
// 		PauseTime:    1 * time.Second,
// 	}

// blockCollector := stats.NewBlockCollector("loadtest-local")
// ramper := NewRamper(cfg, blockCollector, limiter)

// 	// Set TPS below threshold first
// 	ramper.currentTps = 400.0

// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer cancel()

// 	// Start watching SLO
// 	sloChan := ramper.WatchSLO(ctx)

// 	// Give it a moment to start
// 	time.Sleep(100 * time.Millisecond)

// 	// Wait for channel signal
// 	select {
// 	case <-sloChan:
// 		// Expected behavior - SLO violation detected
// 	case <-time.After(3 * time.Second):
// 		t.Fatal("Expected SLO violation signal but timeout occurred")
// 	}
// }

func TestRamper_Run_StepProgression(t *testing.T) {
	limiter := rate.NewLimiter(0, 1)
	cfg := &RamperConfig{
		IncrementTps: 50.0,
		LoadTime:     2 * time.Second,
		PauseTime:    1 * time.Second,
	}

	blockCollector := stats.NewBlockCollector("loadtest-local")
	ramper := NewRamper(cfg, blockCollector, limiter)

	// Create a context with timeout to prevent infinite running
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	// Track the progression
	startTime := time.Now()

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

	// Cancel context to stop the run
	cancel()

	// Wait for completion
	select {
	case err := <-done:
		if err != context.Canceled {
			t.Fatalf("Expected context.Canceled error, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not complete within timeout")
	}

	// Verify total time is reasonable (should be around 6+ seconds for our test)
	totalTime := time.Since(startTime)
	if totalTime < 3*time.Second {
		t.Fatalf("Test completed too quickly (%v), expected at least 3 seconds", totalTime)
	}
}
