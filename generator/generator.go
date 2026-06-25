package generator

import (
	"errors"
	"fmt"
	"log"
	"time"
	mrand "math/rand/v2"

	"github.com/ethereum/go-ethereum/common"

	"github.com/sei-protocol/sei-load/config"
	"github.com/sei-protocol/sei-load/generator/scenarios"
	"github.com/sei-protocol/sei-load/types"
	"github.com/sei-protocol/sei-load/utils/rng"
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
	config         *config.LoadConfig
	registry       *types.AccountRegistry
	instances      []*scenarioInstance
	deployer       *types.Account
	sharedAccounts *types.AccountPool // Shared account pool when using top-level config
}

// CreateScenarios creates scenario instances based on the configuration
// Each scenario entry in config creates a separate instance, even if same name
func (g *generatorBuilder) createScenarios() error {
	if g.config.Accounts != nil {
		g.sharedAccounts = g.registry.NewPool(&types.AccountConfig{
			InitialSize:    g.config.Accounts.Accounts,
			NewAccountRate: g.config.Accounts.NewAccountRate,
		})
	}

	for i, scenarioCfg := range g.config.Scenarios {
		// Create scenario instance using factory
		scenario := scenarios.CreateScenario(scenarioCfg)

		// Determine account pool to use
		var accountPool *types.AccountPool
		if accounts := scenarioCfg.Accounts; accounts != nil {
			// Scenario defines its own account settings - create separate pool
			accountPool = g.registry.NewPool(&types.AccountConfig{
				InitialSize:    accounts.Accounts,
				NewAccountRate: accounts.NewAccountRate,
			})
		} else if g.sharedAccounts != nil {
			// Use shared account pool from top-level config
			accountPool = g.sharedAccounts
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
func (g *generatorBuilder) mockDeployAll() error {
	for _, instance := range g.instances {
		addr := types.NewAccount().Address
		if err := instance.Scenario.Attach(g.config, addr); err != nil {
			return err
		}
	}
	return nil
}

// DeployAll deploys all scenario instances that require deployment
func (g *generatorBuilder) deployAll() error {
	if g.config.MockDeploy {
		return g.mockDeployAll()
	}

	// Deploy sequentially to ensure proper nonce management
	for _, instance := range g.instances {
		// Deploy the scenario
		log.Printf("Deploying scenario %s", instance.Name)
		address := instance.Scenario.Deploy(g.config, g.deployer)
		if address!=(common.Address{}) {
			log.Printf("🚀 Deployed %s at address: %s\n", instance.Name, address.Hex())
		}
	}

	return nil
}

type Generator struct {
	registry *types.AccountRegistry
	scenarios []*scenarioInstance
	counter    uint64
}

func (g *Generator) Accounts() []*types.Account {
	return g.registry.Accounts()
}

// NewPrewarmGenerator creates a new prewarm generator using all account pools from the registry.
func (g *Generator) PrewarmTxs(rng *mrand.Rand, cfg *config.LoadConfig) []*types.LoadTx {
	// Create EVMTransfer scenario for prewarming
	evmScenario := scenarios.NewEVMTransferScenario(config.Scenario{})
	// Deploy/initialize the scenario (EVMTransfer doesn't need actual deployment)
	evmScenario.Deploy(cfg, types.NewAccount())
	var txs []*types.LoadTx
	for _,account := range g.registry.Accounts() {
		// Create self-transfer transaction
		scenario := &types.TxScenario{
			Name:     "EVMTransfer",
			Sender:   account,
			Receiver: account.Address, // Send to self
		}
		txs = append(txs,evmScenario.Generate(rng, scenario))
	}
	return txs
}

// Generate generates 1 transaction.
func (w *Generator) Generate(rng *mrand.Rand) {
	g := w.scenarios[int(w.counter) % len(w.scenarios)]
	w.counter++
	sender := g.Accounts.NextAccount(rng)
	receiver := g.Accounts.NextAccount(rng)
	// TODO: This should probably hold a lock on sender.
	// Stamp before hand-off while sole owner: race-free (see LoadTx). This is
	// the back-pressured enqueue time, not a true schedule instant.
	tx := g.Scenario.Generate(rng, &types.TxScenario{
		Name:     g.Scenario.Name(),
		Sender:   sender,
		Receiver: receiver.Address,
	})
	tx.IntendedSendTime = time.Now()
	sender.PushTx(tx)
}

// createWeightedGenerator creates a weighted scenarioGenerator from deployed scenarios
func (g *generatorBuilder) build(rng *mrand.Rand) (*Generator, error) {
	// Create weighted configurations
	var gens []*scenarioInstance
	for _, instance := range g.instances {
		if instance.Weight == 0 {
			log.Printf("Skipping scenario %s with weight 0", instance.Name)
			continue
		}
		// Create a scenarioGenerator for this scenario instance
		for range instance.Weight {
			gens = append(gens,instance)
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

// resolveSeed returns the run's PRNG source, defaulting an unseeded config to a
// random seed. The resolved seed is written back to cfg.Seed and logged so any
// run is replayable after the fact; the run summary (PLT-467) reads it there.
func ResolveSeed(cfg *config.LoadConfig) *rng.Source {
	if cfg.Seed != nil {
		return rng.NewSource(*cfg.Seed)
	}
	src, seed := rng.NewRandomSource()
	cfg.Seed = &seed
	log.Printf("🎲 No seed configured; generated random seed %d (set \"seed\" to replay)", seed)
	return src
}

// NewConfigBasedGenerator is a convenience method that combines all steps.
func NewGenerator(rng *mrand.Rand, cfg *config.LoadConfig) (*Generator, error) {
	b := &generatorBuilder{
		config:    cfg,
		registry:  types.NewAccountRegistry(),
		instances: make([]*scenarioInstance, 0),
		deployer:  types.NewAccount(),
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
