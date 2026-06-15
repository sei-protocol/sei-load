package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Settings holds all CLI-configurable parameters
type Settings struct {
	TasksPerEndpoint int      `json:"workers,omitempty"`
	TPS              float64  `json:"tps,omitempty"`
	StatsInterval    Duration `json:"statsInterval,omitempty"`
	// InclusionReapAfter bounds how long an un-included tx stays in the inclusion
	// registry before it is reaped as expired. Tune to expected inclusion time on
	// congested chains: too short reaps slow inclusions as expired (inflated
	// un-included), too long inflates the in-flight map.
	InclusionReapAfter    Duration `json:"inclusionReapAfter,omitempty"`
	BufferSize            int      `json:"bufferSize,omitempty"`
	DryRun                bool     `json:"dryRun,omitempty"`
	Debug                 bool     `json:"debug,omitempty"`
	TrackReceipts         bool     `json:"trackReceipts,omitempty"`
	TrackBlocks           bool     `json:"trackBlocks,omitempty"`
	TrackUserLatency      bool     `json:"trackUserLatency,omitempty"`
	Prewarm               bool     `json:"prewarm,omitempty"`
	RampUp                bool     `json:"rampUp,omitempty"`
	ReportPath            string   `json:"reportPath,omitempty"`
	TxsDir                string   `json:"txsDir,omitempty"`
	TargetGas             uint64   `json:"targetGas,omitempty"`
	NumBlocksToWrite      int      `json:"numBlocksToWrite,omitempty"`
	PostSummaryFlushDelay Duration `json:"postSummaryFlushDelay,omitempty"`
	// ArrivalModel selects the transaction arrival model: "open_loop" schedules
	// tx i at t₀ + i/λ independent of sender availability (the
	// coordinated-omission fix), "closed_loop" (default) keeps the legacy
	// generate-then-send lockstep as the regression baseline.
	ArrivalModel string `json:"arrivalModel,omitempty"`
	// MaxInFlight bounds concurrent in-flight sends in the open-loop model;
	// txs that would exceed it at their scheduled instant are dropped and
	// counted rather than throttling the arrival clock. Ignored in closed-loop.
	MaxInFlight int `json:"maxInFlight,omitempty"`
	// ScheduleLagVoidThreshold is the fraction of the arrival interval (1/λ) that
	// schedule_lag_p99 may reach before an open-loop run is VOID. Zero
	// uses the provisional built-in default; set via config to retune without a
	// rebuild. Ignored in closed-loop.
	ScheduleLagVoidThreshold float64 `json:"scheduleLagVoidThreshold,omitempty"`
}

// Arrival model identifiers for the ArrivalModel setting.
const (
	ArrivalModelClosedLoop = "closed_loop"
	ArrivalModelOpenLoop   = "open_loop"
)

// Validate checks resolved settings for self-consistent run configuration,
// failing fast on combinations that would otherwise produce a silently
// degenerate run. Call once after ResolveSettings.
func (s Settings) Validate() error {
	switch s.ArrivalModel {
	case ArrivalModelClosedLoop, ArrivalModelOpenLoop:
	default:
		return fmt.Errorf("invalid arrival-model %q: must be %q or %q",
			s.ArrivalModel, ArrivalModelOpenLoop, ArrivalModelClosedLoop)
	}

	// Open-loop derives the inter-arrival gap as 1/λ. With no finite positive
	// arrival rate, λ is rate.Inf, the gap collapses to 0, IntendedSendTime
	// never advances past t₀, and the scheduler spins and drops everything —
	// the latency anchor degenerates to "time since campaign start". A finite λ
	// comes from either a configured TPS>0 or a ramp curve (RampUp), which the
	// ramper drives to finite limits. Reject the degenerate case up front.
	if s.ArrivalModel == ArrivalModelOpenLoop && s.TPS <= 0 && !s.RampUp {
		return fmt.Errorf("arrival-model %q requires a finite positive arrival rate: set --tps>0 or --ramp-up", ArrivalModelOpenLoop)
	}
	return nil
}

// DefaultSettings returns the default configuration values
func DefaultSettings() Settings {
	return Settings{
		TasksPerEndpoint:      1,
		TPS:                   0.0,
		StatsInterval:         Duration(10 * time.Second),
		InclusionReapAfter:    Duration(30 * time.Second),
		BufferSize:            1000,
		DryRun:                false,
		Debug:                 false,
		TrackReceipts:         false,
		TrackBlocks:           false,
		TrackUserLatency:      false,
		Prewarm:               false,
		RampUp:                false,
		ReportPath:            "",
		TxsDir:                "",
		TargetGas:             10_000_000,
		NumBlocksToWrite:      100,
		PostSummaryFlushDelay: Duration(25 * time.Second),
		ArrivalModel:          ArrivalModelClosedLoop,
		MaxInFlight:              10_000,
		ScheduleLagVoidThreshold: 0,
	}
}

