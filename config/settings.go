package config

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Settings holds all CLI-configurable parameters
type Settings struct {
	Workers          int           `json:"workers"`
	TPS              float64       `json:"tps"`
	StatsInterval    Duration `json:"statsInterval"`
	BufferSize       int           `json:"bufferSize"`
	DryRun           bool          `json:"dryRun"`
	Debug            bool          `json:"debug"`
	TrackReceipts    bool          `json:"trackReceipts"`
	TrackBlocks      bool          `json:"trackBlocks"`
	TrackUserLatency bool          `json:"trackUserLatency"`
	Prewarm          bool          `json:"prewarm"`
	RampUp           bool          `json:"rampUp"`
	ReportPath       string        `json:"reportPath"`
}

// DefaultSettings returns the default configuration values
func DefaultSettings() Settings {
	return Settings{
		Workers:          1,
		TPS:              0.0,
		StatsInterval:    Duration(10 * time.Second),
		BufferSize:       1000,
		DryRun:           false,
		Debug:            false,
		TrackReceipts:    false,
		TrackBlocks:      false,
		TrackUserLatency: false,
		Prewarm:          false,
		RampUp:           false,
		ReportPath:       "",
	}
}

// InitializeViper sets up Viper with CLI flags and defaults
func InitializeViper(cmd *cobra.Command) error {
	// Bind flags to viper with error checking
	flagBindings := map[string]string{
		"statsInterval":    "stats-interval",
		"bufferSize":       "buffer-size",
		"tps":              "tps",
		"dryRun":           "dry-run",
		"debug":            "debug",
		"trackReceipts":    "track-receipts",
		"trackBlocks":      "track-blocks",
		"prewarm":          "prewarm",
		"trackUserLatency": "track-user-latency",
		"workers":          "workers",
		"rampUp":           "ramp-up",
		"reportPath":       "report-path",
	}

	for viperKey, flagName := range flagBindings {
		if err := viper.BindPFlag(viperKey, cmd.Flags().Lookup(flagName)); err != nil {
			return fmt.Errorf("failed to bind flag %s: %w", flagName, err)
		}
	}

	// Set defaults in Viper
	defaults := DefaultSettings()
	viper.SetDefault("statsInterval", defaults.StatsInterval.ToDuration())
	viper.SetDefault("bufferSize", defaults.BufferSize)
	viper.SetDefault("tps", defaults.TPS)
	viper.SetDefault("dryRun", defaults.DryRun)
	viper.SetDefault("debug", defaults.Debug)
	viper.SetDefault("trackReceipts", defaults.TrackReceipts)
	viper.SetDefault("trackBlocks", defaults.TrackBlocks)
	viper.SetDefault("prewarm", defaults.Prewarm)
	viper.SetDefault("trackUserLatency", defaults.TrackUserLatency)
	viper.SetDefault("workers", defaults.Workers)
	viper.SetDefault("rampUp", defaults.RampUp)
	viper.SetDefault("reportPath", defaults.ReportPath)
	return nil
}

// LoadConfigFile reads and merges the config file into Viper
func LoadConfigFile(configFile string) error {
	if configFile == "" {
		return fmt.Errorf("config file path is required")
	}

	viper.SetConfigFile(configFile)
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file %s: %w", configFile, err)
	}

	return nil
}

// ResolveSettings gets the final resolved settings from Viper
func ResolveSettings() Settings {
	return Settings{
		Workers:          viper.GetInt("workers"),
		TPS:              viper.GetFloat64("tps"),
		StatsInterval:    Duration(viper.GetDuration("statsInterval")),
		BufferSize:       viper.GetInt("bufferSize"),
		DryRun:           viper.GetBool("dryRun"),
		Debug:            viper.GetBool("debug"),
		TrackReceipts:    viper.GetBool("trackReceipts"),
		TrackBlocks:      viper.GetBool("trackBlocks"),
		TrackUserLatency: viper.GetBool("trackUserLatency"),
		Prewarm:          viper.GetBool("prewarm"),
		RampUp:           viper.GetBool("rampUp"),
		ReportPath:       viper.GetString("reportPath"),
	}
}
