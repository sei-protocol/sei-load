package sender

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/sei-protocol/sei-load/stats"
	"github.com/sei-protocol/sei-load/utils/service"
	"golang.org/x/time/rate"
)

// This will manage the ramping process for the loadtest
// Ramping loadtest will being at the StartTps and spend LoadTime at each step, ending when we violate the chain SLO of
// 1 block per second over a given ramp period (as measured in the back half of the ramp time)
// If we successfully pass a given TPS, we will pause for PauseTime, and then start the next step.
// If we fail to pass a given TPS, we will stop the loadtest.

var ErrRampTestFailedSLO = errors.New("Ramp Test failed SLO")

type RamperConfig struct {
	IncrementTps float64
	LoadTime     time.Duration
	PauseTime    time.Duration
}

// stats from the highest successful step
type RampStats struct {
	Step             int
	TargetTPS        float64
	WindowBlockStats stats.BlockStats
}

func (r RampStats) FormatRampStats() string {
	return fmt.Sprintf(`
─────────────────────────────────────────
              RAMP STATISTICS
─────────────────────────────────────────
 Step:       %d
 Target TPS: %.2f
─────────────────────────────────────────
 Window Block Stats:
 %s
─────────────────────────────────────────`,
		r.Step, r.TargetTPS, r.WindowBlockStats.FormatBlockStats())
}

type Ramper struct {
	sharedLimiter  *rate.Limiter
	cfg            *RamperConfig
	blockCollector stats.BlockStatsProvider
	currentTps     float64
	step           int
	startTime      time.Time
	stopTime       time.Time
	latestStats    RampStats
}

func NewRamper(cfg *RamperConfig, blockCollector stats.BlockStatsProvider, sharedLimiter *rate.Limiter) *Ramper {
	sharedLimiter.SetLimit(rate.Limit(0)) // reset limiter to 0
	return &Ramper{
		sharedLimiter:  sharedLimiter,
		cfg:            cfg,
		blockCollector: blockCollector,
		currentTps:     0,
		step:           0,
		startTime:      time.Now(),
		stopTime:       time.Time{},
	}
}

func (r *Ramper) NewStep() error {
	r.step++
	r.currentTps = r.cfg.IncrementTps * float64(r.step)
	r.sharedLimiter.SetLimit(rate.Limit(r.currentTps))
	r.startTime = time.Now()
	log.Printf("📈 Ramping to step %d with TPS %f for %v", r.step, r.currentTps, r.cfg.LoadTime)
	return nil
}

func (r *Ramper) LogFinalStats() {
	log.Printf("Final Ramp stats: \n%s", r.latestStats.FormatRampStats())
}

// For ramping loadtest SLO, we'll look at the block time p50, if this increases beyond 1s, we consider it an uptime failure
func (r *Ramper) WatchSLO(ctx context.Context) <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		// reset blockCollector window
		defer close(ch)
		r.blockCollector.ResetWindowStats()
		// wait for half of the load time
		log.Println("🔍 Ramping watching chain SLO")
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// we need to watch the monitoring endpoint for the SLO
				// Add appropriate monitoring logic here with timeout/context respect
				// check window stats
				if r.blockCollector.GetWindowBlockTimePercentile(90) > 1*time.Second {
					ch <- struct{}{}
				}
				time.Sleep(200 * time.Millisecond) // TODO: maybe this is too frequent?
				continue
			}
		}
	}()
	return ch
}

// Start initializes and starts all workers
func (r *Ramper) Run(ctx context.Context) error {
	return service.Run(ctx, func(ctx context.Context, s service.Scope) error {
		// TODO: Implement ramping logic
		for {
			r.NewStep()
			loadTimer := time.After(r.cfg.LoadTime)
			sloChan := r.WatchSLO(ctx)
			select {
			case <-sloChan:
				r.sharedLimiter.SetLimit(rate.Limit(1))
				log.Printf("❌ Ramping failed to pass SLO, stopping loadtest, failure window blockstats:")
				log.Println(r.blockCollector.GetWindowBlockStats().FormatBlockStats())
				return ErrRampTestFailedSLO
			case <-loadTimer:
				r.sharedLimiter.SetLimit(rate.Limit(1)) // set limit to 1 to "pause" load
				log.Printf("✅ Ramping passed current step, sleeping for %v", r.cfg.PauseTime)
				// newest stats
				stepStats := RampStats{
					Step:             r.step,
					TargetTPS:        r.currentTps,
					WindowBlockStats: r.blockCollector.GetWindowBlockStats(),
				}
				r.latestStats = stepStats
				log.Printf("🔍 Block stats: %s", r.blockCollector.GetWindowBlockStats().FormatBlockStats())
				time.Sleep(r.cfg.PauseTime)
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	})
}
