package config

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"sync"

	"github.com/sei-protocol/sei-load/utils/rng"
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

// Distribution is a tagged keyspace index sampler selected by a "Name"
// discriminator on the JSON wire. The discriminator strings and "theta"
// parameter are a FROZEN saved-workload contract; see package doc.
type Distribution struct {
	name     string
	delegate indexSampler
}

func (d *Distribution) Name() string { return d.name }

// SetStream binds the sampler to a deterministic sub-stream (nil = unseeded
// global RNG); a zero-value Distribution draws nothing, so it no-ops. See
// package doc for the reproducibility contract.
func (d *Distribution) SetStream(s *rng.Stream) {
	switch delegate := d.delegate.(type) {
	case *UniformDistribution:
		delegate.stream = s
	case *ZipfianDistribution:
		delegate.stream = s
	}
}

// SampleIndex delegates to the selected sampler; a zero-value Distribution
// returns 0.
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
		// No JSON parameters; the stream is bound later via SetStream.
		d.delegate = &UniformDistribution{}
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
type UniformDistribution struct {
	stream *rng.Stream
}

func (u *UniformDistribution) SampleIndex(n uint64) (uint64, error) {
	if n == 0 {
		return 0, fmt.Errorf("uniform sample: empty keyspace (n == 0)")
	}
	if u.stream != nil {
		return u.stream.Uint64N(n), nil
	}
	return rand.Uint64N(n), nil
}

// ZipfianDistribution is the YCSB precomputed-zeta generator: zeta(n, theta) is
// computed once per keyspace size n and cached, so each draw is O(1). theta == 0
// is uniform; larger theta hotspots the low indices. See package doc for the math.
//
// not copy-safe: holds a sync.Mutex; use only via *ZipfianDistribution.
type ZipfianDistribution struct {
	Theta float64 `json:"theta"`

	stream *rng.Stream

	mu    sync.Mutex
	state *zipfState // memoized for state.n; recomputed when n changes.
}

// zipfState holds the O(1) draw constants for one keyspace size n. See package doc.
type zipfState struct {
	n            uint64
	theta        float64
	zetaN        float64 // zeta(n, theta)
	alpha        float64 // 1 / (1 - theta)
	eta          float64
	halfPowTheta float64 // 0.5^theta, the boundary mass for index 1
}

// newZipfState precomputes zeta(n, theta) in O(n) and the O(1) draw constants;
// see package doc for the derivation.
func newZipfState(n uint64, theta float64) *zipfState {
	zetaN := zeta(n, theta)
	zeta2 := zeta(2, theta)

	// eta is unread for n <= 2; pin its NaN denominator to 0. See package doc.
	denom := 1.0 - zeta2/zetaN
	var eta float64
	if denom != 0 {
		eta = (1.0 - math.Pow(2.0/float64(n), 1.0-theta)) / denom
	}

	return &zipfState{
		n:            n,
		theta:        theta,
		zetaN:        zetaN,
		alpha:        1.0 / (1.0 - theta),
		eta:          eta,
		halfPowTheta: math.Pow(0.5, theta),
	}
}

// zeta returns the generalized harmonic number sum_{i=1..n} 1/i^theta, summed
// smallest-term-first for numerical stability; see package doc.
func zeta(n uint64, theta float64) float64 {
	var sum float64
	for i := n; i >= 1; i-- {
		sum += 1.0 / math.Pow(float64(i), theta)
	}
	return sum
}

// zipfianThetaMax bounds theta to the numerically well-behaved range; see package doc.
const zipfianThetaMax = 1.0

func (z *ZipfianDistribution) validate() error {
	if z.Theta < 0 || z.Theta >= zipfianThetaMax {
		return fmt.Errorf("zipfian theta out of range: %v (want [0, %v))", z.Theta, zipfianThetaMax)
	}
	return nil
}

// SampleIndex draws a Zipf-skewed index in [0, n). n must be stable per sampler:
// the zeta cache is keyed on n, so a changing n recomputes O(n) every draw. See
// package doc.
func (z *ZipfianDistribution) SampleIndex(n uint64) (uint64, error) {
	if n == 0 {
		return 0, fmt.Errorf("zipfian sample: empty keyspace (n == 0)")
	}

	z.mu.Lock()
	if z.state == nil || z.state.n != n || z.state.theta != z.Theta {
		z.state = newZipfState(n, z.Theta)
	}
	st := z.state
	z.mu.Unlock()

	var u float64
	if z.stream != nil {
		u = z.stream.Float64()
	} else {
		u = rand.Float64()
	}
	uz := u * st.zetaN
	if uz < 1.0 {
		return 0, nil
	}
	if uz < 1.0+st.halfPowTheta {
		return 1, nil
	}
	idx := uint64(float64(n) * math.Pow(st.eta*u-st.eta+1.0, st.alpha))
	if idx >= n { // guard against float rounding at the top of the range.
		idx = n - 1
	}
	return idx, nil
}
