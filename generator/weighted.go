package generator

import (
	mrand "math/rand/v2"

	"github.com/sei-protocol/sei-load/types"
)

// WeightedCfg is a configuration for a weighted scenarioGenerator.
type WeightedCfg struct {
	Weight    int
	Generator Generator
}

// WeightedConfig creates a configuration for a weighted scenarioGenerator.
func WeightedConfig(weight int, generator Generator) *WeightedCfg {
	return &WeightedCfg{
		Weight:    weight,
		Generator: generator,
	}
}

type weightedGenerator struct {
	generators []Generator
	counter    uint64
}

// NewWeightedGenerator creates a new scenarioGenerator that will randomly select
// from the provided generators.
func NewWeightedGenerator(rng *mrand.Rand, cfgs []*WeightedCfg) Generator {
	var weighted []Generator
	for _, cfg := range cfgs {
		for range cfg.Weight {
			weighted = append(weighted, cfg.Generator)
		}
	}
	rng.Shuffle(len(weighted), func(i, j int) {
		weighted[i], weighted[j] = weighted[j], weighted[i]
	})

	return &weightedGenerator{generators: weighted}
}

// Generate generates 1 transaction.
func (w *weightedGenerator) Generate(rng *mrand.Rand) (*types.LoadTx, bool) {
	idx := int(w.counter) % len(w.generators)
	w.counter++
	return w.generators[idx].Generate(rng)
}
