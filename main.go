package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sei-protocol/sei-load/utils/service"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/sei-protocol/sei-load/config"
	"github.com/sei-protocol/sei-load/generator"
	"github.com/sei-protocol/sei-load/sender"
	"github.com/sei-protocol/sei-load/stats"
	"github.com/sei-protocol/sei-load/utils"
)

var (
	configFile        string
	statsInterval     time.Duration
	bufferSize        int
	tps               float64
	dryRun            bool
	debug             bool
	workers           int
	trackReceipts     bool
	trackBlocks       bool
	prewarm           bool
	trackUserLatency  bool
)

// ResolvedSettings holds the final resolved settings after applying precedence
type ResolvedSettings struct {
	Workers          int
	TPS              float64
	StatsInterval    time.Duration
	BufferSize       int
	DryRun           bool
	Debug            bool
	TrackReceipts    bool
	TrackBlocks      bool
	TrackUserLatency bool
	Prewarm          bool
}

// resolveSettings applies precedence: CLI > Config > Default
func resolveSettings(cfg *config.LoadConfig, cmd *cobra.Command) ResolvedSettings {
	settings := ResolvedSettings{
		// Default values
		Workers:          1,
		TPS:              0,
		StatsInterval:    10 * time.Second,
		BufferSize:       1000,
		DryRun:           false,
		Debug:            false,
		TrackReceipts:    false,
		TrackBlocks:      false,
		TrackUserLatency: false,
		Prewarm:          false,
	}

	// Apply config values if present
	if cfg.Settings != nil {
		if cfg.Settings.Workers != nil {
			settings.Workers = *cfg.Settings.Workers
		}
		if cfg.Settings.TPS != nil {
			settings.TPS = *cfg.Settings.TPS
		}
		if cfg.Settings.StatsInterval != nil {
			settings.StatsInterval = *cfg.Settings.StatsInterval
		}
		if cfg.Settings.BufferSize != nil {
			settings.BufferSize = *cfg.Settings.BufferSize
		}
		if cfg.Settings.DryRun != nil {
			settings.DryRun = *cfg.Settings.DryRun
		}
		if cfg.Settings.Debug != nil {
			settings.Debug = *cfg.Settings.Debug
		}
		if cfg.Settings.TrackReceipts != nil {
			settings.TrackReceipts = *cfg.Settings.TrackReceipts
		}
		if cfg.Settings.TrackBlocks != nil {
			settings.TrackBlocks = *cfg.Settings.TrackBlocks
		}
		if cfg.Settings.TrackUserLatency != nil {
			settings.TrackUserLatency = *cfg.Settings.TrackUserLatency
		}
		if cfg.Settings.Prewarm != nil {
			settings.Prewarm = *cfg.Settings.Prewarm
		}
	}

	// Apply CLI values if explicitly set (CLI wins over config)
	if cmd.Flags().Changed("workers") {
		settings.Workers = workers
	}
	if cmd.Flags().Changed("tps") {
		settings.TPS = tps
	}
	if cmd.Flags().Changed("stats-interval") {
		settings.StatsInterval = statsInterval
	}
	if cmd.Flags().Changed("buffer-size") {
		settings.BufferSize = bufferSize
	}
	if cmd.Flags().Changed("dry-run") {
		settings.DryRun = dryRun
	}
	if cmd.Flags().Changed("debug") {
		settings.Debug = debug
	}
	if cmd.Flags().Changed("track-receipts") {
		settings.TrackReceipts = trackReceipts
	}
	if cmd.Flags().Changed("track-blocks") {
		settings.TrackBlocks = trackBlocks
	}
	if cmd.Flags().Changed("track-user-latency") {
		settings.TrackUserLatency = trackUserLatency
	}
	if cmd.Flags().Changed("prewarm") {
		settings.Prewarm = prewarm
	}

	return settings
}

