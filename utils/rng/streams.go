package rng

import "fmt"

// FROZEN STREAM IDS — DO NOT CHANGE (see the FROZEN block in rng.go, input #2).
//
// Each stream id is hashed into a sub-stream seed, so renaming any of these — or
// changing an indexed format string — reseeds that stream and invalidates every
// saved replay for the same config_sha256. They are centralized here so the
// frozen naming surface is reviewable in one place and not edited at call sites.
const (
	// StreamAccountsShared seeds the shared (top-level) account pool.
	StreamAccountsShared = "accounts:shared"
	// StreamWeightedShuffle seeds the weighted scenario selector's shuffle.
	StreamWeightedShuffle = "weighted:shuffle"
)

// AccountsScenarioStream is the stream id for scenario i's own account pool.
func AccountsScenarioStream(i int) string {
	return fmt.Sprintf("accounts:scenario:%d", i)
}

// GasBaseStream is the stream id for scenario i's base gas picker.
func GasBaseStream(i int) string { return fmt.Sprintf("gas:%d:base", i) }

// GasTipStream is the stream id for scenario i's tip-cap gas picker.
func GasTipStream(i int) string { return fmt.Sprintf("gas:%d:tip", i) }

// GasFeeCapStream is the stream id for scenario i's fee-cap gas picker.
func GasFeeCapStream(i int) string { return fmt.Sprintf("gas:%d:feecap", i) }

// KeyDistributionStream is the stream id for scenario i's key-distribution
// index sampler (PLT-460).
func KeyDistributionStream(i int) string { return fmt.Sprintf("dist:%d:key", i) }

// SizeDistributionStream is the stream id for scenario i's size-distribution
// index sampler (PLT-460).
func SizeDistributionStream(i int) string { return fmt.Sprintf("dist:%d:size", i) }
