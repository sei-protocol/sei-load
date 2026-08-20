package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"time"
)

// LoadConfig stores the configuration for load-related settings.
type LoadConfig struct {
	ChainID int64 `json:"chainId,omitempty"`
	// SeiChainID is the textual chain ID used for tagging metric collection.
	SeiChainID string         `json:"seiChainID,omitempty"`
	Endpoints  []string       `json:"endpoints"`
	Accounts   *AccountConfig `json:"accounts,omitempty"`
	Scenarios  []Scenario     `json:"scenarios,omitempty"`
	MockDeploy bool           `json:"mockDeploy,omitempty"`
	Settings   *Settings      `json:"settings,omitempty"`
	// Funding, when set, funds the generated account pool from a root key at
	// startup so the run works against a real chain. See funding.go.
	Funding *FundingConfig `json:"funding,omitempty"`
	// Path to write a JSON report of the load test.
	ReportPath string `json:"reportPath,omitempty"`
	// Seed roots the PRNG behind every workload draw: key and size
	// distributions, gas pickers, operation mixes, and account selection. The
	// same seed and the same config reproduce the same draw sequence.
	//
	// One stream serves the whole run, so the axes are reproducible together
	// rather than independently: adding, removing, or reweighting any axis
	// changes how many draws each transaction takes, which shifts every other
	// axis's sequence. Two runs compare only when their configs match — a saved
	// workload is the seed and the config together, never the seed alone.
	//
	// On-chain arrival order is concurrent regardless. A nil Seed means
	// "unseeded": the generator resolves a random one and records it for
	// after-the-fact replay.
	Seed *uint64 `json:"seed,omitempty"`
}

// ParseLoadConfig decodes a profile strictly: a key that maps to no config field
// is an error naming that key, and the scenario that carries it. A mistyped or
// stale key therefore fails the run instead of keeping its default.
//
// Four kinds of key still pass, because encoding/json resolves them before the
// strict check sees them:
//
//   - a key inside a type with its own UnmarshalJSON, such as a Distribution or
//     a GasPicker object, which parses its own payload;
//   - a key of a map-typed field, which accepts every key by construction;
//   - a key differing from a field's tag only by case, which encoding/json
//     matches case-insensitively;
//   - a repeated key, where the last occurrence wins.
//
// ParseLoadConfig checks no semantics. A caller MUST run ValidateScenarios and
// ValidateFunding next.
func ParseLoadConfig(data []byte) (*LoadConfig, error) {
	// Scenarios decode one at a time so an unknown key is reported against the
	// scenario that carries it. The declared field shadows the embedded one.
	var wire struct {
		LoadConfig
		Scenarios []json.RawMessage `json:"scenarios,omitempty"`
	}
	if err := decodeStrict(data, &wire); err != nil {
		return nil, err
	}

	cfg := wire.LoadConfig
	cfg.Scenarios = make([]Scenario, len(wire.Scenarios))
	for i, raw := range wire.Scenarios {
		if err := decodeStrict(raw, &cfg.Scenarios[i]); err != nil {
			return nil, fmt.Errorf("scenario %s: %w", scenarioLabel(raw, i), err)
		}
	}
	return &cfg, nil
}

// decodeStrict unmarshals JSON into v. It rejects a key that maps to no field,
// and data after the value, which json.Unmarshal also rejects.
func decodeStrict(data []byte, v any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return fmt.Errorf("unexpected data after the top-level JSON value")
	}
	return nil
}

// scenarioLabel names a scenario the way Scenario.Validate does, and falls back
// to its position when the scenario carries no name.
func scenarioLabel(raw json.RawMessage, index int) string {
	var named struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &named); err == nil && named.Name != "" {
		return fmt.Sprintf("%q", named.Name)
	}
	return fmt.Sprintf("at index %d", index)
}

func (c *LoadConfig) TotalQueueSize() int {
	// Backward compatible formula, consider making it a config value.
	return len(c.Endpoints) * c.Settings.BufferSize
}

// Duration wraps time.Duration to provide JSON unmarshaling support
type Duration time.Duration

// UnmarshalJSON implements json.Unmarshaler for Duration
func (d *Duration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration format: %w", err)
	}
	*d = Duration(parsed)
	return nil
}

// ToDuration converts Duration back to time.Duration
func (d Duration) ToDuration() time.Duration {
	return time.Duration(d)
}

// MarshalJSON implements json.Marshaler for Duration
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// GetChainID returns the chain ID as a big.Int.
func (c *LoadConfig) GetChainID() *big.Int {
	return big.NewInt(c.ChainID)
}

// AccountConfig stores the configuration for account generation.
type AccountConfig struct {
	NewAccountRate float64 `json:"newAccountRate,omitempty"`
	Accounts       int     `json:"count,omitempty"`
}

