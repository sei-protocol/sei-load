package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestLoadConfigParsesStrictly asserts that the run entrypoint rejects a key the
// config does not define. config.TestParseLoadConfigRejectsUnknownKey covers
// which keys and which nesting levels; this covers only that loadConfig routes
// through that parser rather than a lenient unmarshal.
func TestLoadConfigParsesStrictly(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "profile.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"chainId":1,"endpoints":["http://127.0.0.1:8545"],
		"scenarios":[{"name":"EVMTransfer","weight":1}],
		"settings":{"tps":10,"workers":50}}`), 0o600))

	_, err := loadConfig(path)
	require.ErrorContains(t, err, `"workers"`)
}
