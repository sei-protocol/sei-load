package generator

import (
	"errors"
	"fmt"
	"log"
	mrand "math/rand/v2"

	"github.com/ethereum/go-ethereum/common"

	"github.com/sei-protocol/sei-load/config"
	"github.com/sei-protocol/sei-load/generator/scenarios"
	"github.com/sei-protocol/sei-load/types"
	"github.com/sei-protocol/sei-load/utils/rng"
)

// Generator defines the contract for transaction generators.
//
// Generators are not thread-safe. Callers must serialize all access to a given
// Generator instance.
type Generator interface {
	Generate(rng *mrand.Rand) (*types.LoadTx, bool) // Returns transaction and true if more available, nil/false when done
}

// GenerateN drains up to n transactions from g by repeated Generate calls.
func GenerateN(rng *mrand.Rand, g Generator, n int) []*types.LoadTx {
	txs := make([]*types.LoadTx, 0, n)
	for range n {
		if tx, ok := g.Generate(rng); ok {
			txs = append(txs, tx)
		} else {
			break
		}
	}
	return txs
}

// scenarioInstance represents a scenario instance with its configuration
type scenarioInstance struct {
	Name     string
	Weight   int
	Scenario scenarios.TxGenerator
	Accounts *types.AccountPool
	Deployed bool
}

// configBasedGenerator manages scenario creation and deployment from config
type configBasedGenerator struct {
	config         *config.LoadConfig
	registry       *types.AccountRegistry
	instances      []*scenarioInstance
	deployer       *types.Account
	sharedAccounts *types.AccountPool // Shared account pool when using top-level config
}

// CreateScenarios creates scenario instances based on the configuration
// Each scenario entry in config creates a separate instance, even if same name
func (g *configBasedGenerator) createScenarios() error {
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
			Deployed: false,
		}

		g.instances = append(g.instances, instance)
	}

	return nil
}

// mockDeployAll deploys all scenario instances that require deployment (for unit tests).
func (g *configBasedGenerator) mockDeployAll() error {
	for _, instance := range g.instances {
		addr := types.GenerateAccounts(1)[0].Address
		if err := instance.Scenario.Attach(g.config, addr); err != nil {
			return err
		}
		instance.Deployed = true
	}
	return nil
}

// DeployAll deploys all scenario instances that require deployment
func (g *configBasedGenerator) deployAll() error {
	if g.config.MockDeploy {
		return g.mockDeployAll()
	}

	// Deploy sequentially to ensure proper nonce management
	for _, instance := range g.instances {
		// Deploy the scenario
		log.Printf("Deploying scenario %s", instance.Name)
		address := instance.Scenario.Deploy(g.config, g.deployer)
		instance.Deployed = true

		if address.Cmp(common.Address{}) != 0 {
			log.Printf("🚀 Deployed %s at address: %s\n", instance.Name, address.Hex())
		}
	}

	return nil
}

// createWeightedGenerator creates a weighted scenarioGenerator from deployed scenarios
func (g *configBasedGenerator) createWeightedGenerator(rng *mrand.Rand) (Generator, error) {
	if len(g.instances) == 0 {
		return nil, fmt.Errorf("no scenario instances created")
	}

	// Check that all scenarios are deployed
	for _, instance := range g.instances {
		if !instance.Deployed {
			return nil, fmt.Errorf("scenario %s is not deployed", instance.Name)
		}
	}

	// Create weighted configurations
	var weightedConfigs []*WeightedCfg
	for _, instance := range g.instances {
		if instance.Weight == 0 {
			log.Printf("Skipping scenario %s with weight 0", instance.Name)
			continue
		}
		// Create a scenarioGenerator for this scenario instance
		gen := NewScenarioGenerator(instance.Accounts, instance.Scenario)

		// Add to weighted config with the specified weight
		weightedConfigs = append(weightedConfigs, WeightedConfig(instance.Weight, gen))
	}

	if len(weightedConfigs) == 0 {
		return nil, fmt.Errorf("no scenario instances created (define some scenarios)")
	}

	// Create and return the weighted scenarioGenerator
	return NewWeightedGenerator(rng, weightedConfigs), nil
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
func NewConfigBasedGenerator(rng *mrand.Rand, cfg *config.LoadConfig, registry *types.AccountRegistry) (Generator, error) {
	generator := &configBasedGenerator{
		config:    cfg,
		registry:  registry,
		instances: make([]*scenarioInstance, 0),
		deployer:  types.GenerateAccounts(1)[0],
	}

	// Step 1: Create scenarios
	if err := generator.createScenarios(); err != nil {
		return nil, fmt.Errorf("failed to create scenarios: %w", err)
	}

	// Step 2: Deploy all scenarios
	if err := generator.deployAll(); err != nil {
		return nil, fmt.Errorf("failed to deploy scenarios: %w", err)
	}

	// Step 3: Create weighted scenarioGenerator
	weightedGen, err := generator.createWeightedGenerator(rng)
	if err != nil {
		return nil, fmt.Errorf("failed to create weighted scenarioGenerator: %w", err)
	}

	return weightedGen, nil
}
