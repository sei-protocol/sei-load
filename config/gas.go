package config

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"

	"github.com/sei-protocol/sei-load/utils/rng"
)

var (
	_ gasGenerator = (*GasPicker)(nil)
	_ gasGenerator = (*FixedGasGenerator)(nil)
	_ gasGenerator = (*RandomGasGenerator)(nil)
)

type gasGenerator interface {
	GenerateGas() (uint64, error)
}

type GasPicker struct {
	name     string
	delegate gasGenerator
}

func (g *GasPicker) Name() string { return g.name }

// SetStream binds the picker's random delegate to a deterministic sub-stream. A
// nil stream leaves the picker on the unseeded global RNG.
//
// Only a random delegate has anything to seed: fixed and empty pickers draw no
// randomness, so the type assertion intentionally no-ops for them rather than
// erroring.
func (g *GasPicker) SetStream(s *rng.Stream) {
	if r, ok := g.delegate.(*RandomGasGenerator); ok {
		r.stream = s
	}
}

func (g *GasPicker) GenerateGas() (uint64, error) {
	if g.delegate == nil {
		return 0, nil
	}
	return g.delegate.GenerateGas()
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

func (f *FixedGasGenerator) GenerateGas() (uint64, error) {
	return f.Gas, nil
}

type RandomGasGenerator struct {
	Min uint64 `json:"Min"`
	Max uint64 `json:"Max"`

	stream *rng.Stream
}

func (r *RandomGasGenerator) GenerateGas() (uint64, error) {
	if r.Min >= r.Max {
		return 0, fmt.Errorf("invalid random gas range: min %d must be less than max %d", r.Min, r.Max)
	}
	span := r.Max - r.Min + 1
	if r.stream != nil {
		return r.Min + r.stream.Uint64N(span), nil
	}
	return r.Min + rand.Uint64N(span), nil
}
