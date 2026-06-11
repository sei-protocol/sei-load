package config

import (
	"encoding/json"
	"math/big"
	"testing"
)

func TestBigIntUnmarshal(t *testing.T) {
	var b BigInt
	if err := json.Unmarshal([]byte(`"1000000000000000000"`), &b); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if (*big.Int)(&b).String() != "1000000000000000000" {
		t.Fatalf("got %s", (*big.Int)(&b).String())
	}
	// Above 2^53 — must survive as exact integer (the reason for string-based JSON).
	if (*big.Int)(&b).Cmp(big.NewInt(1<<53)) <= 0 {
		t.Fatalf("expected value above 2^53")
	}
	if err := json.Unmarshal([]byte(`"not-a-number"`), &b); err == nil {
		t.Fatal("expected error on non-numeric string")
	}
	if err := json.Unmarshal([]byte(`123`), &b); err == nil {
		t.Fatal("expected error on bare JSON number (must be a string)")
	}
}

func TestFundingDefaults(t *testing.T) {
	var fc *FundingConfig
	if fc.FundAmount().String() != "1000000000000000000" {
		t.Fatalf("nil funding default amount = %s", fc.FundAmount().String())
	}
	if fc.Batch() != DefaultFundBatchSize {
		t.Fatalf("nil funding default batch = %d", fc.Batch())
	}
	fc = &FundingConfig{}
	if fc.FundAmount().String() != "1000000000000000000" {
		t.Fatalf("empty funding default amount = %s", fc.FundAmount().String())
	}
	fc.BatchSize = 50
	if fc.Batch() != 50 {
		t.Fatalf("batch override = %d", fc.Batch())
	}
}

func TestValidateFunding(t *testing.T) {
	cases := []struct {
		name    string
		cfg     LoadConfig
		wantErr bool
	}{
		{
			name: "no funding is fine",
			cfg:  LoadConfig{Accounts: &AccountConfig{Accounts: 10, NewAccountRate: 0.5}},
		},
		{
			name:    "funding without a key errors",
			cfg:     LoadConfig{Funding: &FundingConfig{}, Accounts: &AccountConfig{Accounts: 10}},
			wantErr: true,
		},
		{
			name: "funding with env key and rate 0 is fine",
			cfg: LoadConfig{
				Funding:  &FundingConfig{RootKeyEnv: "K"},
				Accounts: &AccountConfig{Accounts: 10, NewAccountRate: 0},
			},
		},
		{
			name: "funding with newAccountRate>0 on top-level pool errors",
			cfg: LoadConfig{
				Funding:  &FundingConfig{RootKey: "0xabc"},
				Accounts: &AccountConfig{Accounts: 10, NewAccountRate: 0.1},
			},
			wantErr: true,
		},
		{
			name: "funding with newAccountRate>0 on a scenario pool errors",
			cfg: LoadConfig{
				Funding:   &FundingConfig{RootKey: "0xabc"},
				Scenarios: []Scenario{{Name: "EVMTransfer", Accounts: &AccountConfig{Accounts: 5, NewAccountRate: 0.2}}},
			},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.ValidateFunding()
			if tc.wantErr != (err != nil) {
				t.Fatalf("wantErr=%v got err=%v", tc.wantErr, err)
			}
		})
	}
}
