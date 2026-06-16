// Package rng derives independent, reproducible pseudo-random sub-streams from
// a single run seed.
//
// Reproducibility contract — read precisely:
//
// Same seed + same config => identical per-stream draw multiset. The workload
// (the distribution of keys, sizes, gas, and accounts) is statistically
// reproducible, which is what fair A/B comparison of two runs requires.
//
// What is NOT guaranteed: per-tx emission ordering across runs is reproducible
// only at a single worker. Above one worker, workers interleave their draws
// into the shared streams non-deterministically, so the ordered tx sequence
// differs run to run even at the same seed (the multiset still matches). On-chain
// arrival order is concurrent regardless of worker count, so it is never
// reproducible. Nothing here makes individual transactions byte-identical.
//
// Sub-streams are keyed by a *logical* stream id (a string naming the
// consumer/purpose), never by a live-goroutine counter, so the per-stream draw
// multiset a seed yields is invariant to --workers.
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
// Four inputs are FROZEN, not just this formula. Each perturbs the draw
// sequence with no formula change, so each is a one-way door requiring a
// config_sha256 version bump:
//
//  1. The derivation formula above (hash, diffusion, PCG argument order).
//  2. The set of stream-id strings (defined as constants in streams.go). The
//     streamID feeds fnv1a64, so renaming "gas:0:base" reseeds that stream.
//     Additions are append-only and do not perturb existing streams (a new id
//     hashes to its own sub-stream); PLT-460 added "dist:%d:key" and
//     "dist:%d:size" for the per-scenario distribution index samplers, and
//     PLT-465 added "dist:%d:op" for the per-scenario operation-mix selector.
//  3. The per-stream draw order. Each stream is a sequence; drawing base before
//     tip before feecap is part of the contract — reordering draws within a
//     stream shifts every downstream value.
//  4. The per-tx account draw cadence: sender then receiver NextAccount() per tx
//     (generator/scenario.go), each consuming the account stream. This is a
//     draw-order on the account stream just like #3 is for the gas streams —
//     reordering or adding an account draw per tx shifts every downstream
//     account value.
//
// Replay archives are keyed by config_sha256 (PLT-467). Changing any of the
// three silently produces a different draw sequence for the same (seed, config)
// and invalidates every saved replay.
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
// (seed, streamID) always yields the same draw sequence for a given call order
// into the stream, independent of any other stream's draws. Concurrent workers
// drawing from one stream still see a reproducible multiset, but their
// interleaving — and thus the per-call ordering — is non-deterministic.
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