var rootCmd = &cobra.Command{
	Use:   "seiload",
	Short: "Sei Chain Load Test v2",
	Long: `A load test generator for Sei Chain.

Supports both contract and non-contract scenarios with factory 
and weighted scenario selection mechanisms. Features sharded sending 
to multiple endpoints with account pooling management.

Use --dry-run to test configuration and view transaction details 
without actually sending requests or deploying contracts.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runLoadTest(context.Background(), cmd, args); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "Path to configuration file (required)")
	rootCmd.Flags().DurationVarP(&statsInterval, "stats-interval", "s", 10*time.Second, "Interval for logging statistics")
	rootCmd.Flags().IntVarP(&bufferSize, "buffer-size", "b", 1000, "Buffer size per worker")
	rootCmd.Flags().Float64VarP(&tps, "tps", "t", 0, "Transactions per second (0 = no limit)")
	rootCmd.Flags().BoolVarP(&dryRun, "dry-run", "", false, "Mock deployment and requests")
	rootCmd.Flags().BoolVarP(&debug, "debug", "", false, "Log each request")
	rootCmd.Flags().BoolVarP(&trackReceipts, "track-receipts", "", false, "Track receipts")
	rootCmd.Flags().BoolVarP(&trackBlocks, "track-blocks", "", false, "Track blocks")
	rootCmd.Flags().BoolVarP(&prewarm, "prewarm", "", false, "Prewarm accounts with self-transactions")
	rootCmd.Flags().BoolVarP(&trackUserLatency, "track-user-latency", "", false, "Track user latency")
	rootCmd.Flags().IntVarP(&workers, "workers", "w", 1, "Number of workers")

	if err := rootCmd.MarkFlagRequired("config"); err != nil {
		log.Fatal(err)
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		_, err := fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(1)
	}
}

func runLoadTest(ctx context.Context, cmd *cobra.Command, args []string) error {
	// Parse the config file into a config.LoadConfig struct
	cfg, err := loadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Resolve settings with precedence: CLI > Config > Default
	settings := resolveSettings(cfg, cmd)

	log.Printf("🚀 Starting Sei Chain Load Test v2")
	log.Printf("📁 Config file: %s", configFile)
	log.Printf("🎯 Endpoints: %d", len(cfg.Endpoints))
	log.Printf("👥 Workers per endpoint: %d", settings.Workers)
	log.Printf("🔧 Total workers: %d", len(cfg.Endpoints)*settings.Workers)
	log.Printf("📊 Scenarios: %d", len(cfg.Scenarios))
	log.Printf("⏱️  Stats interval: %v", settings.StatsInterval)
	log.Printf("📦 Buffer size per worker: %d", settings.BufferSize)
	if settings.TPS > 0 {
		log.Printf("📈 Transactions per second: %.2f", settings.TPS)
	}
	if settings.DryRun {
		log.Printf("📝 Dry run: enabled")
	}
	if settings.TrackReceipts {
		log.Printf("📝 Track receipts: enabled")
	}
	if settings.TrackBlocks {
		log.Printf("📝 Track blocks: enabled")
	}
	if settings.Prewarm {
		log.Printf("📝 Prewarm: enabled")
	}
	if settings.TrackUserLatency {
		log.Printf("📝 Track user latency: enabled")
	}
	log.Println()

	// Enable mock deployment in dry-run mode
	if settings.DryRun {
		cfg.MockDeploy = true
	}

	// Create statistics collector and logger
	collector := stats.NewCollector()
	logger := stats.NewLogger(collector, settings.StatsInterval, settings.Debug)

	err = service.Run(ctx, func(ctx context.Context, s service.Scope) error {
		// Create the generator from the config struct
		gen, err := generator.NewConfigBasedGenerator(cfg)
		if err != nil {
			return fmt.Errorf("failed to create generator: %w", err)
		}

		// Create the sender from the config struct
		snd, err := sender.NewShardedSender(cfg, settings.BufferSize, settings.Workers)
		if err != nil {
			return fmt.Errorf("failed to create sender: %w", err)
		}

		// Create and start block collector if endpoints are available
		var blockCollector *stats.BlockCollector
		if len(cfg.Endpoints) > 0 && settings.TrackBlocks {
			blockCollector = stats.NewBlockCollector()
			collector.SetBlockCollector(blockCollector)
			s.SpawnBgNamed("block collector", func() error {
				return blockCollector.Run(ctx, cfg.Endpoints[0])
			})
		}

		// Create and start user latency tracker if endpoints are available
		if len(cfg.Endpoints) > 0 && settings.TrackUserLatency {
			userLatencyTracker := stats.NewUserLatencyTracker(settings.StatsInterval)
			s.SpawnBgNamed("user latency tracker", func() error {
				return userLatencyTracker.Run(ctx, cfg.Endpoints[0])
			})
		}

		// Enable dry-run mode in sender if specified
		if settings.DryRun {
			snd.SetDryRun(true)
		}
		if settings.Debug {
			snd.SetDebug(true)
		}
		if settings.TrackReceipts {
			snd.SetTrackReceipts(true)
		}
		if settings.TrackBlocks {
			snd.SetTrackBlocks(true)
		}

		// Set statistics collector for sender and its workers
		snd.SetStatsCollector(collector, logger)

		// Create dispatcher
		dispatcher := sender.NewDispatcher(gen, snd)
		if settings.TPS > 0 {
			// Convert TPS to interval: 1/tps seconds = (1/tps) * 1e9 nanoseconds
			intervalNs := int64((1.0 / settings.TPS) * 1e9)
			dispatcher.SetRateLimit(time.Duration(intervalNs))
		}

		// Set statistics collector for dispatcher
		dispatcher.SetStatsCollector(collector)

		// Set up prewarming if enabled
		if settings.Prewarm {
			log.Printf("🔥 Creating prewarm generator...")
			prewarmGen := generator.NewPrewarmGenerator(cfg, gen)
			dispatcher.SetPrewarmGenerator(prewarmGen)
			log.Printf("✅ Prewarm generator ready")
			log.Printf("📝 Prewarm mode: Accounts will be prewarmed")
		}

		// Start the sender (starts all workers)
		s.SpawnBgNamed("sender", func() error { return snd.Run(ctx) })
		log.Printf("✅ Connected to %d endpoints", snd.GetNumShards())

		// Perform prewarming if enabled (before starting logger to avoid logging prewarm transactions)
		if settings.Prewarm {
			if err := dispatcher.Prewarm(ctx); err != nil {
				return fmt.Errorf("failed to prewarm accounts: %w", err)
			}
		}

		// Start logger (after prewarming to capture only main load test metrics)
		s.SpawnBgNamed("logger", func() error { return logger.Run(ctx) })
		log.Printf("✅ Started statistics logger")

		// Start dispatcher for main load test
		s.SpawnBgNamed("dispatcher", func() error { return dispatcher.Run(ctx) })
		log.Printf("✅ Started dispatcher")

		// Set up signal handling for graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		log.Printf("📈 Logging statistics every %v (Press Ctrl+C to stop)", settings.StatsInterval)
		if settings.DryRun {
			log.Printf("📝 Dry-run mode: Simulating requests without sending")
		}
		if settings.Debug {
			log.Printf("🐛 Debug mode: Each transaction will be logged")
		}
		if settings.TrackReceipts {
			log.Printf("📝 Track receipts mode: Receipts will be tracked")
		}
		if settings.TrackBlocks {
			log.Printf("📝 Track blocks mode: Block data will be collected")
		}
		if settings.TrackUserLatency {
			log.Printf("📝 Track user latency mode: User latency will be tracked")
		}
		log.Print(strings.Repeat("=", 60))

		// Main loop - wait for shutdown signal
		if _, err := utils.Recv(ctx, sigChan); err != nil {
			return err
		}
		log.Print("\n🛑 Received shutdown signal, stopping gracefully...")
		return nil
	})
	// Print final statistics
	logger.LogFinalStats()
	log.Printf("👋 Shutdown complete")
	return err
}

// loadConfig reads and parses the configuration file
func loadConfig(filename string) (*config.LoadConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg config.LoadConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config json: %w", err)
	}

	// Validate configuration
	if len(cfg.Endpoints) == 0 {
		return nil, fmt.Errorf("no endpoints specified in config")
	}

	if len(cfg.Scenarios) == 0 {
		return nil, fmt.Errorf("no scenarios specified in config")
	}

	return &cfg, nil
}
