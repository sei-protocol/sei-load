package stats

import (
	"time"
)

// MockBlockStats is a test implementation of BlockStatsProvider
type MockBlockStats struct {
	// Preset return values for methods
	blockStats       BlockStats
	windowBlockStats BlockStats
	percentileValues map[int]time.Duration
	resetCallCount   int
}

// NewMockBlockStats creates a new MockBlockStats instance
func NewMockBlockStats() *MockBlockStats {
	return &MockBlockStats{
		percentileValues: make(map[int]time.Duration),
	}
}

// Setter methods for configuring mock return values

// SetBlockStats sets the return value for GetBlockStats()
func (m *MockBlockStats) SetBlockStats(stats BlockStats) *MockBlockStats {
	m.blockStats = stats
	return m
}

// SetWindowBlockStats sets the return value for GetWindowBlockStats()
func (m *MockBlockStats) SetWindowBlockStats(stats BlockStats) *MockBlockStats {
	m.windowBlockStats = stats
	return m
}

// SetNextPercentile sets the return value for the next call to GetWindowBlockTimePercentile with the given percentile
func (m *MockBlockStats) SetNextPercentile(percentile int, duration time.Duration) *MockBlockStats {
	m.percentileValues[percentile] = duration
	return m
}

// SetPercentile is an alias for SetNextPercentile for more natural usage
func (m *MockBlockStats) SetPercentile(percentile int, duration time.Duration) *MockBlockStats {
	return m.SetNextPercentile(percentile, duration)
}

// Interface implementation methods

// GetBlockStats returns the preset block stats
func (m *MockBlockStats) GetBlockStats() BlockStats {
	return m.blockStats
}

// GetWindowBlockStats returns the preset window block stats
func (m *MockBlockStats) GetWindowBlockStats() BlockStats {
	return m.windowBlockStats
}

// GetWindowBlockTimePercentile returns the preset value for the given percentile
func (m *MockBlockStats) GetWindowBlockTimePercentile(percentile int) time.Duration {
	if duration, exists := m.percentileValues[percentile]; exists {
		// Remove the value after returning it to simulate "next call" behavior
		delete(m.percentileValues, percentile)
		return duration
	}
	return 0
}

// ResetWindowStats tracks the number of times it's called
func (m *MockBlockStats) ResetWindowStats() {
	m.resetCallCount++
}

// Test helper methods

// GetResetCallCount returns how many times ResetWindowStats was called
func (m *MockBlockStats) GetResetCallCount() int {
	return m.resetCallCount
}

// HasPendingPercentile checks if there's a pending value for the given percentile
func (m *MockBlockStats) HasPendingPercentile(percentile int) bool {
	_, exists := m.percentileValues[percentile]
	return exists
}
