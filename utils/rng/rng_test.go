package rng

import (
	"sync"
	"testing"
)

// drawSeq pulls n uint64 draws from the named stream of a fresh source.
func drawSeq(seed uint64, streamID string, n int) []uint64 {
	s := NewSource(seed).Stream(streamID)
	out := make([]uint64, n)
	for i := range out {
		out[i] = s.Uint64N(1 << 32)
	}
	return out
}

func TestSameSeedSameStreamReproduces(t *testing.T) {
	a := drawSeq(42, "gas:0:base", 64)
	b := drawSeq(42, "gas:0:base", 64)
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("draw %d differs: %d != %d", i, a[i], b[i])
		}
	}
}

// TestDistributionStreamsAreDistinct pins that scenario i's key, size, and op
// distribution streams are mutually distinct ids. Sharing any two would couple
// the axes — drawing a size or op would perturb the key sequence — which the
// StorageRW independence tests rely on being impossible.
func TestDistributionStreamsAreDistinct(t *testing.T) {
	ids := map[string]string{
		"key":  KeyDistributionStream(0),
		"size": SizeDistributionStream(0),
		"op":   OpDistributionStream(0),
	}
	seen := map[string]string{}
	for name, id := range ids {
		if prev, dup := seen[id]; dup {
			t.Fatalf("stream id %q shared by %s and %s", id, prev, name)
		}
		seen[id] = name
	}
}

func TestDifferentStreamsDiverge(t *testing.T) {
	a := drawSeq(42, "gas:0:base", 64)
	b := drawSeq(42, "gas:1:base", 64)
	same := true
	for i := range a {
		if a[i] != b[i] {
			same = false
			break
		}
	}
	if same {
		t.Fatal("near-identical stream ids produced identical sequences; diffusion failed")
	}
}

// The replay invariant: a stream's sequence depends only on call order into
// that stream, never on how many other streams exist or the order they were
// created. A goroutine-counter-based scheme would shift the sequence when more
// streams (more workers) are live; a logical-id scheme must not.
func TestStreamIndependentOfSiblings(t *testing.T) {
	const seed = 7
	want := drawSeq(seed, "accounts:shared", 100)

	src := NewSource(seed)
	for i := 0; i < 32; i++ {
		_ = src.Stream("noise")
	}
	target := src.Stream("accounts:shared")
	for i := 0; i < 16; i++ {
		_ = src.Stream("more-noise")
	}

	for i := range want {
		if got := target.Uint64N(1 << 32); got != want[i] {
			t.Fatalf("draw %d differs after sibling noise: %d != %d", i, got, want[i])
		}
	}
}

// A single stream is safe for concurrent draws (the per-stream mutex serializes
// them); this guards the -race build for the account-pool and weighted paths.
func TestStreamConcurrentDrawsRaceFree(t *testing.T) {
	s := NewSource(3).Stream("x")
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = s.Uint64N(1 << 32)
			}
		}()
	}
	wg.Wait()
}

func TestRandomSourceRecordsSeed(t *testing.T) {
	src, seed := NewRandomSource()
	if src.Seed() != seed {
		t.Fatalf("recorded seed %d != source seed %d", seed, src.Seed())
	}
	a := NewSource(seed).Stream("x").Uint64N(1 << 32)
	b := NewSource(seed).Stream("x").Uint64N(1 << 32)
	if a != b {
		t.Fatalf("recorded seed does not replay: %d != %d", a, b)
	}
}
