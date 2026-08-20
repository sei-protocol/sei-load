package generator_test

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
	"github.com/sei-protocol/sei-load/generator"
	"github.com/sei-protocol/sei-load/generator/scenarios"
	"github.com/sei-protocol/sei-load/types"
)

// contractConfig is a two-contract profile against chain: the case every
// committed profile leaves untested, since the one profile that configures
// funding runs EVMTransfer only.
func contractConfig(chain *mockChain) *config.LoadConfig {
	return &config.LoadConfig{
		ChainID:   7777,
		Endpoints: []string{chain.url},
		Accounts:  &config.AccountConfig{Accounts: 4},
		Scenarios: []config.Scenario{
			{Name: scenarios.StorageRW, Weight: 1},
			{Name: scenarios.ERC20, Weight: 1},
		},
	}
}

// rootKeyFile writes a hex private key the way a mounted secret carries it:
// with a trailing newline.
func rootKeyFile(t *testing.T) (types.Account, string) {
	t.Helper()
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "root-key.hex")
	require.NoError(t, os.WriteFile(path, []byte(hex.EncodeToString(crypto.FromECDSA(key))+"\n"), 0o600))
	return types.AccountFromKey(key, false), path
}

// The generator deploys from the account it is handed, at the nonce that
// account has reached on chain — not from a key it mints, and not at the
// scenario's index.
func TestDeployFromReceivedDeployerAtChainNonce(t *testing.T) {
	deployer := types.NewAccount(false)
	const startNonce = 41
	chain := newMockChain(t, mockChainConfig{
		baseNonce: map[common.Address]uint64{deployer.Address: startNonce},
	})

	cfg := contractConfig(chain)
	rng := newTestRng(1)
	gen, err := generator.NewGenerator(t.Context(), rng, cfg, deployer)
	require.NoError(t, err)

	// One creation per contract scenario, in order, from the deployer.
	require.Equal(t, []uint64{startNonce, startNonce + 1}, chain.noncesFrom(deployer.Address))
	for _, tx := range chain.txsFrom(deployer.Address) {
		require.Nil(t, tx.To(), "a deployment is a contract creation")
	}

	// The scenarios generate against the contracts those creations produced.
	deployed := map[common.Address]bool{
		crypto.CreateAddress(deployer.Address, startNonce):   false,
		crypto.CreateAddress(deployer.Address, startNonce+1): false,
	}
	for _, tx := range generateN(t, rng, gen, 20) {
		to := tx.EthTx.To()
		require.NotNil(t, to)
		_, ok := deployed[*to]
		require.True(t, ok, "tx targets %s, which no deployment created", to.Hex())
		deployed[*to] = true
	}
	for address, used := range deployed {
		require.True(t, used, "no tx targets the contract at %s", address.Hex())
	}
}

// A deployment that mines with a failed status fails the run, and says why.
func TestDeployFailureIsAnError(t *testing.T) {
	deployer := types.NewAccount(false)
	chain := newMockChain(t, mockChainConfig{revertDeployments: true})

	_, err := generator.NewGenerator(t.Context(), newTestRng(1), contractConfig(chain), deployer)
	require.ErrorContains(t, err, "failed to deploy scenarios")
	require.ErrorContains(t, err, scenarios.StorageRW)
	require.ErrorContains(t, err, "deployment transaction failed with status 0")
}

// A live deployment needs a key to sign with, and says so when it has none.
func TestDeployWithoutKeyIsAnError(t *testing.T) {
	chain := newMockChain(t, mockChainConfig{})

	_, err := generator.NewGenerator(t.Context(), newTestRng(1), contractConfig(chain), types.Account{})
	require.ErrorContains(t, err, "deployer has no private key")
	require.Zero(t, chain.txCount(), "nothing reaches the chain unsigned")
}

// Funding plus contract scenarios: the deployer is the funding root, and the
// scenario deployments and the funder's own transactions form one ordered nonce
// stream on that key.
func TestFundedRunSharesOneNonceStream(t *testing.T) {
	root, keyPath := rootKeyFile(t)
	const startNonce = 7
	chain := newMockChain(t, mockChainConfig{
		baseNonce: map[common.Address]uint64{root.Address: startNonce},
	})

	cfg := contractConfig(chain)
	cfg.Funding = &config.FundingConfig{RootKeyFile: keyPath, BatchSize: 2}

	deployer, err := funder.Deployer(cfg)
	require.NoError(t, err)
	require.Equal(t, root.Address, deployer.Address, "the funded root is the deployer")

	rng := newTestRng(1)
	gen, err := generator.NewGenerator(t.Context(), rng, cfg, deployer)
	require.NoError(t, err)

	var addrs []common.Address
	for _, account := range gen.Accounts() {
		addrs = append(addrs, account.Address)
	}
	require.Len(t, addrs, 4)
	require.NoError(t, funder.FundAccounts(t.Context(), cfg, addrs))

	// 2 scenario deployments, then the funder's Disperse deployment and 2
	// disperseEther batches for 4 accounts, each at the next nonce.
	require.Equal(t,
		[]uint64{startNonce, startNonce + 1, startNonce + 2, startNonce + 3, startNonce + 4},
		chain.noncesFrom(root.Address),
	)

	// Only the deployments create contracts; the batches call the deployed one.
	txs := chain.txsFrom(root.Address)
	for _, tx := range txs[:3] {
		require.Nil(t, tx.To())
	}
	disperse := crypto.CreateAddress(root.Address, startNonce+2)
	for _, tx := range txs[3:] {
		require.Equal(t, disperse, *tx.To())
	}
}
