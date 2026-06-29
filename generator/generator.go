package generator

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"maps"
	mrand "math/rand/v2"
	"slices"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/sei-protocol/sei-load/config"
	"github.com/sei-protocol/sei-load/generator/scenarios"
	"github.com/sei-protocol/sei-load/types"
)

// scenarioInstance represents a scenario instance with its configuration
type scenarioInstance struct {
	Name     string
	Weight   int
	Scenario scenarios.TxGenerator
	Accounts *types.AccountPool
}

// generatorBuilder manages scenario creation and deployment from config
type generatorBuilder struct {
	config    *config.LoadConfig
	instances []*scenarioInstance
}

// CreateScenarios creates scenario instances based on the configuration
// Each scenario entry in config creates a separate instance, even if same name
func (g *generatorBuilder) createScenarios() error {
	var sharedAccounts *types.AccountPool
	if g.config.Accounts != nil {
		sharedAccounts = types.NewAccountPool(
			g.config.Accounts.Accounts,
			g.config.Accounts.NewAccountRate,
		)
	}

	for i, scenarioCfg := range g.config.Scenarios {
		// Create scenario instance using factory
		scenario := scenarios.CreateScenario(scenarioCfg)

		// Determine account pool to use
		var accountPool *types.AccountPool
		if cfg := scenarioCfg.Accounts; cfg != nil {
			// Scenario defines its own account settings - create separate pool
			accountPool = types.NewAccountPool(cfg.Accounts, cfg.NewAccountRate)
		} else if sharedAccounts != nil {
			// Use shared account pool from top-level config
			accountPool = sharedAccounts
		} else {
			return errors.New("no accounts config defined")
		}

		// Count how many times this scenario name appears in the config
		nameCount := 0
		nameIndex := 0
		for j, otherScenario := range g.config.Scenarios {
			if otherScenario.Name == scenarioCfg.Name {
				if j == i {
					nameIndex = nameCount
				}
				nameCount++
			}
		}

		name := scenarioCfg.Name
		if nameCount > 1 {
			name = fmt.Sprintf("%s_%d", name, nameIndex)
		}

		// Create scenario instance
		instance := &scenarioInstance{
			Name:     name,
			Weight:   scenarioCfg.Weight,
			Scenario: scenario,
			Accounts: accountPool,
		}

		g.instances = append(g.instances, instance)
	}

	return nil
}

// mockDeployAll deploys all scenario instances that require deployment (for unit tests).
func (g *generatorBuilder) mockDeployAll(deployer common.Address) error {
	for _, instance := range g.instances {
		if err := instance.Scenario.Attach(g.config, deployer); err != nil {
			return err
		}
	}
	return nil
}

// DeployAll deploys all scenario instances that require deployment
func (g *generatorBuilder) deployAll() error {
	deployer := types.NewAccount(false)
	if g.config.MockDeploy {
		return g.mockDeployAll(deployer.Address)
	}

	// Deploy sequentially to ensure proper nonce management
	for i, instance := range g.instances {
		// Deploy the scenario
		log.Printf("Deploying scenario %s", instance.Name)
		address := instance.Scenario.Deploy(g.config, deployer, uint64(i))
		if address != (common.Address{}) {
			log.Printf("🚀 Deployed %s at address: %s\n", instance.Name, address.Hex())
		}
	}

	return nil
}

type Generator struct{ scenarios []*scenarioInstance }

func (g *Generator) Accounts() []types.Account {
	accs := map[common.Address]types.Account{}
	for _, s := range g.scenarios {
		for _, a := range s.Accounts.Accounts() {
			accs[a.Address] = a
		}
	}
	return slices.Collect(maps.Values(accs))
}

type TxSender interface {
	Send(ctx context.Context, tx *types.LoadTx) error
	Flush(ctx context.Context) error
	Nonce(acc types.Account) uint64
}

// NewPrewarmGenerator creates a new prewarm generator using all account pools from the registry.
func (g *Generator) Prewarm(ctx context.Context, rng *mrand.Rand, cfg *config.LoadConfig, txSender TxSender) error {
	// Create EVMTransfer scenario for prewarming
	evmScenario := scenarios.NewEVMTransferScenario(config.Scenario{})
	// Deploy/initialize the scenario (EVMTransfer doesn't need actual deployment)
	evmScenario.Deploy(cfg, types.NewAccount(false), 0)
	for _, account := range g.Accounts() {
		// Create self-transfer transaction
		scenario := &types.TxScenario{
			Name:     "EVMTransfer",
			Nonce:    txSender.Nonce(account),
			Sender:   account,
			Receiver: account.Address, // Send to self
		}
		tx, err := evmScenario.Generate(rng, scenario)
		if err != nil {
			return fmt.Errorf("evmScenario.Generate(): %w", err)
		}
		ltx := &types.LoadTx{EthTx: tx, IntendedSendTime: time.Now(), Scenario: scenario}
		if err := txSender.Send(ctx, ltx); err != nil {
			return err
		}
	}
	return txSender.Flush(ctx)
}

