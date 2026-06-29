package types

import (
	"math/big"
	mrand "math/rand/v2"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func newTestRng(seed uint64) *mrand.Rand {
	return mrand.New(mrand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
}

func TestNewAccount(t *testing.T) {
	account := NewAccount(true)

	require.NotEqual(t, common.Address{}, account.Address)
	require.NotNil(t, account.PrivKey)
	require.True(t, account.Tracked)
	require.Equal(t, crypto.PubkeyToAddress(account.PrivKey.PublicKey), account.Address)
}

func TestGenerateAccounts(t *testing.T) {
	accounts := GenerateAccounts(10, false)

	require.Len(t, accounts, 10)
	seen := map[common.Address]bool{}
	for _, account := range accounts {
		require.False(t, account.Tracked)
		require.NotEqual(t, common.Address{}, account.Address)
		require.NotNil(t, account.PrivKey)
		require.False(t, seen[account.Address], "duplicate account %s", account.Address.Hex())
		seen[account.Address] = true
	}
}

func TestAccountPoolRoundRobin(t *testing.T) {
	pool := NewAccountPool(3, 0)
	accounts := pool.Accounts()
	rng := newTestRng(1)

	require.Len(t, accounts, 3)
	require.Equal(t, accounts[1].Address, pool.NextAccount(rng).Address)
	require.Equal(t, accounts[2].Address, pool.NextAccount(rng).Address)
	require.Equal(t, accounts[0].Address, pool.NextAccount(rng).Address)
}

func TestAccountPoolAlwaysCreatesNewAccounts(t *testing.T) {
	pool := NewAccountPool(2, 1.0)
	original := map[common.Address]bool{}
	for _, account := range pool.Accounts() {
		original[account.Address] = true
	}

	rng := newTestRng(1)
	for range 10 {
		account := pool.NextAccount(rng)
		require.False(t, account.Tracked)
		require.False(t, original[account.Address], "expected a fresh account")
	}
}

func TestAccountPoolMixedRate(t *testing.T) {
	pool := NewAccountPool(5, 0.5)
	original := map[common.Address]bool{}
	for _, account := range pool.Accounts() {
		original[account.Address] = true
	}

	const iterations = 100
	originalCount := 0
	newCount := 0
	rng := newTestRng(1)

	for i := 0; i < iterations; i++ {
		account := pool.NextAccount(rng)
		if original[account.Address] {
			originalCount++
		} else {
			newCount++
		}
	}

	// Seeded: the split is exact and reproducible, not probabilistic. Re-running
	// the same seeded pool must reproduce these counts. If the frozen derivation
	// changes, these expected values change with it.
	const expectedNew = 52
	require.Equal(t, expectedNew, newCount, "seeded new-account count is not reproducible")
	require.Equal(t, iterations, originalCount+newCount, "Total accounts don't match iterations")
}

func TestCreateTxFromEthTx(t *testing.T) {
	sender := NewAccount(true)
	receiver := common.HexToAddress("0x1234567890123456789012345678901234567890")
	scenario := &TxScenario{
		Name:     "test",
		Nonce:    42,
		Sender:   sender,
		Receiver: receiver,
	}
	tx := ethtypes.NewTx(&ethtypes.DynamicFeeTx{
		ChainID:   big.NewInt(713714),
		Nonce:     scenario.Nonce,
		GasTipCap: big.NewInt(2_000_000_000),
		GasFeeCap: big.NewInt(20_000_000_000),
		Gas:       21_000,
		To:        &receiver,
		Value:     big.NewInt(1),
	})

	loadTx := CreateTxFromEthTx(tx, scenario)

	require.Equal(t, tx, loadTx.EthTx)
	require.Equal(t, scenario, loadTx.Scenario)
	require.Zero(t, loadTx.SequenceIndex)
	require.True(t, loadTx.IntendedSendTime.IsZero())
	require.True(t, loadTx.AttemptedSendTime.IsZero())
	require.True(t, loadTx.InclusionTime.IsZero())
}

func TestLoadTxLifecycleFieldsDefaultToZero(t *testing.T) {
	loadTx := &LoadTx{}
	require.True(t, loadTx.IntendedSendTime.Equal(time.Time{}))
	require.True(t, loadTx.AttemptedSendTime.Equal(time.Time{}))
	require.True(t, loadTx.InclusionTime.Equal(time.Time{}))
}
