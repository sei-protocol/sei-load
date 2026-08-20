package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestScenarioValidateSizeBuckets: a negative pad length (makeslice panic on the
// hot path) and an over-cap pad length (OOM risk) are both rejected; a valid
// histogram, the cap boundary, and an empty/nil bucket list pass.
func TestScenarioValidateSizeBuckets(t *testing.T) {
	t.Parallel()
	t.Run("negative rejected", func(t *testing.T) {
		s := Scenario{Name: "s", SizeDistribution: &Distribution{}, SizeBuckets: []int{0, -1}}
		require.ErrorContains(t, s.Validate(), "negative")
	})
	t.Run("over cap rejected", func(t *testing.T) {
		s := Scenario{Name: "s", SizeDistribution: &Distribution{}, SizeBuckets: []int{maxCalldataPadBytes + 1}}
		require.ErrorContains(t, s.Validate(), "cap")
	})
	t.Run("valid accepted", func(t *testing.T) {
		s := Scenario{Name: "s", SizeDistribution: &Distribution{}, SizeBuckets: []int{0, 64, maxCalldataPadBytes}}
		require.NoError(t, s.Validate())
	})
	t.Run("empty accepted", func(t *testing.T) {
		require.NoError(t, (&Scenario{Name: "s"}).Validate())
	})
}

// TestScenarioValidateRecordCountCap: the keyspace is capped because the zipfian
// sampler precomputes zeta in O(n) under a mutex on the generator's only
// goroutine, so a stray extra digit stalls the run rather than failing it.
func TestScenarioValidateRecordCountCap(t *testing.T) {
	t.Parallel()
	at := Scenario{Name: "s", KeyDistribution: &Distribution{}, RecordCount: maxRecordCount}
	require.NoError(t, at.Validate())
	over := Scenario{Name: "s", KeyDistribution: &Distribution{}, RecordCount: maxRecordCount + 1}
	require.ErrorContains(t, over.Validate(), "cap")
}

// TestScenarioValidateAxisPairing: an axis needs a sampler and a space to sample.
// Configured with only one half it silently runs the baseline instead of the
// experiment, so each direction is rejected.
func TestScenarioValidateAxisPairing(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		scenario Scenario
		wantErr  string
	}{
		"key distribution without a keyspace": {
			Scenario{Name: "s", KeyDistribution: &Distribution{}},
			"recordCount is 0",
		},
		"keyspace without a key distribution": {
			Scenario{Name: "s", RecordCount: 1000},
			"no keyDistribution",
		},
		"size distribution without buckets": {
			Scenario{Name: "s", SizeDistribution: &Distribution{}},
			"sizeBuckets is empty",
		},
		"buckets without a size distribution": {
			Scenario{Name: "s", SizeBuckets: []int{0, 32}},
			"no sizeDistribution",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			require.ErrorContains(t, tc.scenario.Validate(), tc.wantErr)
		})
	}

	both := Scenario{
		Name:             "s",
		KeyDistribution:  &Distribution{},
		RecordCount:      1000,
		SizeDistribution: &Distribution{},
		SizeBuckets:      []int{0, 32},
	}
	require.NoError(t, both.Validate())
}

// TestScenarioValidateOperationsPresentButEmpty: an explicit all-zero mix is a
// misconfiguration, not the default. Omitting the field is the default, and
// Select's zero-total guard stays a safety net for that case rather than a
// swallower of this one.
func TestScenarioValidateOperationsPresentButEmpty(t *testing.T) {
	t.Parallel()
	empty := Scenario{Name: "s", Operations: &OperationMix{}}
	require.ErrorContains(t, empty.Validate(), "every weight is 0")

	require.NoError(t, (&Scenario{Name: "s"}).Validate())
	require.NoError(t, (&Scenario{Name: "s", Operations: &OperationMix{Rmw: 1}}).Validate())
}

// TestValidateScenariosReportsOffendingScenario: validation runs across every
// scenario and names the one that failed, so an operator can find it in a
// multi-scenario profile.
func TestValidateScenariosReportsOffendingScenario(t *testing.T) {
	t.Parallel()
	cfg := LoadConfig{Scenarios: []Scenario{
		{Name: "good"},
		{Name: "bad", Operations: &OperationMix{}},
	}}
	require.ErrorContains(t, cfg.ValidateScenarios(), `scenario "bad"`)

	cfg.Scenarios[1].Operations = &OperationMix{Read: 1}
	require.NoError(t, cfg.ValidateScenarios())
}