// InitializeViper sets up Viper with CLI flags and defaults
func InitializeViper(cmd *cobra.Command) error {
	// Bind flags to viper with error checking
	flagBindings := map[string]string{
		"statsInterval":         "stats-interval",
		"inclusionReapAfter":    "inclusion-reap-after",
		"bufferSize":            "buffer-size",
		"tps":                   "tps",
		"dryRun":                "dry-run",
		"debug":                 "debug",
		"trackReceipts":         "track-receipts",
		"trackBlocks":           "track-blocks",
		"prewarm":               "prewarm",
		"trackUserLatency":      "track-user-latency",
		"workers":               "workers",
		"rampUp":                "ramp-up",
		"reportPath":            "report-path",
		"txsDir":                "txs-dir",
		"targetGas":             "target-gas",
		"numBlocksToWrite":      "num-blocks-to-write",
		"postSummaryFlushDelay": "post-summary-flush-delay",
		"arrivalModel":          "arrival-model",
		"maxInFlight":           "max-in-flight",
	}

	for viperKey, flagName := range flagBindings {
		if err := viper.BindPFlag(viperKey, cmd.Flags().Lookup(flagName)); err != nil {
			return fmt.Errorf("failed to bind flag %s: %w", flagName, err)
		}
	}

	// Set defaults in Viper
	defaults := DefaultSettings()
	viper.SetDefault("statsInterval", defaults.StatsInterval.ToDuration())
	viper.SetDefault("inclusionReapAfter", defaults.InclusionReapAfter.ToDuration())
	viper.SetDefault("bufferSize", defaults.BufferSize)
	viper.SetDefault("tps", defaults.TPS)
	viper.SetDefault("dryRun", defaults.DryRun)
	viper.SetDefault("debug", defaults.Debug)
	viper.SetDefault("trackReceipts", defaults.TrackReceipts)
	viper.SetDefault("trackBlocks", defaults.TrackBlocks)
	viper.SetDefault("prewarm", defaults.Prewarm)
	viper.SetDefault("trackUserLatency", defaults.TrackUserLatency)
	viper.SetDefault("workers", defaults.TasksPerEndpoint)
	viper.SetDefault("rampUp", defaults.RampUp)
	viper.SetDefault("reportPath", defaults.ReportPath)
	viper.SetDefault("txsDir", defaults.TxsDir)
	viper.SetDefault("targetGas", defaults.TargetGas)
	viper.SetDefault("numBlocksToWrite", defaults.NumBlocksToWrite)
	viper.SetDefault("postSummaryFlushDelay", defaults.PostSummaryFlushDelay.ToDuration())
	viper.SetDefault("arrivalModel", defaults.ArrivalModel)
	viper.SetDefault("maxInFlight", defaults.MaxInFlight)
	viper.SetDefault("scheduleLagVoidThreshold", defaults.ScheduleLagVoidThreshold)
	return nil
}

// LoadSettings reads and merges the config file into Viper
func LoadSettings(settings *Settings) error {
	if settings == nil {
		return fmt.Errorf("config settings are required")
	}

	// settings converted to JSON bytes
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	viper.SetConfigType("json")
	if err := viper.ReadConfig(bytes.NewBuffer(settingsJSON)); err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	return nil
}

// ResolveSettings gets the final resolved settings from Viper
func ResolveSettings() *Settings {
	return &Settings{
		TasksPerEndpoint:      viper.GetInt("workers"),
		TPS:                   viper.GetFloat64("tps"),
		StatsInterval:         Duration(viper.GetDuration("statsInterval")),
		InclusionReapAfter:    Duration(viper.GetDuration("inclusionReapAfter")),
		BufferSize:            viper.GetInt("bufferSize"),
		DryRun:                viper.GetBool("dryRun"),
		Debug:                 viper.GetBool("debug"),
		TrackReceipts:         viper.GetBool("trackReceipts"),
		TrackBlocks:           viper.GetBool("trackBlocks"),
		TrackUserLatency:      viper.GetBool("trackUserLatency"),
		Prewarm:               viper.GetBool("prewarm"),
		RampUp:                viper.GetBool("rampUp"),
		ReportPath:            viper.GetString("reportPath"),
		TxsDir:                viper.GetString("txsDir"),
		TargetGas:             viper.GetUint64("targetGas"),
		NumBlocksToWrite:      viper.GetInt("numBlocksToWrite"),
		PostSummaryFlushDelay: Duration(viper.GetDuration("postSummaryFlushDelay")),
		ArrivalModel:             viper.GetString("arrivalModel"),
		MaxInFlight:              viper.GetInt("maxInFlight"),
		ScheduleLagVoidThreshold: viper.GetFloat64("scheduleLagVoidThreshold"),
	}
}
