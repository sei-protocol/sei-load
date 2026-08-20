package funder_test

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/sei-protocol/sei-load/config"
	"github.com/sei-protocol/sei-load/funder"
	"github.com/sei-protocol/sei-load/types"
)

// testKey returns a private key and its hex encoding.
func testKey(t *testing.T) (types.Account, string) {
	t.Helper()
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	return types.AccountFromKey(key, false), hex.EncodeToString(crypto.FromECDSA(key))
}

// writeKeyFile writes keyHex to a file the way a mounted secret carries it.
func writeKeyFile(t *testing.T, keyHex string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "root-key.hex")
	require.NoError(t, os.WriteFile(path, []byte(keyHex), 0o600))
	return path
}

// With funding configured the deployer is the root key, whichever way the key
// is supplied and however the secret is padded.
func TestDeployerIsTheFundingRoot(t *testing.T) {
	root, keyHex := testKey(t)

	t.Run("file", func(t *testing.T) {
		cfg := &config.LoadConfig{Funding: &config.FundingConfig{
			RootKeyFile: writeKeyFile(t, keyHex),
		}}
		deployer, err := funder.Deployer(cfg)
		require.NoError(t, err)
		require.Equal(t, root.Address, deployer.Address)
		require.NotNil(t, deployer.PrivKey)
		require.False(t, deployer.Tracked, "the deployer is not a pool account")
	})

	t.Run("file with 0x prefix and trailing newline", func(t *testing.T) {
		cfg := &config.LoadConfig{Funding: &config.FundingConfig{
			RootKeyFile: writeKeyFile(t, "0x"+keyHex+"\n"),
		}}
		deployer, err := funder.Deployer(cfg)
		require.NoError(t, err)
		require.Equal(t, root.Address, deployer.Address)
	})

	t.Run("env", func(t *testing.T) {
		t.Setenv("SEILOAD_TEST_ROOT_KEY", keyHex)
		cfg := &config.LoadConfig{Funding: &config.FundingConfig{
			RootKeyEnv: "SEILOAD_TEST_ROOT_KEY",
		}}
		deployer, err := funder.Deployer(cfg)
		require.NoError(t, err)
		require.Equal(t, root.Address, deployer.Address)
	})
}

// Without funding the deployer is a fresh key: a chain that credits unknown
// senders pays for it, and no two runs share it.
func TestDeployerWithoutFundingIsFresh(t *testing.T) {
	cfg := &config.LoadConfig{}

	first, err := funder.Deployer(cfg)
	require.NoError(t, err)
	second, err := funder.Deployer(cfg)
	require.NoError(t, err)

	require.NotNil(t, first.PrivKey)
	require.NotEqual(t, common.Address{}, first.Address)
	require.NotEqual(t, first.Address, second.Address)
}

// Mock deploy never reaches a chain, so it never reads the root key: a dry run
// works on a host that has no key mounted.
func TestDeployerUnderMockDeployIgnoresTheRootKey(t *testing.T) {
	root, keyHex := testKey(t)
	cfg := &config.LoadConfig{
		MockDeploy: true,
		Funding:    &config.FundingConfig{RootKeyFile: writeKeyFile(t, keyHex) + ".absent"},
	}

	deployer, err := funder.Deployer(cfg)
	require.NoError(t, err)
	require.NotNil(t, deployer.PrivKey)
	require.NotEqual(t, root.Address, deployer.Address)
}

// An unusable root key fails at resolution, where the run can still report it.
func TestDeployerRootKeyErrors(t *testing.T) {
	_, keyHex := testKey(t)

	for _, tc := range []struct {
		name    string
		funding *config.FundingConfig
		want    string
	}{
		{
			name:    "no key source",
			funding: &config.FundingConfig{},
			want:    "no root key",
		},
		{
			name:    "missing file",
			funding: &config.FundingConfig{RootKeyFile: writeKeyFile(t, keyHex) + ".absent"},
			want:    "read rootKeyFile",
		},
		{
			name:    "empty file",
			funding: &config.FundingConfig{RootKeyFile: writeKeyFile(t, "\n")},
			want:    "is empty",
		},
		{
			name:    "not a key",
			funding: &config.FundingConfig{RootKeyFile: writeKeyFile(t, "not-a-key")},
			want:    "parse root key",
		},
		{
			name:    "empty env",
			funding: &config.FundingConfig{RootKeyEnv: "SEILOAD_TEST_ROOT_KEY_UNSET"},
			want:    "is empty",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := funder.Deployer(&config.LoadConfig{Funding: tc.funding})
			require.ErrorContains(t, err, tc.want)
		})
	}
}
