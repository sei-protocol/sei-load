package config

import (
	"encoding/json"
	"fmt"
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

// maxCalldataPadBytes caps each SizeBuckets entry at 1 MiB. It catches a config
// typo — a stray extra digit would OOM the generator on the make([]byte, n) hot
// path — and is not a security boundary: configs are author-controlled.
const maxCalldataPadBytes = 1 << 20 // 1 MiB

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
	return nil
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
