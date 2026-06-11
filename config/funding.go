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
// seiload can run against a real chain. When nil, accounts are left unfunded.
//
// The root account must be funded at its EVM address (the bech32 cast of the
// key's 0x address) or already associated; its first EVM tx auto-associates so
// the balance becomes EVM-spendable.
type FundingConfig struct {
	// RootKeyFile is a path to a file containing the funded root account's hex
	// ECDSA private key. Preferred over RootKeyEnv: a mounted file is not
	// exposed in the process environment, /proc/<pid>/environ, or child procs.
	RootKeyFile string `json:"rootKeyFile,omitempty"`
	// RootKeyEnv names an env var holding the hex key. Fallback when RootKeyFile
	// is unset.
	RootKeyEnv string `json:"rootKeyEnv,omitempty"`
	// FundAmountWei is the per-account funding in wei. Defaults to 1 SEI.
	FundAmountWei *BigInt `json:"fundAmountWei,omitempty"`
	// BatchSize is the recipients per disperseEther call; defaults to DefaultFundBatchSize.
	BatchSize int `json:"batchSize,omitempty"`
}

// FundAmount returns the configured per-account amount or the default.
func (f *FundingConfig) FundAmount() *big.Int {
	if f == nil || f.FundAmountWei == nil {
		return DefaultFundAmountWei.ToBigInt()
	}
	return f.FundAmountWei.ToBigInt()
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

func (b *BigInt) ToBigInt() *big.Int { return (*big.Int)(b) }

// UnmarshalJSON accepts a decimal string (e.g. "1000000000000000000").
func (b *BigInt) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("BigInt must be a decimal string: %w", err)
	}
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return fmt.Errorf("BigInt: invalid decimal string %q", s)
	}
	*b = BigInt(*v)
	return nil
}

// MarshalJSON renders the value as a decimal string.
func (b *BigInt) MarshalJSON() ([]byte, error) {
	return json.Marshal((*big.Int)(b).String())
}

// ValidateFunding checks funding invariants against the rest of the config. It
// must be called after the config is loaded.
func (c *LoadConfig) ValidateFunding() error {
	if c.Funding == nil {
		return nil
	}
	if c.Funding.RootKeyFile == "" && c.Funding.RootKeyEnv == "" {
		return fmt.Errorf("funding: one of rootKeyFile or rootKeyEnv is required")
	}
	// Newly-minted accounts (newAccountRate > 0) are never funded — their first
	// tx would fail for lack of gas. Funding requires a fixed pool.
	bad := func(a *AccountConfig, where string) error {
		if a != nil && a.NewAccountRate > 0 {
			return fmt.Errorf("funding: requires newAccountRate=0 (%s has %.3f); "+
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
