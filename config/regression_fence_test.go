package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

// PLT-464: regression fence for the additive-by-construction promise. The
// workload-modeler work (arrival_model, key/size distributions, max-in-flight)
// is additive: every field is omitempty and every new knob defaults to the
// legacy behavior. These tests are the CI gate that the promise holds against
// the *current* profiles/ — they fail if a shipped profile stops parsing or if
// a config with none of the new fields stops resolving to the legacy path.

// resolveProfile runs a profile JSON through the same two-stage path the binary
// uses (see main.loadConfig + runLoadTest): unmarshal the file into LoadConfig,
// then resolve Settings through the real exported viper path where the additive
// defaults are applied. main.loadConfig is unexported, so the raw unmarshal is
// restated here; the settings resolution — the stage that supplies the legacy
// defaults — is the binary's own LoadSettings/ResolveSettings.
//
// viper is a process-global; callers must hold it for the duration (no t.Parallel).
func resolveProfile(t *testing.T, data []byte) *LoadConfig {
	t.Helper()

	var cfg LoadConfig
	require.NoError(t, json.Unmarshal(data, &cfg), "unmarshal profile into LoadConfig")

	viper.Reset()
	cmd := &cobra.Command{Use: "test"}
	registerSettingsFlags(cmd)
	require.NoError(t, InitializeViper(cmd), "initialize viper")
	require.NoError(t, LoadSettings(cfg.Settings), "load settings")
	cfg.Settings = ResolveSettings()
	return &cfg
}

// registerSettingsFlags declares the flags InitializeViper binds to, with
// zero-value defaults so the config file / viper defaults win (mirrors the
// flag set in main and the precedence setup in settings_test.go).
func registerSettingsFlags(cmd *cobra.Command) {
	cmd.Flags().Duration("stats-interval", 0, "")
	cmd.Flags().Int("workers", 0, "")
	cmd.Flags().Float64("tps", 0, "")
	cmd.Flags().Bool("dry-run", false, "")
	cmd.Flags().Bool("debug", false, "")
	cmd.Flags().Bool("track-receipts", false, "")
	cmd.Flags().Bool("track-blocks", false, "")
	cmd.Flags().Bool("prewarm", false, "")
	cmd.Flags().Bool("track-user-latency", false, "")
	cmd.Flags().Int("buffer-size", 0, "")
	cmd.Flags().Bool("ramp-up", false, "")
	cmd.Flags().String("report-path", "", "")
	cmd.Flags().String("txs-dir", "", "")
	cmd.Flags().Uint64("target-gas", 0, "")
	cmd.Flags().Int("num-blocks-to-write", 0, "")
	cmd.Flags().Duration("post-summary-flush-delay", 0, "")
	cmd.Flags().String("arrival-model", "", "")
	cmd.Flags().Int("max-in-flight", 0, "")
}

// profileFiles discovers every shipped profile so the fence covers new profiles
// automatically rather than tracking a hardcoded list.
func profileFiles(t *testing.T) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "profiles", "*.json"))
	require.NoError(t, err, "glob profiles")
	require.NotEmpty(t, files, "no profiles found under ../profiles")
	return files
}

// TestProfilesResolveThroughBinaryPath proves every shipped profile still
// parses and resolves through the binary's settings path — the parse half of
// the additive-by-construction promise. Unlike profiles/profiles_test.go (raw
// unmarshal only), this exercises LoadSettings/ResolveSettings/Validate, so a
// profile that parses but resolves to an invalid settings combination also
// fails the fence.
func TestProfilesResolveThroughBinaryPath(t *testing.T) {
	for _, path := range profileFiles(t) {
		name := filepath.Base(path)
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(path)
			require.NoError(t, err, "read profile")

			cfg := resolveProfile(t, data)
			require.NotNil(t, cfg.Settings, "%s: settings nil after resolve", name)
			require.NoError(t, cfg.Settings.Validate(), "%s: resolved settings invalid", name)
		})
	}
}

