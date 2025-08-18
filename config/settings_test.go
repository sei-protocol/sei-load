package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestArgumentPrecedence(t *testing.T) {
	tests := []struct {
		name            string
		configContent   string
		cliArgs         []string
		expectedStats   time.Duration
		expectedWorkers int
		expectedTPS     float64
	}{
		{
			name: "config file only",
			configContent: `{
				"statsInterval": "5s",
				"workers": 3,
				"tps": 100.5
			}`,
			cliArgs:         []string{},
			expectedStats:   5 * time.Second,
			expectedWorkers: 3,
			expectedTPS:     100.5,
		},
		{
			name: "CLI overrides config",
			configContent: `{
				"statsInterval": "5s",
				"workers": 3,
				"tps": 100.5
			}`,
			cliArgs:         []string{"--stats-interval", "3s", "--workers", "7"},
			expectedStats:   3 * time.Second,
			expectedWorkers: 7,
			expectedTPS:     100.5, // Not overridden by CLI
		},
		{
			name: "defaults when neither CLI nor config",
			configContent: `{
				"endpoints": ["http://localhost:8545"]
			}`,
			cliArgs:         []string{},
			expectedStats:   10 * time.Second, // Default
			expectedWorkers: 1,                // Default
			expectedTPS:     0.0,              // Default
		},
		{
			name: "CLI overrides defaults",
			configContent: `{
				"endpoints": ["http://localhost:8545"]
			}`,
			cliArgs:         []string{"--stats-interval", "15s", "--tps", "50"},
			expectedStats:   15 * time.Second,
			expectedWorkers: 1,    // Default (not overridden)
			expectedTPS:     50.0, // CLI override
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset viper for each test
			viper.Reset()

			// Create temporary config file
			configFile := createTempConfigFile(t, tt.configContent)

			// Create test command with flags
			cmd := &cobra.Command{
				Use: "test",
			}

			// Add flags (with zero defaults to avoid precedence issues)
			cmd.Flags().Duration("stats-interval", 0, "Stats interval")
			cmd.Flags().Int("workers", 0, "Number of workers")
			cmd.Flags().Float64("tps", 0, "TPS")
			cmd.Flags().Bool("dry-run", false, "Dry run")
			cmd.Flags().Bool("debug", false, "Debug")
			cmd.Flags().Bool("track-receipts", false, "Track receipts")
			cmd.Flags().Bool("track-blocks", false, "Track blocks")
			cmd.Flags().Bool("prewarm", false, "Prewarm")
			cmd.Flags().Bool("track-user-latency", false, "Track user latency")
			cmd.Flags().Int("buffer-size", 0, "Buffer size")
			cmd.Flags().Bool("ramp-up", false, "Ramp up loadtest")

			// Parse CLI args
			if len(tt.cliArgs) > 0 {
				cmd.SetArgs(tt.cliArgs)
				require.NoError(t, cmd.Execute(), "Failed to parse CLI args")
			}

			// Initialize Viper
			require.NoError(t, InitializeViper(cmd), "Failed to initialize Viper")

			// Load config file
			require.NoError(t, LoadConfigFile(configFile), "Failed to load config file")

			// Resolve settings
			settings := ResolveSettings()

			// Verify expectations
			require.Equal(t, tt.expectedStats, settings.StatsInterval, "StatsInterval: expected %v, got %v", tt.expectedStats, settings.StatsInterval)
			require.Equal(t, tt.expectedWorkers, settings.Workers, "Workers: expected %d, got %d", tt.expectedWorkers, settings.Workers)
			require.Equal(t, tt.expectedTPS, settings.TPS, "TPS: expected %f, got %f", tt.expectedTPS, settings.TPS)
		})
	}
}

func TestDefaultSettings(t *testing.T) {
	defaults := DefaultSettings()

	expected := Settings{
		Workers:          1,
		TPS:              0.0,
		StatsInterval:    10 * time.Second,
		BufferSize:       1000,
		DryRun:           false,
		Debug:            false,
		TrackReceipts:    false,
		TrackBlocks:      false,
		TrackUserLatency: false,
		Prewarm:          false,
		RampUp:           false,
	}

	if defaults != expected {
		t.Errorf("DefaultSettings mismatch.\nExpected: %+v\nGot: %+v", expected, defaults)
	}
}

// Helper function to create temporary config files for testing
func createTempConfigFile(t *testing.T, content string) string {
	t.Helper()
	destination := filepath.Join(t.TempDir(), "test-config.json")
	err := os.WriteFile(destination, []byte(content), 0644)
	require.NoError(t, err, "Failed to create temp config file: %v", err)
	return destination
}
