package stats

import (
	"testing"
	"time"
)

func TestNewMockBlockStats(t *testing.T) {
	mock := NewMockBlockStats()

	if mock == nil {
		t.Fatal("NewMockBlockStats should not return nil")
	}

	if mock.percentileValues == nil {
		t.Error("percentileValues map should be initialized")
	}

	if mock.resetCallCount != 0 {
		t.Error("resetCallCount should start at 0")
	}
}

func TestMockBlockStats_SetBlockStats(t *testing.T) {
	mock := NewMockBlockStats()
	expectedStats := BlockStats{
		MaxBlockNumber: 1000,
		P50BlockTime:   5 * time.Second,
		P99BlockTime:   10 * time.Second,
		MaxBlockTime:   15 * time.Second,
		P50GasUsed:     21000,
		P99GasUsed:     50000,
		MaxGasUsed:     100000,
		SampleCount:    100,
	}

	// Test fluent API
	result := mock.SetBlockStats(expectedStats)
	if result != mock {
		t.Error("SetBlockStats should return the same mock instance for chaining")
	}

	// Test that the stats are stored
	actualStats := mock.GetBlockStats()
	if actualStats != expectedStats {
		t.Errorf("GetBlockStats() = %+v, want %+v", actualStats, expectedStats)
	}
}

func TestMockBlockStats_SetWindowBlockStats(t *testing.T) {
	mock := NewMockBlockStats()
	expectedStats := BlockStats{
		MaxBlockNumber: 500,
		P50BlockTime:   3 * time.Second,
		SampleCount:    50,
	}

	// Test fluent API
	result := mock.SetWindowBlockStats(expectedStats)
	if result != mock {
		t.Error("SetWindowBlockStats should return the same mock instance for chaining")
	}

	// Test that the stats are stored
	actualStats := mock.GetWindowBlockStats()
	if actualStats != expectedStats {
		t.Errorf("GetWindowBlockStats() = %+v, want %+v", actualStats, expectedStats)
	}
}

func TestMockBlockStats_PercentileMethods(t *testing.T) {
	mock := NewMockBlockStats()

	// Test SetNextPercentile
	duration90 := 12 * time.Millisecond
	result := mock.SetPercentile(90, duration90)
	if result != mock {
		t.Error("SetNextPercentile should return the same mock instance for chaining")
	}

	// Test SetPercentile (alias)
	duration95 := 20 * time.Millisecond
	result = mock.SetPercentile(95, duration95)
	if result != mock {
		t.Error("SetPercentile should return the same mock instance for chaining")
	}

	// Test HasPendingPercentile
	if !mock.HasPendingPercentile(90) {
		t.Error("HasPendingPercentile(90) should return true after setting")
	}
	if !mock.HasPendingPercentile(95) {
		t.Error("HasPendingPercentile(95) should return true after setting")
	}
	if mock.HasPendingPercentile(99) {
		t.Error("HasPendingPercentile(99) should return false when not set")
	}

	// Test GetWindowBlockTimePercentile returns correct values
	actual90 := mock.GetWindowBlockTimePercentile(90)
	if actual90 != duration90 {
		t.Errorf("GetWindowBlockTimePercentile(90) = %v, want %v", actual90, duration90)
	}

	actual95 := mock.GetWindowBlockTimePercentile(95)
	if actual95 != duration95 {
		t.Errorf("GetWindowBlockTimePercentile(95) = %v, want %v", actual95, duration95)
	}

	// Test that values persist after being read (no longer cleared)
	if !mock.HasPendingPercentile(90) {
		t.Error("HasPendingPercentile(90) should still return true after reading value")
	}
	if !mock.HasPendingPercentile(95) {
		t.Error("HasPendingPercentile(95) should still return true after reading value")
	}

	// Second call should return the same value (not cleared)
	second90 := mock.GetWindowBlockTimePercentile(90)
	if second90 != duration90 {
		t.Errorf("Second call to GetWindowBlockTimePercentile(90) = %v, want %v", second90, duration90)
	}

	// Test unknown percentile returns 0
	unknown := mock.GetWindowBlockTimePercentile(99)
	if unknown != 0 {
		t.Errorf("GetWindowBlockTimePercentile(99) = %v, want 0 for unknown percentile", unknown)
	}
}

func TestMockBlockStats_ResetWindowStats(t *testing.T) {
	mock := NewMockBlockStats()

	// Initial count should be 0
	if mock.GetResetCallCount() != 0 {
		t.Errorf("Initial GetResetCallCount() = %d, want 0", mock.GetResetCallCount())
	}

	// Call ResetWindowStats multiple times
	mock.ResetWindowStats()
	if mock.GetResetCallCount() != 1 {
		t.Errorf("After 1 call, GetResetCallCount() = %d, want 1", mock.GetResetCallCount())
	}

	mock.ResetWindowStats()
	mock.ResetWindowStats()
	if mock.GetResetCallCount() != 3 {
		t.Errorf("After 3 calls, GetResetCallCount() = %d, want 3", mock.GetResetCallCount())
	}
}

func TestMockBlockStats_FluentChaining(t *testing.T) {
	mock := NewMockBlockStats()

	blockStats := BlockStats{MaxBlockNumber: 1000}
	windowStats := BlockStats{MaxBlockNumber: 500}

	// Test chaining multiple setter calls
	result := mock.
		SetBlockStats(blockStats).
		SetWindowBlockStats(windowStats).
		SetPercentile(50, 5*time.Millisecond).
		SetPercentile(90, 15*time.Millisecond)

	if result != mock {
		t.Error("Fluent chaining should return the same mock instance")
	}

	// Verify all values were set correctly
	if mock.GetBlockStats() != blockStats {
		t.Error("Block stats not set correctly through chaining")
	}
	if mock.GetWindowBlockStats() != windowStats {
		t.Error("Window block stats not set correctly through chaining")
	}
	if !mock.HasPendingPercentile(50) {
		t.Error("Percentile 50 not set correctly through chaining")
	}
	if !mock.HasPendingPercentile(90) {
		t.Error("Percentile 90 not set correctly through chaining")
	}
}

func TestMockBlockStats_BlockStatsProviderInterface(t *testing.T) {
	// Test that MockBlockStats implements BlockStatsProvider interface
	var provider BlockStatsProvider = NewMockBlockStats()

	// This should compile without errors if interface is properly implemented
	_ = provider.GetBlockStats()
	_ = provider.GetWindowBlockStats()
	_ = provider.GetWindowBlockTimePercentile(50)
	provider.ResetWindowStats()
}

func TestMockBlockStats_DefaultValues(t *testing.T) {
	mock := NewMockBlockStats()

	// Test default empty BlockStats
	blockStats := mock.GetBlockStats()
	expectedEmpty := BlockStats{}
	if blockStats != expectedEmpty {
		t.Errorf("Default GetBlockStats() = %+v, want empty BlockStats", blockStats)
	}

	windowStats := mock.GetWindowBlockStats()
	if windowStats != expectedEmpty {
		t.Errorf("Default GetWindowBlockStats() = %+v, want empty BlockStats", windowStats)
	}

	// Test default percentile value
	percentile := mock.GetWindowBlockTimePercentile(50)
	if percentile != 0 {
		t.Errorf("Default GetWindowBlockTimePercentile(50) = %v, want 0", percentile)
	}
}
