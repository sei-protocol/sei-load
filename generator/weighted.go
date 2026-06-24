package generator

import (
	"context"
	"math/rand/v2"

	"github.com/sei-protocol/sei-load/types"
	"github.com/sei-protocol/sei-load/utils/rng"
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

// GenerateInfinite generates transactions indefinitely.
func (w *weightedGenerator) GenerateInfinite(ctx context.Context) <-chan *types.LoadTx {
	output := make(chan *types.LoadTx, 10000)
	go func() {
		defer close(output)
		for ctx.Err() == nil {
			select {
			case <-ctx.Done():
				return
			default:
				select {
				case <-ctx.Done():
					return
				case output <- func() *types.LoadTx {
					tx, _ := w.nextGenerator().Generate()
					return tx
				}():
				}
			}
		}
	}()
	return output
}

func (w *weightedGenerator) nextIndex() int {
	idx := int(w.counter) % len(w.generators)
	w.counter++
	return idx
}

// generators are randomized at startup.
func (w *weightedGenerator) nextGenerator() Generator {
	return w.generators[w.nextIndex()]
}

// Generate generates 1 transaction.
func (w *weightedGenerator) Generate() (*types.LoadTx, bool) {
	return w.nextGenerator().Generate()
}

// GetAccountPools returns all account pools from underlying generators
func (w *weightedGenerator) GetAccountPools() []*types.AccountPool {
	var allPools []*types.AccountPool

	// Collect pools from all underlying generators
	for _, gen := range w.generators {
		pools := gen.GetAccountPools()
		allPools = append(allPools, pools...)
	}

	return allPools
}

// NewWeightedGenerator creates a new scenarioGenerator that will randomly select
// from the provided generators. A nil stream leaves the startup shuffle on the
// unseeded global RNG.
func NewWeightedGenerator(stream *rng.Stream, cfgs ...*WeightedCfg) Generator {
	// no need for clever weighting logic if we just have 1 scenarioGenerator anyway.
	if len(cfgs) == 1 {
		return cfgs[0].Generator
	}
	var weighted []Generator
	for _, cfg := range cfgs {
		for range cfg.Weight {
			weighted = append(weighted, cfg.Generator)
		}
	}

	shuffle := rand.Shuffle
	if stream != nil {
		shuffle = stream.Shuffle
	}
	shuffle(len(weighted), func(i, j int) {
		weighted[i], weighted[j] = weighted[j], weighted[i]
	})

	return &weightedGenerator{
		generators: weighted,
	}
}