// TestParseLoadConfigRejectsUnknownKey: a plain unmarshal drops a key that maps
// to no field, which leaves that field at its default and makes a mistyped
// profile look like a successful run. Strict parsing names the key instead, at
// every nesting level, and names the scenario that carries it.
func TestParseLoadConfigRejectsUnknownKey(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		profile  string
		wantErrs []string
	}{
		"top level": {
			`{"chainId":1,"seeds":42}`,
			[]string{`"seeds"`},
		},
		"settings": {
			`{"settings":{"tps":10,"workers":50}}`,
			[]string{`"workers"`},
		},
		"accounts": {
			`{"accounts":{"counts":500}}`,
			[]string{`"counts"`},
		},
		"funding": {
			`{"funding":{"rootKey":"/etc/key.hex"}}`,
			[]string{`"rootKey"`},
		},
		"scenario": {
			`{"scenarios":[{"name":"StorageRW","operation":{"read":1}}]}`,
			[]string{`scenario "StorageRW"`, `"operation"`},
		},
		"operation mix": {
			`{"scenarios":[{"name":"StorageRW","operations":{"reads":1}}]}`,
			[]string{`scenario "StorageRW"`, `"reads"`},
		},
		"later scenario": {
			`{"scenarios":[{"name":"good"},{"name":"bad","sizeBucket":[64]}]}`,
			[]string{`scenario "bad"`, `"sizeBucket"`},
		},
		"unnamed scenario": {
			`{"scenarios":[{"weight":1,"sizeBucket":[64]}]}`,
			[]string{"scenario at index 0", `"sizeBucket"`},
		},
		"trailing object": {
			`{"chainId":1} {"chainId":2}`,
			[]string{"unexpected data after the top-level JSON value"},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseLoadConfig([]byte(tc.profile))
			require.Error(t, err)
			for _, want := range tc.wantErrs {
				require.ErrorContains(t, err, want)
			}
		})
	}
}

// TestParseLoadConfigKeepsEveryField: scenarios decode one at a time so an
// unknown key can be named against its scenario, so the parse must still carry
// every field of a well-formed profile.
func TestParseLoadConfigKeepsEveryField(t *testing.T) {
	t.Parallel()

	cfg, err := ParseLoadConfig([]byte(`{
		"chainId": 713714,
		"seiChainID": "sei-chain",
		"endpoints": ["http://127.0.0.1:8545"],
		"accounts": {"count": 500, "newAccountRate": 0.0},
		"funding": {"rootKeyFile": "/etc/key.hex", "batchSize": 200},
		"seed": 42,
		"scenarios": [
			{"name": "EVMTransfer", "weight": 7},
			{
				"name": "StorageRW",
				"weight": 3,
				"keyDistribution": {"Name": "zipfian", "theta": 0.9},
				"recordCount": 1000,
				"sizeDistribution": {"Name": "uniform"},
				"sizeBuckets": [0, 64],
				"operations": {"read": 50, "write": 30, "rmw": 20}
			}
		],
		"settings": {"tps": 10, "statsInterval": "10s", "bufferSize": 1000, "prewarm": true}
	}`))
	require.NoError(t, err)
	require.NoError(t, cfg.ValidateScenarios())

	require.Equal(t, int64(713714), cfg.ChainID)
	require.Equal(t, "sei-chain", cfg.SeiChainID)
	require.Equal(t, []string{"http://127.0.0.1:8545"}, cfg.Endpoints)
	require.Equal(t, 500, cfg.Accounts.Accounts)
	require.Equal(t, "/etc/key.hex", cfg.Funding.RootKeyFile)
	require.Equal(t, uint64(42), *cfg.Seed)
	require.Equal(t, 10.0, cfg.Settings.TPS)
	require.True(t, cfg.Settings.Prewarm)

	require.Len(t, cfg.Scenarios, 2)
	require.Equal(t, "EVMTransfer", cfg.Scenarios[0].Name)
	storage := cfg.Scenarios[1]
	require.Equal(t, "zipfian", storage.KeyDistribution.Name())
	require.Equal(t, uint64(1000), storage.RecordCount)
	require.Equal(t, "uniform", storage.SizeDistribution.Name())
	require.Equal(t, []int{0, 64}, storage.SizeBuckets)
	require.Equal(t, &OperationMix{Read: 50, Write: 30, Rmw: 20}, storage.Operations)
}

// TestParseLoadConfigAcceptsUnknownKeyInsideTaggedObject: Distribution and
// GasPicker parse their own payload through UnmarshalJSON, which the strict
// decoder does not reach into. A misspelled theta leaves the sampler at theta 0,
// which draws uniformly. This is the one nesting level strict parsing does not
// cover.
func TestParseLoadConfigAcceptsUnknownKeyInsideTaggedObject(t *testing.T) {
	t.Parallel()

	cfg, err := ParseLoadConfig([]byte(
		`{"scenarios":[{"name":"s","keyDistribution":{"Name":"zipfian","thet":0.9},"recordCount":1000}]}`))
	require.NoError(t, err)

	zipfian, ok := cfg.Scenarios[0].KeyDistribution.delegate.(*ZipfianDistribution)
	require.True(t, ok)
	require.Zero(t, zipfian.Theta)
}