// Scenario represents each scenario in the load configuration.
type Scenario struct {
	Name             string         `json:"name,omitempty"`
	Weight           int            `json:"weight,omitempty"`
	Accounts         *AccountConfig `json:"accounts,omitempty"`
	GasPicker        *GasPicker     `json:"gasPicker,omitempty"`
	GasFeeCapPicker  *GasPicker     `json:"gasFeeCapPicker,omitempty"`
	GasTipCapPicker  *GasPicker     `json:"gasTipCapPicker,omitempty"`
	KeyDistribution  *Distribution  `json:"keyDistribution,omitempty"`
	SizeDistribution *Distribution  `json:"sizeDistribution,omitempty"`
	// RecordCount is the keyspace size the KeyDistribution indexes into: the
	// per-tx slot is a draw in [0, RecordCount). Zero (the default) is the
	// single-slot, 100%-conflict behavior.
	RecordCount uint64 `json:"recordCount,omitempty"`
	// SizeBuckets is the pad-length histogram the SizeDistribution indexes into:
	// the per-tx pad length is SizeBuckets[draw]. Empty (the default) is the
	// empty-pad behavior. Each entry must be between 0 and 1 MiB; Validate
	// rejects the config otherwise.
	SizeBuckets []int `json:"sizeBuckets,omitempty"`
	// Operations is the read/write/rmw selection mix. Nil (the default) is the
	// all-rmw behavior.
	Operations *OperationMix `json:"operations,omitempty"`
}

const (
	// maxCalldataPadBytes caps each SizeBuckets entry at 128 KiB. Two ceilings
	// sit above it and the cap stays under both: the CometBFT mempool rejects a
	// transaction over 1 MiB, which a 1 MiB pad already breaches on calldata
	// alone, and the pad is charged at the EIP-7623 floor rate, so 128 KiB costs
	// about 1.4M gas against a 50M block. It also keeps a stray extra digit from
	// pinning gigabytes: the send queue holds up to a few thousand transactions,
	// each retaining its own pad.
	maxCalldataPadBytes = 128 << 10 // 128 KiB

	// maxRecordCount caps the keyspace. The zipfian sampler precomputes zeta in
	// O(n) on its first draw, under a mutex, on the generator's only goroutine:
	// measured at ~26 ns per element, so 1e7 costs ~270 ms once and 1e11 would
	// stall the run for roughly a minute per order of magnitude with nothing
	// logged. The package doc puts the design target at ~1e6, so this leaves an
	// order of magnitude of headroom.
	maxRecordCount = 10_000_000
)

// Validate checks the per-scenario invariants that a malformed config would
// otherwise surface as a hot-path panic or an OOM. loadConfig calls it through
// ValidateScenarios after unmarshalling; any new entrypoint must do the same.
func (s *Scenario) Validate() error {
	for i, n := range s.SizeBuckets {
		if n < 0 {
			return fmt.Errorf("scenario %q: sizeBuckets[%d] is negative (%d)", s.Name, i, n)
		}
		if n > maxCalldataPadBytes {
			return fmt.Errorf("scenario %q: sizeBuckets[%d]=%d exceeds the 1 MiB (%d-byte) cap", s.Name, i, n, maxCalldataPadBytes)
		}
	}

	if s.RecordCount > maxRecordCount {
		return fmt.Errorf("scenario %q: recordCount=%d exceeds the %d cap", s.Name, s.RecordCount, maxRecordCount)
	}

	// An axis needs both halves to do anything: a sampler and a space to sample.
	// Half-configured, the scenario silently runs its baseline instead of the
	// experiment the operator asked for, which is the worst outcome available to
	// a benchmark. Reject the pairing rather than degenerate.
	if s.KeyDistribution != nil && s.RecordCount == 0 {
		return fmt.Errorf("scenario %q: keyDistribution is set but recordCount is 0, so every tx would target one slot", s.Name)
	}
	if s.KeyDistribution == nil && s.RecordCount != 0 {
		return fmt.Errorf("scenario %q: recordCount is %d but no keyDistribution samples it", s.Name, s.RecordCount)
	}
	if s.SizeDistribution != nil && len(s.SizeBuckets) == 0 {
		return fmt.Errorf("scenario %q: sizeDistribution is set but sizeBuckets is empty, so every tx would send an empty pad", s.Name)
	}
	if s.SizeDistribution == nil && len(s.SizeBuckets) != 0 {
		return fmt.Errorf("scenario %q: sizeBuckets has %d entries but no sizeDistribution samples them", s.Name, len(s.SizeBuckets))
	}
	return s.Operations.validate(s.Name)
}

// ValidateScenarios runs each scenario's Validate and names the scenario that
// failed. loadConfig calls it after unmarshalling.
func (c *LoadConfig) ValidateScenarios() error {
	for i := range c.Scenarios {
		if err := c.Scenarios[i].Validate(); err != nil {
			return err
		}
	}
	return nil
}
