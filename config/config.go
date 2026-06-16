package config

import (
	"encoding/json"
	"fmt"
	"github.com/sei-protocol/sei-load/utils"
	"math/big"
	"time"
)

// LoadConfig stores the configuration for load-related settings.
type LoadConfig struct {
	ChainID int64 `json:"chainId,omitempty"`
	// SeiChainID is the textual chain ID used for tagging metric collection.
	SeiChainID string   `json:"seiChainID,omitempty"`
	Endpoints  []string `json:"endpoints"`
	// Number of shards to divide the senders into.
	// Txs within each shard are sent sequentially.
	// Defaults to Endpoints * Settings.TasksPerEndpoint.
	// WARNING: this is unrelated to the server-side autobahn sharding
	// (which assigns tx sender addrs to lanes). It is solely used to maximize
	// txs/s throughput of the load generator.
	NumShards  utils.Option[int] `json:"numShards,omitzero"`
	Accounts   *AccountConfig    `json:"accounts,omitempty"`
	Scenarios  []Scenario        `json:"scenarios,omitempty"`
	MockDeploy bool              `json:"mockDeploy,omitempty"`
	Settings   *Settings         `json:"settings,omitempty"`
	// Funding, when set, funds the generated account pool from a root key at
	// startup so the run works against a real chain. See funding.go.
	Funding *FundingConfig `json:"funding,omitempty"`
	// Path to write a JSON report of the load test.
	ReportPath string `json:"reportPath,omitempty"`
	// Seed roots the deterministic PRNG sub-streams that drive the run. Same
	// seed + config reproduces the per-stream draw multiset, so the workload
	// (the distribution of keys, sizes, gas, and accounts) is statistically
	// reproducible for fair A/B comparison. Per-tx emission ordering is
	// reproducible only at a single worker; above one worker the multiset still
	// matches but ordering does not, and on-chain arrival order is concurrent
	// regardless. A nil Seed means "unseeded": the generator resolves a random
	// one and records it for after-the-fact replay.
	Seed *uint64 `json:"seed,omitempty"`
}

func (c *LoadConfig) GetNumShards() int {
	return c.NumShards.Or(len(c.Endpoints) * c.Settings.TasksPerEndpoint)
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
}
