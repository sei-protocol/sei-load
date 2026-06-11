// Package rng derives independent, reproducible pseudo-random sub-streams from
// a single run seed.
//
// The run summary promises replay: same seed + same config produces the same
// run. That holds only if every random draw is reproducible AND if the draw a
// given logical consumer sees does not depend on how many worker goroutines are
// running. Sub-streams are therefore keyed by a *logical* stream id (a string
// naming the consumer/purpose), never by a live-goroutine counter.
package rng

import (
	"crypto/rand"
	"encoding/binary"
	mrand "math/rand/v2"
	"sync"
)

// FROZEN DERIVATION — DO NOT CHANGE.
//
// substream(seed, streamID) = NewPCG(seed, splitmix64(fnv1a64(streamID)))
//
// where seed is the run seed and streamID is the logical consumer name. The
// FNV-1a hash maps the name to a uint64; splitmix64 diffuses it so that
// near-identical names (e.g. "gas:0" / "gas:1") seed well-separated PCG states.
//
// Replay archives are keyed by config_sha256 (PLT-467). Changing this formula —
// the hash, the diffusion step, or the PCG argument order — silently produces a
// different draw sequence for the same (seed, streamID) and invalidates every
// saved replay. Any change is a one-way door requiring a config_sha256 version
// bump.
func substream(seed uint64, streamID string) *mrand.PCG {
	return mrand.NewPCG(seed, splitmix64(fnv1a64(streamID)))
}

const (
	fnvOffset64 = 1469598103934665603
	fnvPrime64  = 1099511628211
)

func fnv1a64(s string) uint64 {
	h := uint64(fnvOffset64)
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= fnvPrime64
	}
	return h
}

func splitmix64(x uint64) uint64 {
	x += 0x9e3779b97f4a7c15
	x = (x ^ (x >> 30)) * 0xbf58476d1ce4e5b9
	x = (x ^ (x >> 27)) * 0x94d049bb133111eb
	return x ^ (x >> 31)
}

// Source derives sub-streams for a single run from one seed.
type Source struct {
	seed uint64
}

// NewSource returns a Source rooted at the given seed.
func NewSource(seed uint64) *Source {
	return &Source{seed: seed}
}

// NewRandomSource generates a cryptographically-random seed and returns a Source
// rooted at it alongside the resolved seed, so an unseeded run can still be
// replayed after the fact by re-running with the returned seed.
func NewRandomSource() (*Source, uint64) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("rng: crypto/rand failed: " + err.Error())
	}
	seed := binary.LittleEndian.Uint64(b[:])
	return NewSource(seed), seed
}

// Seed returns the run seed this Source is rooted at.
func (s *Source) Seed() uint64 { return s.seed }

// Stream returns the sub-stream for a logical consumer named streamID. The same
// (seed, streamID) always yields the same draw sequence, independent of worker
// count or of any other stream's draws.
func (s *Source) Stream(streamID string) *Stream {
	return &Stream{rand: mrand.New(substream(s.seed, streamID))}
}

// Stream is a single consumer's reproducible sub-stream. It is safe for
// concurrent use: draws are serialized so the per-stream sequence depends only
// on call order into this stream, not on the goroutine that made the call.
type Stream struct {
	mu   sync.Mutex
	rand *mrand.Rand
}

// Float64 returns a draw in [0.0, 1.0).
func (s *Stream) Float64() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rand.Float64()
}

// Uint64N returns a draw in [0, n).
func (s *Stream) Uint64N(n uint64) uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rand.Uint64N(n)
}

// Shuffle pseudo-randomizes the order of n elements via swap.
func (s *Stream) Shuffle(n int, swap func(i, j int)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rand.Shuffle(n, swap)
}
