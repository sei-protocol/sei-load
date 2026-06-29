// Package config holds the load-test workload configuration: the JSON wire
// types (LoadConfig, Scenario, Settings, gas and funding pickers) and the
// keyspace Distribution primitive that scenarios use to skew which key/size
// index they touch.
//
// # Distribution primitive
//
// A Distribution is a tagged sampler that draws an index in [0, n) from some
// keyspace distribution (see distribution.go). It is selected on the JSON wire
// by a "Name" discriminator and bound at run time to an explicit seeded PRNG so
// that two runs at the same seed draw the same sequence of indices.
//
// # Wire format (FROZEN one-way door)
//
// The discriminator lives in the "Name" field and selects the delegate:
//
//	{"Name": "uniform"}
//	{"Name": "zipfian", "theta": 0.9}
//
// The discriminator strings ("uniform", "zipfian") and the "theta" parameter
// name are a FROZEN saved-workload contract: saved configs are keyed by
// config_sha256, so renaming any of them changes how an old config parses (or
// stops it parsing) and silently diverges replays. Treat them as a one-way
// door — add new names, never rename existing ones. A zero-value Distribution
// (empty Name) draws no randomness and samples 0.
//
// # Semantics: uniform vs zipfian(theta)
//
// uniform draws every index in [0, n) with equal probability.
//
// zipfian(theta) draws with a Zipf-distributed skew controlled by theta in
// [0, 1): theta -> 0 approaches uniform, while theta -> 1 concentrates draws on
// the low indices, producing a hotspot. theta is validated to [0, 1) because
// the precomputed-zeta generator below is numerically well-behaved only over
// that range (alpha = 1/(1-theta) diverges at theta = 1).
//
// # YCSB precomputed-zeta math
//
// The zipfian sampler is the YCSB precomputed-zeta generator. It rests on the
// generalized harmonic number
//
//	zeta(n, theta) = sum_{i=1..n} 1 / i^theta
//
// which is O(n) to compute. zeta(n, theta) is summed from the largest i (the
// smallest term) down to i = 1: accumulating smallest-term-first keeps the
// running sum from being swamped by its leading terms, which matters for the
// n ~ 1e6 keyspaces this generator targets.
//
// From zeta(n, theta) the generator derives a set of constants that make each
// draw O(1):
//
//	alpha = 1 / (1 - theta)                       // draw exponent
//	eta   = (1 - (2/n)^(1-theta)) / (1 - zeta(2,theta)/zeta(n,theta))
//	0.5^theta                                     // boundary mass for index 1
//
// A draw takes one uniform u in [0, 1), forms uz = u * zeta(n, theta), and
// branches: uz < 1 returns index 0; uz < 1 + 0.5^theta returns index 1;
// otherwise the index is floor(n * (eta*u - eta + 1)^alpha), clamped to n-1 to
// absorb floating-point rounding at the top of the range.
//
// Precompute-once design: zeta(n, theta) and the constants depend only on
// (n, theta), so they are computed once and cached, keyed on n. n arrives at
// sample time (not at unmarshal time), so the cache fills lazily on first draw
// and is recomputed only if a later draw presents a different n. After the
// first draw, every subsequent draw is O(1).
//
// Edge behavior: at n <= 2, zeta(2, theta) == zeta(n, theta) so eta's
// denominator is 0 and eta would be NaN. eta is provably never read for those
// keyspaces (they are fully served by the uz < 1 and uz < 1 + 0.5^theta
// branches), but a NaN cached in state is a refactor hazard, so it is pinned to
// 0. theta = 0 reduces the generator to uniform sampling.
//
// # n must be stable per sampler
//
// The cache is keyed on n, so each sampler must be presented a stable n across
// its draws. A changing n triggers an O(n) zeta recompute on every draw and
// serializes those draws behind the cache mutex. Callers bind one sampler per
// fixed-size keyspace. (ZipfianDistribution holds that mutex and is therefore
// not copy-safe; use it only via pointer.)
//
// # Seeded-stream reproducibility (FROZEN inputs)
//
// Draws go through an explicitly supplied *rand.Rand seeded from the run seed.
// This is what gives the workload its reproducibility contract: same seed +
// same config yields the same draw sequence for the same call order.
package config