type EthClient interface {
	Send(ctx context.Context, tx *types.LoadTx) error
	Nonce(ctx context.Context, acc types.Account) (uint64, error)
}

// Generate generates 1 transaction.
func (w *Generator) Run(ctx context.Context, rng *mrand.Rand, txSender TxSender) error {
	counter := 0
	for {
		g := w.scenarios[int(counter)%len(w.scenarios)]
		counter++
		sender := g.Accounts.NextAccount(rng)
		receiver := g.Accounts.NextAccount(rng)
		// TODO: This should probably hold a lock on sender.
		// Stamp before hand-off while sole owner: race-free (see LoadTx). This is
		// the back-pressured enqueue time, not a true schedule instant.
		scenario := &types.TxScenario{
			Name:     g.Scenario.Name(),
			Nonce:    txSender.Nonce(sender),
			Sender:   sender,
			Receiver: receiver.Address,
		}
		tx, err := g.Scenario.Generate(rng, scenario)
		if err != nil {
			return fmt.Errorf("g.Scenario.Generate(): %w", err)
		}
		ltx := &types.LoadTx{EthTx: tx, IntendedSendTime: time.Now(), Scenario: scenario}
		if err := txSender.Send(ctx, ltx); err != nil {
			return err
		}
	}
}

// createWeightedGenerator creates a weighted scenarioGenerator from deployed scenarios
func (b *generatorBuilder) build(rng *mrand.Rand) (*Generator, error) {
	// Create weighted configurations
	var gens []*scenarioInstance
	for _, instance := range b.instances {
		if instance.Weight == 0 {
			log.Printf("Skipping scenario %s with weight 0", instance.Name)
			continue
		}
		// Create a scenarioGenerator for this scenario instance
		for range instance.Weight {
			gens = append(gens, instance)
		}
	}

	if len(gens) == 0 {
		return nil, fmt.Errorf("no scenario instances created (define some scenarios)")
	}
	rng.Shuffle(len(gens), func(i, j int) {
		gens[i], gens[j] = gens[j], gens[i]
	})
	return &Generator{scenarios: gens}, nil
}

func newSeededRand(seed uint64) *mrand.Rand {
	return mrand.New(mrand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
}

func randomSeed() (uint64, error) {
	var buf [8]byte
	if _, err := cryptorand.Read(buf[:]); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(buf[:]), nil
}

// ResolveSeed returns the run's PRNG, defaulting an unseeded config to a random
// seed. The resolved seed is written back to cfg.Seed and logged so any run is
// replayable after the fact; the run summary (PLT-467) reads it there.
func ResolveSeed(cfg *config.LoadConfig) *mrand.Rand {
	if cfg.Seed != nil {
		return newSeededRand(*cfg.Seed)
	}
	seed, err := randomSeed()
	if err != nil {
		panic(fmt.Errorf("randomSeed(): %w", err))
	}
	cfg.Seed = &seed
	log.Printf("🎲 No seed configured; generated random seed %d (set \"seed\" to replay)", seed)
	return newSeededRand(seed)
}

// NewConfigBasedGenerator is a convenience method that combines all steps.
func NewGenerator(rng *mrand.Rand, cfg *config.LoadConfig) (*Generator, error) {
	b := &generatorBuilder{
		config:    cfg,
		instances: make([]*scenarioInstance, 0),
	}

	// Step 1: Create scenarios
	if err := b.createScenarios(); err != nil {
		return nil, fmt.Errorf("failed to create scenarios: %w", err)
	}

	// Step 2: Deploy all scenarios
	if err := b.deployAll(); err != nil {
		return nil, fmt.Errorf("failed to deploy scenarios: %w", err)
	}

	// Step 3: Create weighted scenarioGenerator
	g, err := b.build(rng)
	if err != nil {
		return nil, fmt.Errorf("failed to create weighted scenarioGenerator: %w", err)
	}

	return g, nil
}
