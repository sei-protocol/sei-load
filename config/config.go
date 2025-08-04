package config

import (
	"math/big"
	"time"
)

// LoadConfig stores the configuration for load-related settings.
type LoadConfig struct {
	ChainID    int64          `json:"chainId,omitempty"`
	Endpoints  []string       `json:"endpoints"`
	Accounts   *AccountConfig `json:"accounts,omitempty"`
	Scenarios  []Scenario     `json:"scenarios,omitempty"`
	MockDeploy bool           `json:"mockDeploy,omitempty"`
	Settings   *Settings      `json:"settings,omitempty"`
}

// Settings stores CLI-configurable settings that can be specified in config file
type Settings struct {
	Workers           *int           `json:"workers,omitempty"`
	TPS               *float64       `json:"tps,omitempty"`
	StatsInterval     *time.Duration `json:"statsInterval,omitempty"`
	BufferSize        *int           `json:"bufferSize,omitempty"`
	DryRun            *bool          `json:"dryRun,omitempty"`
	Debug             *bool          `json:"debug,omitempty"`
	TrackReceipts     *bool          `json:"trackReceipts,omitempty"`
	TrackBlocks       *bool          `json:"trackBlocks,omitempty"`
	TrackUserLatency  *bool          `json:"trackUserLatency,omitempty"`
	Prewarm           *bool          `json:"prewarm,omitempty"`
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
	Name     string         `json:"name,omitempty"`
	Weight   int            `json:"weight,omitempty"`
	Accounts *AccountConfig `json:"accounts,omitempty"`
}