// TestProfilesNoNewFieldsResolveToLegacyPath is the behavioral half of the
// promise: every shipped profile that sets none of the additive fields must
// resolve to the legacy arrival model and carry no distributions, so the binary
// takes the legacy closed-loop / round-robin path (main: openLoop is false ⇒
// the dispatcher keeps its NewDispatcher default, ArrivalClosedLoop).
//
// Today every profile is a no-new-fields profile; if one starts opting into the
// new model it is filtered out here rather than failing — the fence guards the
// legacy default for legacy configs, not that profiles never adopt new fields.
func TestProfilesNoNewFieldsResolveToLegacyPath(t *testing.T) {
	legacyArrivalModel := DefaultSettings().ArrivalModel
	require.Equal(t, ArrivalModelClosedLoop, legacyArrivalModel,
		"legacy default drifted: closed-loop is the additive baseline")

	checked := 0
	for _, path := range profileFiles(t) {
		name := filepath.Base(path)
		data, err := os.ReadFile(path)
		require.NoError(t, err, "read profile")

		if usesNewFields(t, data) {
			t.Logf("%s opts into new fields; skipping legacy-default assertion", name)
			continue
		}
		checked++

		t.Run(name, func(t *testing.T) {
			cfg := resolveProfile(t, data)

			require.Equal(t, legacyArrivalModel, cfg.Settings.ArrivalModel,
				"%s: no arrival_model set must resolve to the legacy default", name)

			for i, sc := range cfg.Scenarios {
				require.Nil(t, sc.KeyDistribution,
					"%s: scenario %d (%s) has a key distribution but set no distribution config", name, i, sc.Name)
				require.Nil(t, sc.SizeDistribution,
					"%s: scenario %d (%s) has a size distribution but set no distribution config", name, i, sc.Name)
			}
		})
	}
	require.Positive(t, checked, "expected at least one no-new-fields profile to fence the legacy path")
}

// TestNoNewFieldsConfigResolvesToLegacyDefaults pins the additive contract on a
// synthetic minimal config (independent of which profiles happen to ship): a
// config with no arrival_model and no distributions resolves to the legacy
// default and leaves distributions nil.
func TestNoNewFieldsConfigResolvesToLegacyDefaults(t *testing.T) {
	const minimal = `{
		"endpoints": ["http://localhost:8545"],
		"scenarios": [{"name": "EVMTransfer", "weight": 1}],
		"settings": {"workers": 1}
	}`

	cfg := resolveProfile(t, []byte(minimal))

	require.Equal(t, DefaultSettings().ArrivalModel, cfg.Settings.ArrivalModel,
		"no arrival_model must resolve to the legacy default")
	require.Equal(t, ArrivalModelClosedLoop, cfg.Settings.ArrivalModel,
		"legacy default is closed-loop")
	require.NoError(t, cfg.Settings.Validate(), "legacy-default settings must be valid")

	for i, sc := range cfg.Scenarios {
		require.Nil(t, sc.KeyDistribution, "scenario %d: no distribution config must leave KeyDistribution nil", i)
		require.Nil(t, sc.SizeDistribution, "scenario %d: no distribution config must leave SizeDistribution nil", i)
	}
}

// usesNewFields reports whether a profile opts into any additive field at the
// JSON layer, so the legacy-default assertion only fences genuine legacy
// configs. Distribution is checked at the raw-JSON layer because the Distribution
// type unmarshals an empty object into a usable (zero-value) sampler.
func usesNewFields(t *testing.T, data []byte) bool {
	t.Helper()
	var probe struct {
		Scenarios []struct {
			KeyDistribution  *json.RawMessage `json:"keyDistribution"`
			SizeDistribution *json.RawMessage `json:"sizeDistribution"`
		} `json:"scenarios"`
		Settings struct {
			ArrivalModel string           `json:"arrivalModel"`
			MaxInFlight  *json.RawMessage `json:"maxInFlight"`
		} `json:"settings"`
	}
	require.NoError(t, json.Unmarshal(data, &probe), "probe profile for new fields")

	if probe.Settings.ArrivalModel != "" || probe.Settings.MaxInFlight != nil {
		return true
	}
	for _, sc := range probe.Scenarios {
		if sc.KeyDistribution != nil || sc.SizeDistribution != nil {
			return true
		}
	}
	return false
}
