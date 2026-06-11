package config

import (
	"encoding/json"
	"fmt"
)

var (
	_ indexSampler = (*Distribution)(nil)
	_ indexSampler = (*UniformDistribution)(nil)
	_ indexSampler = (*ZipfianDistribution)(nil)
)

// indexSampler draws an index in [0, n) from some keyspace distribution.
type indexSampler interface {
	SampleIndex(n uint64) (uint64, error)
}

// Distribution is a tagged wrapper over a keyspace index distribution, selected
// by a "Name" discriminator on the JSON wire format. The discriminator strings
// ("uniform", "zipfian") and the "theta" parameter name are a frozen
// saved-workload contract; do not rename them.
type Distribution struct {
	name     string
	delegate indexSampler
}

func (d *Distribution) Name() string { return d.name }

// SampleIndex delegates to the selected distribution. A zero-value (no Name)
// Distribution samples nothing and returns 0.
func (d *Distribution) SampleIndex(n uint64) (uint64, error) {
	if d.delegate == nil {
		return 0, nil
	}
	return d.delegate.SampleIndex(n)
}

func (d *Distribution) UnmarshalJSON(data []byte) error {
	var temp struct {
		Name string `json:"Name"`
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	d.name = temp.Name
	switch d.name {
	case "":
		return nil
	case "uniform":
		var uniform UniformDistribution
		if err := json.Unmarshal(data, &uniform); err != nil {
			return err
		}
		d.delegate = &uniform
		return nil
	case "zipfian":
		var zipfian ZipfianDistribution
		if err := json.Unmarshal(data, &zipfian); err != nil {
			return err
		}
		if err := zipfian.validate(); err != nil {
			return err
		}
		d.delegate = &zipfian
		return nil
	default:
		return fmt.Errorf("unknown distribution name: %s", d.name)
	}
}

// UniformDistribution draws each index with equal probability.
type UniformDistribution struct{}

func (UniformDistribution) SampleIndex(n uint64) (uint64, error) {
	// PLT-460: implement the seeded uniform draw. Out of scope for PLT-455
	// (wire format + validation only).
	return 0, nil
}

// ZipfianDistribution draws indices with a Zipf-distributed skew controlled by
// theta. theta == 0 is uniform; larger theta concentrates draws on low indices.
type ZipfianDistribution struct {
	Theta float64 `json:"theta"`
}

// zipfianThetaMax bounds theta to the range over which the YCSB precomputed-zeta
// generator (PLT-460) is numerically well-behaved.
const zipfianThetaMax = 1.0

func (z *ZipfianDistribution) validate() error {
	if z.Theta < 0 || z.Theta >= zipfianThetaMax {
		return fmt.Errorf("zipfian theta out of range: %v (want [0, %v))", z.Theta, zipfianThetaMax)
	}
	return nil
}

func (z *ZipfianDistribution) SampleIndex(n uint64) (uint64, error) {
	// PLT-460: implement the YCSB precomputed-zeta zipfian draw with a seeded
	// RNG. Out of scope for PLT-455 (wire format + validation only).
	return 0, nil
}
