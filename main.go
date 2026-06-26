package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel"
	"golang.org/x/time/rate"

	"github.com/sei-protocol/sei-load/config"
	"github.com/sei-protocol/sei-load/funder"
	"github.com/sei-protocol/sei-load/generator"
	"github.com/sei-protocol/sei-load/observability"
	"github.com/sei-protocol/sei-load/sender"
	"github.com/sei-protocol/sei-load/stats"
	"github.com/sei-protocol/sei-load/types"
	"github.com/sei-protocol/sei-load/utils"
	"github.com/sei-protocol/sei-load/utils/scope"
)

var (
	configFile string
)

var rootCmd = &cobra.Command{
	Use:   "seiload",
	Short: "Sei Chain Load Test v2",
	Long: `A load test generator for Sei Chain.

Supports both contract and non-contract scenarios with factory
and weighted scenario selection mechanisms. Features sharded sending
to multiple endpoints with account pooling management.

Use --dry-run to test configuration and view transaction details
without actually sending requests or deploying contracts.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLoadTest(cmd.Context(), cmd)
	},
}

func init() {
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "Path to configuration file (required)")
	rootCmd.Flags().DurationP("stats-interval", "s", 0, "Interval for logging statistics")
	rootCmd.Flags().Duration("inclusion-reap-after", 30*time.Second, "How long an un-included tx stays in the inclusion registry before reaping as expired (tune to expected inclusion time on congested chains)")
	rootCmd.Flags().IntP("buffer-size", "b", 0, "Buffer size per worker")
	rootCmd.Flags().Float64P("tps", "t", 0, "Transactions per second (0 = no limit)")
	rootCmd.Flags().Bool("dry-run", false, "Mock deployment and requests")
	rootCmd.Flags().Bool("debug", false, "Log each request")
	rootCmd.Flags().Bool("track-receipts", false, "Track receipts")
	rootCmd.Flags().Bool("track-blocks", false, "Track blocks")
	rootCmd.Flags().Bool("prewarm", false, "Prewarm accounts with self-transactions")
	rootCmd.Flags().Bool("track-user-latency", false, "Track user latency")
	rootCmd.Flags().IntP("workers", "w", 0, "Number of workers")
	rootCmd.Flags().IntP("nodes", "n", 0, "Number of nodes/endpoints to use (0 = use all)")
	rootCmd.Flags().String("metricsListenAddr", "0.0.0.0:9090", "The ip:port on which to export prometheus metrics.")
	rootCmd.Flags().Bool("ramp-up", false, "Ramp up loadtest")
	rootCmd.Flags().String("report-path", "", "Path to save the report")
	rootCmd.Flags().String("txs-dir", "", "Path to save the transactions")
	rootCmd.Flags().Uint64("target-gas", 10_000_000, "Target gas per block")
	rootCmd.Flags().Int("num-blocks-to-write", 100, "Number of blocks to write")
	rootCmd.Flags().Duration("post-summary-flush-delay", 25*time.Second, "In-process delay after run-summary metrics are recorded, allowing Prometheus to scrape them before exit")
	rootCmd.Flags().Duration("duration", 0, "Run duration (0 = until SIGTERM/SIGINT)")
	rootCmd.Flags().String("arrival-model", config.ArrivalModelClosedLoop, "Transaction arrival model: open_loop (schedule t0+i/lambda, drop on overrun) or closed_loop (legacy generate-then-send)")
	rootCmd.Flags().Int("max-in-flight", 10_000, "Open-loop only: max concurrent in-flight sends before overdue txs are dropped")

	// Initialize Viper with proper error handling
	if err := config.InitializeViper(rootCmd); err != nil {
		log.Fatalf("Failed to initialize configuration: %v", err)
	}

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

func runLoadTest(ctx context.Context, cmd *cobra.Command) error {
	// Parse the config file into a config.LoadConfig struct
	cfg, err := loadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Load settings into Viper
	if err := config.LoadSettings(cfg.Settings); err != nil {
		return fmt.Errorf("failed to load config file: %w", err)
	}

	// Get resolved settings from the config package
	cfg.Settings = config.ResolveSettings()
	if err := cfg.Settings.Validate(); err != nil {
		return fmt.Errorf("invalid settings: %w", err)
	}

	// Handle --nodes flag to limit number of endpoints
	nodes, _ := cmd.Flags().GetInt("nodes")
	if nodes > 0 && nodes < len(cfg.Endpoints) {
		log.Printf("🔧 Limiting endpoints from %d to %d nodes", len(cfg.Endpoints), nodes)
		cfg.Endpoints = cfg.Endpoints[:nodes]
	}
	// Enable mock deployment in dry-run mode
	if cfg.Settings.DryRun {
		cfg.MockDeploy = true
	}

	log.Printf("🚀 Starting Sei Chain Load Test v2")
	log.Printf("📁 Config file: %s", configFile)
	log.Printf("🎯 Endpoints: %d", len(cfg.Endpoints))
	log.Printf("👥 Tasks per endpoint: %d", cfg.Settings.TasksPerEndpoint)
	log.Printf("🔧 Total tasks: %d", len(cfg.Endpoints)*cfg.Settings.TasksPerEndpoint)
	log.Printf("📊 Scenarios: %d", len(cfg.Scenarios))
	log.Printf("⏱️  Stats interval: %v", cfg.Settings.StatsInterval.ToDuration())
	log.Printf("📦 Buffer size per worker: %d", cfg.Settings.BufferSize)
	if cfg.Settings.TPS > 0 {
		log.Printf("📈 Transactions per second: %.2f", cfg.Settings.TPS)
	}
	if cfg.Settings.DryRun {
		log.Printf("📝 Dry run: enabled")
	}
	if cfg.Settings.TrackReceipts {
		log.Printf("📝 Track receipts: enabled")
	}
	if cfg.Settings.TrackBlocks {
		log.Printf("📝 Track blocks: enabled")
	}
	if cfg.Settings.Prewarm {
		log.Printf("📝 Prewarm: enabled")
	}
	if cfg.Settings.TrackUserLatency {
		log.Printf("📝 Track user latency: enabled")
	}

	listenAddr := cmd.Flag("metricsListenAddr").Value.String()
	log.Printf("serving metrics at %s/metrics", listenAddr)

	obsShutdown, err := observability.Setup(ctx, observability.Config{
		RunScope:     observability.RunScopeFromEnv(),
		OTLPEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
	})
	if err != nil {
		return fmt.Errorf("observability setup: %w", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := obsShutdown(shutdownCtx); err != nil {
			log.Printf("observability shutdown: %v", err)
		}
	}()

	// EnableOpenMetrics is load-bearing: the default promhttp.Handler() strips
	// exemplars regardless of the scraper's Accept header.
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{EnableOpenMetrics: true},
	))
	metricsServer := &http.Server{
		Addr:              listenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("failed to serve metrics: %v", err)
		}
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := metricsServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("metrics server shutdown: %v", err)
		}
	}()

	if duration, _ := cmd.Flags().GetDuration("duration"); duration > 0 {
		log.Printf("⏰ Run duration: %s", duration)
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, duration)
		defer cancel()
	}

	ctx, runSpan := otel.Tracer("github.com/sei-protocol/sei-load").Start(ctx, "seiload.run")
	defer runSpan.End()

	// Create statistics collector and logger
	collector := stats.NewCollector()
	logger := stats.NewLogger(collector, cfg.Settings.StatsInterval.ToDuration(), cfg.Settings.ReportPath, cfg.Settings.Debug)
	rng := generator.ResolveSeed(cfg).Rand("")
	var ramper *sender.Ramper
	inclusion := utils.None[*stats.InclusionTracker]()

	err = scope.Run(ctx, func(ctx context.Context, s scope.Scope) error {
		// Create the generator from the config struct
		gen, err := generator.NewGenerator(rng, cfg)
		if err != nil {
			return fmt.Errorf("failed to create generator: %w", err)
		}

		// Create the shared rate authority for the whole run.
		sharedLimiter := rate.NewLimiter(rate.Inf, 1)
		if cfg.Settings.TPS > 0 {
			sharedLimiter = rate.NewLimiter(rate.Limit(cfg.Settings.TPS), 1)
			log.Printf("📈 Rate limiting enabled: %.2f TPS shared across all workers", cfg.Settings.TPS)
		}

		// Create and start block collector if endpoints are available
		var blockCollector *stats.BlockCollector
		if len(cfg.Endpoints) > 0 && cfg.Settings.TrackBlocks {
			blockCollector = stats.NewBlockCollector(cfg.SeiChainID)
			collector.SetBlockCollector(blockCollector)
			s.SpawnBgNamed("block collector", func() error {
				return blockCollector.Run(ctx, cfg.Endpoints[0])
			})
		}

		if cfg.Settings.RampUp {
			ramperBlockCollector := stats.NewBlockCollector(cfg.SeiChainID)
			s.SpawnBgNamed("ramper block collector", func() error {
				return ramperBlockCollector.Run(ctx, cfg.Endpoints[0])
			})

			ramper = sender.NewRamper(
				sender.NewRampCurveStep(100, 100, 120*time.Second, 30*time.Second),
				ramperBlockCollector,
				sharedLimiter,
			)
			s.SpawnBgNamed("ramper", func() error { return ramper.Run(ctx) })
		}

		// Create and start user latency tracker if endpoints are available
		if len(cfg.Endpoints) > 0 && cfg.Settings.TrackUserLatency {
			userLatencyTracker := stats.NewUserLatencyTracker(cfg.Settings.StatsInterval.ToDuration())
			s.SpawnBgNamed("user latency tracker", func() error {
				return userLatencyTracker.Run(ctx, cfg.Endpoints[0])
			})
		}

		// The --track-receipts flag now enables the block-indexed inclusion
		// tracker (the lossy per-tx receipt path is retired).
		// Not wired under --dry-run: simulated sends never hit the chain, so they
		// would all reap as expired and pollute the inclusion stats.
		if len(cfg.Endpoints) > 0 && cfg.Settings.TrackReceipts && !cfg.Settings.DryRun {
			reapAfter := cfg.Settings.InclusionReapAfter.ToDuration()
			inclusionTracker := stats.NewInclusionTracker(
				cfg.SeiChainID,
				reapAfter,
				inclusionRegistryCap(cfg.Settings.MaxInFlight, cfg.Settings.TPS, reapAfter),
				cfg.Settings.ArrivalModel == config.ArrivalModelOpenLoop,
			)
			inclusion = utils.Some(inclusionTracker)
			s.SpawnBgNamed("inclusion tracker", func() error {
				return inclusionTracker.Run(ctx, cfg.Endpoints[0])
			})
		}

		q := types.NewTxsQueue()
		if cfg.Settings.TxsDir != "" {
			// get latest height
			eth, err := ethclient.Dial(cfg.Endpoints[0])
			if err != nil {
				return fmt.Errorf("failed to create ethclient: %w", err)
			}
			latestHeight, err := eth.BlockNumber(ctx)
			if err != nil {
				return fmt.Errorf("failed to get latest height: %w", err)
			}
			numBlocksToWrite := cfg.Settings.NumBlocksToWrite
			writerHeight := latestHeight + 10 // some buffer
			log.Printf("🔍 Latest height: %d, writer start height: %d", latestHeight, writerHeight)
			writer := sender.NewTxsWriter(cfg.Settings.TargetGas, cfg.Settings.TxsDir, writerHeight, uint64(numBlocksToWrite))
			s.SpawnBgNamed("writer", func() error { return writer.Run(ctx, q) })
		} else {
			// Fund the pool before prewarm/dispatch — both spend gas the accounts
			// don't have until funded.
			if cfg.Funding != nil && !cfg.Settings.DryRun {
				var addrs []common.Address
				for _, a := range gen.Accounts() {
					addrs = append(addrs, a.Address)
				}
				if err := funder.FundAccounts(ctx, cfg, addrs); err != nil {
					return fmt.Errorf("failed to fund accounts: %w", err)
				}
			}
			// Create the sender from the config struct
			sharedSender := sender.NewShardedSender(cfg, sharedLimiter, collector, inclusion)
			// Start the sender (starts all workers)
			s.SpawnBgNamed("sender", func() error { return sharedSender.Run(ctx, q) })
			log.Printf("✅ Connected to %d endpoints", len(cfg.Endpoints))
		}

		// Set up prewarming if enabled
		if cfg.Settings.Prewarm {
			log.Printf("🔥 Creating prewarm generator...")
			if err := gen.Prewarm(ctx, rng, cfg, q); err != nil {
				return fmt.Errorf("gen.Prewarm(): %w", err)
			}
			log.Printf("🔥 Prewarming complete!")
		}

		// Start logger (after prewarming to capture only main load test metrics)
		s.SpawnBgNamed("logger", func() error { return logger.Run(ctx) })
		log.Printf("✅ Started statistics logger")

		// Start dispatcher for main load test
		s.SpawnBgNamed("generator", func() error { return gen.Run(ctx, rng, q) })

		log.Printf("✅ Started dispatcher")

		// Set up signal handling for graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		log.Printf("📈 Logging statistics every %v (Press Ctrl+C to stop)", cfg.Settings.StatsInterval.ToDuration())
		if cfg.Settings.DryRun {
			log.Printf("📝 Dry-run mode: Simulating requests without sending")
		}
		if cfg.Settings.Debug {
			log.Printf("🐛 Debug mode: Each transaction will be logged")
		}
		if cfg.Settings.TrackReceipts {
			log.Printf("📝 Track receipts mode: Receipts will be tracked")
		}
		if cfg.Settings.TrackBlocks {
			log.Printf("📝 Track blocks mode: Block data will be collected")
		}
		if cfg.Settings.TrackUserLatency {
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
	if cfg.Settings.RampUp && ramper != nil {
		ramper.LogFinalStats()
	}
	summary := stats.RunSummary{ArrivalModel: config.ArrivalModelClosedLoop}
	// Read AFTER service.Run returns: both workers and the tracker have joined,
	// so inflightAtShutdown is final and the conservation identity holds.
	if inclusionTracker, ok := inclusion.Get(); ok {
		incl := inclusionTracker.Summary()
		summary.InclusionTracked = true
		summary.Included = incl.Included
		summary.Expired = incl.Expired
		summary.DroppedAtCap = incl.DroppedAtCap
		summary.InflightAtShutdown = incl.InflightAtShutdown
		log.Printf("📦 Inclusion: included=%d expired=%d dropped_at_cap=%d inflight_at_shutdown=%d",
			incl.Included, incl.Expired, incl.DroppedAtCap, incl.InflightAtShutdown)
	}
	collector.EmitRunSummary(ctx, summary)
	if d := cfg.Settings.PostSummaryFlushDelay.ToDuration(); d > 0 {
		log.Printf("⏳ Holding pod for post-summary scrape window (%s)...", d)
		time.Sleep(d)
	}
	log.Printf("👋 Shutdown complete")
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		err = nil
	}
	return err
}

// inclusionRegistryCap sizes the inclusion registry. A registry entry lives from
// send-completion until block-match or reapAfter — far longer than a send is
// in-flight — so MaxInFlight (which bounds concurrent SENDS) under-sizes it. By
// Little's law the steady-state registry size ≈ sendRate × residency, so for a
// fixed rate the cap must come from TPS × reapAfter (×1.5 headroom for jitter),
// not send concurrency, or healthy high-TPS runs hit dropped_at_cap and
// undercount inclusion. We take the MAX of that term and the legacy MaxInFlight×4
// floor. For TPS<=0 (a ramped run with no fixed rate known at config time) the
// Little's-law term is 0 and we fall back to the floor; if the ramp peak exceeds
// it the run surfaces dropped_at_cap (un-defer: derive from the ramp peak then).
func inclusionRegistryCap(maxInFlight int, tps float64, reapAfter time.Duration) int {
	const maxInflightMultiple = 4
	const headroom = 1.5
	floor := maxInFlight * maxInflightMultiple
	little := int(math.Ceil(tps * reapAfter.Seconds() * headroom))
	if little > floor {
		return little
	}
	return floor
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

	if err := cfg.ValidateFunding(); err != nil {
		return nil, err
	}

	return &cfg, nil
}
