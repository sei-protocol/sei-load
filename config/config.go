package config

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
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
	// Path to write a JSON report of the load test.
	ReportPath string `json:"reportPath,omitempty"`
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
	// DeterministicEvmStressKeys uses types.EvmStressPrivateKey (sender indices
	// 1..count). Fund those Cosmos accounts in genesis to match the pool.
	DeterministicEvmStressKeys bool `json:"deterministicEvmStressKeys,omitempty"`
	// SingleUseSenders gives each pooled account at most one turn as sender;
	// when exhausted, generation stops (ok=false). One transaction per pooled sender.
	// Incompatible with newAccountRate > 0 (validated at pool creation) and with
	// settings.prewarm (validated at startup via ValidatePrewarmAccountPools).
	SingleUseSenders bool `json:"singleUseSenders,omitempty"`
}

// Scenario represents each scenario in the load configuration.
type Scenario struct {
	Name            string         `json:"name,omitempty"`
	Weight          int            `json:"weight,omitempty"`
	Accounts        *AccountConfig `json:"accounts,omitempty"`
	GasPicker       *GasPicker     `json:"gasPicker,omitempty"`
	GasFeeCapPicker *GasPicker     `json:"gasFeeCapPicker,omitempty"`
	GasTipCapPicker *GasPicker     `json:"gasTipCapPicker,omitempty"`
	// FixedReceiver is an optional hex EVM address. When set, all transactions
	// in this scenario are sent to this single address (single-recipient stress
	// mode). If empty, the receiver is picked from the account pool as usual.
	FixedReceiver string `json:"fixedReceiver,omitempty"`
}

// ValidatePrewarmAccountPools returns an error if prewarm is enabled while any
// account pool uses singleUseSenders. Prewarm iterates the same pools and is
// incompatible with single-use exhaustion semantics.
func ValidatePrewarmAccountPools(cfg *LoadConfig, prewarm bool) error {
	if !prewarm || cfg == nil {
		return nil
	}
	if cfg.Accounts != nil && cfg.Accounts.SingleUseSenders {
		return fmt.Errorf("settings.prewarm cannot be used with accounts.singleUseSenders (prewarm shares the same account pool)")
	}
	for i, sc := range cfg.Scenarios {
		if sc.Accounts != nil && sc.Accounts.SingleUseSenders {
			return fmt.Errorf("settings.prewarm cannot be used with scenarios[%d] (%q) accounts.singleUseSenders", i, sc.Name)
		}
	}
	return nil
}

// ValidateFixedReceiverAddresses returns an error if any scenario has a
// non-empty fixedReceiver that is not a valid 20-byte hex address (0x + 40 hex).
// Malformed values would otherwise be passed to common.HexToAddress and can
// map to the zero address without an obvious failure at config load time.
func ValidateFixedReceiverAddresses(cfg *LoadConfig) error {
	if cfg == nil {
		return nil
	}
	for i, sc := range cfg.Scenarios {
		addr := strings.TrimSpace(sc.FixedReceiver)
		if addr == "" {
			continue
		}
		if !common.IsHexAddress(addr) {
			return fmt.Errorf(
				"scenarios[%d] (%q): invalid fixedReceiver %q (want 0x-prefixed 40 hex characters)",
				i, sc.Name, sc.FixedReceiver,
			)
		}
	}
	return nil
}
