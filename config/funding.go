package config

import (
	"encoding/json"
	"fmt"
	"math/big"
)

// DefaultFundAmountWei is the per-account funding when FundAmountWei is unset.
// 1e18 wei = 1e6 usei = 1 SEI (the EVM surface is 18-dec; bank-backed in 1e12
// chunks, so a whole-SEI amount carries no sub-usei dust).
var DefaultFundAmountWei = func() *BigInt {
	b := new(big.Int)
	b.SetString("1000000000000000000", 10)
	return (*BigInt)(b)
}()

// DefaultFundBatchSize is the recipients-per-disperseEther call when unset.
const DefaultFundBatchSize = 200

// FundingConfig configures root-key funding of the generated account pool so
// seiload can run against a real chain (where accounts are not auto-funded by
// mock_balances or genesis). When nil, accounts are left unfunded — the
// existing mock/genesis behavior is unchanged.
type FundingConfig struct {
	// RootKeyEnv names the env var holding the funded root account's hex ECDSA
	// private key. Preferred over RootKey so the key never lands in a config
	// file or log line.
	RootKeyEnv string `json:"rootKeyEnv,omitempty"`
	// RootKey is an inline hex private key, used only when RootKeyEnv is empty.
	// Avoid in committed configs.
	RootKey string `json:"rootKey,omitempty"`
	// FundAmountWei is the per-account funding in wei. Defaults to 1 SEI.
	FundAmountWei *BigInt `json:"fundAmountWei,omitempty"`
	// BatchSize is the number of recipients per disperseEther call.
	BatchSize int `json:"batchSize,omitempty"`
	// DisperseAddress, when set, reuses a pre-deployed Disperse contract instead
	// of deploying a fresh one (saves a deploy tx and avoids needing the root to
	// deploy on every restart).
	DisperseAddress string `json:"disperseAddress,omitempty"`
}

// FundAmount returns the configured per-account amount or the default.
func (f *FundingConfig) FundAmount() *big.Int {
	if f == nil || f.FundAmountWei == nil {
		return (*big.Int)(DefaultFundAmountWei)
	}
	return (*big.Int)(f.FundAmountWei)
}

// Batch returns the configured batch size or the default.
func (f *FundingConfig) Batch() int {
	if f == nil || f.BatchSize <= 0 {
		return DefaultFundBatchSize
	}
	return f.BatchSize
}

// BigInt wraps big.Int with string-based JSON (un)marshaling, since JSON
// numbers lose precision above 2^53 and wei amounts exceed that.
type BigInt big.Int

// UnmarshalJSON accepts a decimal string (e.g. "1000000000000000000").
func (b *BigInt) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("fundAmountWei must be a decimal string: %w", err)
	}
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return fmt.Errorf("invalid integer %q", s)
	}
	*b = BigInt(*v)
	return nil
}

// MarshalJSON renders the value as a decimal string.
func (b *BigInt) MarshalJSON() ([]byte, error) {
	return json.Marshal((*big.Int)(b).String())
}

// Validate checks funding invariants against the rest of the config. It must be
// called after the config is loaded.
func (c *LoadConfig) ValidateFunding() error {
	if c.Funding == nil {
		return nil
	}
	if c.Funding.RootKeyEnv == "" && c.Funding.RootKey == "" {
		return fmt.Errorf("funding: one of rootKeyEnv or rootKey is required")
	}
	// Newly-minted accounts (newAccountRate > 0) are never funded — their first
	// tx would fail for lack of gas. Funding requires a fixed, fully-funded pool.
	bad := func(a *AccountConfig, where string) error {
		if a != nil && a.NewAccountRate > 0 {
			return fmt.Errorf("funding requires newAccountRate=0 (%s has %.3f); "+
				"on-demand accounts cannot be funded", where, a.NewAccountRate)
		}
		return nil
	}
	if err := bad(c.Accounts, "accounts"); err != nil {
		return err
	}
	for i := range c.Scenarios {
		if err := bad(c.Scenarios[i].Accounts, "scenario "+c.Scenarios[i].Name); err != nil {
			return err
		}
	}
	return nil
}
