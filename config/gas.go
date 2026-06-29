package config

import (
	"encoding/json"
	"fmt"
	mrand "math/rand/v2"
)

var (
	_ gasGenerator = (*GasPicker)(nil)
	_ gasGenerator = (*FixedGasGenerator)(nil)
	_ gasGenerator = (*RandomGasGenerator)(nil)
)

type gasGenerator interface {
	GenerateGas(rng *mrand.Rand) (uint64, error)
}

type GasPicker struct {
	name     string
	delegate gasGenerator
}

func (g *GasPicker) Name() string { return g.name }

func (g *GasPicker) GenerateGas(rng *mrand.Rand) (uint64, error) {
	if g.delegate == nil {
		return 0, nil
	}
	return g.delegate.GenerateGas(rng)
}

func (g *GasPicker) UnmarshalJSON(data []byte) error {
	var temp struct {
		Name string `json:"Name"`
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	g.name = temp.Name
	switch g.name {
	case "":
		return nil
	case "fixed":
		var fixed FixedGasGenerator
		if err := json.Unmarshal(data, &fixed); err != nil {
			return err
		}
		g.delegate = &fixed
		return nil
	case "random":
		var random RandomGasGenerator
		if err := json.Unmarshal(data, &random); err != nil {
			return err
		}
		g.delegate = &random
		return nil
	default:
		return fmt.Errorf("unknown gas generator name: %s", g.name)
	}
}

type FixedGasGenerator struct {
	Gas uint64 `json:"Gas"`
}

func (f *FixedGasGenerator) GenerateGas(rng *mrand.Rand) (uint64, error) {
	return f.Gas, nil
}

type RandomGasGenerator struct {
	Min uint64 `json:"Min"`
	Max uint64 `json:"Max"`
}

func (r *RandomGasGenerator) GenerateGas(rng *mrand.Rand) (uint64, error) {
	if r.Min >= r.Max {
		return 0, fmt.Errorf("invalid random gas range: min %d must be less than max %d", r.Min, r.Max)
	}
	span := r.Max - r.Min + 1
	return r.Min + rng.Uint64N(span), nil
}
