package types

import (
	"math/big"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"

	"github.com/sei-protocol/sei-load/utils/rng"
	"github.com/sei-protocol/sei-load/utils/require"
)

func TestNewAccount(t *testing.T) {
	account := NewAccount()
	require.NotNil(t, account)

	// Verify account has valid address and private key
	require.NotEqual(t, common.Address{}, account.Address)
	require.NotNil(t, account.PrivKey)

	// Verify address matches private key
	expectedAddress := crypto.PubkeyToAddress(account.PrivKey.PublicKey)
	require.Equal(t, expectedAddress, account.Address)

	// Verify initial nonce is 0
	require.Equal(t, uint64(0), account.Nonce.Load())
}

func TestAccountNonceManagement(t *testing.T) {
	account := NewAccount()

	// Test sequential nonce increments
	for i := range uint64(10) {
		nonce := account.GetAndIncrementNonce()
		require.Equal(t, i, nonce)
	}

	// Verify final nonce value
	require.Equal(t, 10, account.Nonce.Load())
}

func TestAccountNonceConcurrency(t *testing.T) {
	account := NewAccount()

	const numGoroutines = 100
	const noncesPerGoroutine = 10

	var wg sync.WaitGroup
	nonces := make([]uint64, numGoroutines*noncesPerGoroutine)

	// Launch concurrent goroutines to increment nonce
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < noncesPerGoroutine; j++ {
				nonce := account.GetAndIncrementNonce()
				nonces[goroutineID*noncesPerGoroutine+j] = nonce
			}
		}(i)
	}

	wg.Wait()

	// Verify all nonces are unique and in expected range
	nonceSet := make(map[uint64]bool)
	for _, nonce := range nonces {
		require.False(t, nonceSet[nonce], "Duplicate nonce found: %d", nonce)
		nonceSet[nonce] = true
		require.Less(t, nonce, uint64(numGoroutines*noncesPerGoroutine))
	}

	// Verify we got exactly the expected number of unique nonces
	require.Len(t, nonceSet, numGoroutines*noncesPerGoroutine)

	// Verify final nonce value
	require.Equal(t, uint64(numGoroutines*noncesPerGoroutine), account.Nonce.Load())
}

func TestGenerateAccounts(t *testing.T) {
	tests := []struct {
		name  string
		count int
	}{
		{"Zero accounts", 0},
		{"Single account", 1},
		{"Multiple accounts", 10},
		{"Large batch", 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accounts := GenerateAccounts(tt.count)
			require.Len(t, accounts, tt.count)

			// Verify all accounts are unique and valid
			addressSet := make(map[common.Address]bool)
			for i, account := range accounts {
				require.NotNil(t, account, "Account %d is nil", i)
				require.NotEqual(t, common.Address{}, account.Address, "Account %d has zero address", i)
				require.NotNil(t, account.PrivKey, "Account %d has nil private key", i)
				require.Equal(t, 0, account.Nonce.Load(), "Account %d has non-zero initial nonce", i)

				// Verify address uniqueness
				require.False(t, addressSet[account.Address], "Duplicate address found: %s", account.Address.Hex())
				addressSet[account.Address] = true

				// Verify address matches private key
				expectedAddress := crypto.PubkeyToAddress(account.PrivKey.PublicKey)
				require.Equal(t, expectedAddress, account.Address, "Account %d address doesn't match private key", i)
			}
		})
	}
}

func TestAccountPoolRoundRobin(t *testing.T) {
	registry := NewAccountRegistry()
	config := &AccountConfig{
		InitialSize:    3,
		NewAccountRate: 0.0, // No new accounts, pure round-robin
	}

	pool := registry.NewPool(config)
	accounts := pool.GetAccounts()

	// The account pool starts from index 1 (due to nextIndex() incrementing first)
	// So the first call returns accounts[1], second returns accounts[2], third returns accounts[0]
	expectedOrder := []int{1, 2, 0} // The actual order the pool returns accounts

	// Test multiple rounds of round-robin selection
	rng := rng.NewSource(1).Rand("types:test")
	for round := 0; round < 3; round++ {
		for i, expectedIndex := range expectedOrder {
			selectedAccount := pool.NextAccount(rng)
			expectedAccount := accounts[expectedIndex]
			require.Equal(t, expectedAccount.Address, selectedAccount.Address,
				"Round %d, position %d: expected %s, got %s",
				round, i, expectedAccount.Address.Hex(), selectedAccount.Address.Hex())
		}
	}
}

func TestAccountPoolNewAccountRate(t *testing.T) {
	registry := NewAccountRegistry()
	config := &AccountConfig{
		InitialSize:    2,
		NewAccountRate: 1.0, // Always generate new accounts
	}

	pool := registry.NewPool(config)
	accounts := pool.GetAccounts()

	// With 100% new account rate, should never get original accounts
	originalAddresses := make(map[common.Address]bool)
	for _, account := range accounts {
		originalAddresses[account.Address] = true
	}

	for i := 0; i < 10; i++ {
		selectedAccount := pool.NextAccount(rng.NewSource(1).Rand("types:test"))
		require.False(t, originalAddresses[selectedAccount.Address],
			"Iteration %d: got original account %s when expecting new account",
			i, selectedAccount.Address.Hex())
	}
}

func TestAccountPoolMixedRate(t *testing.T) {
	registry := NewAccountRegistry()
	config := &AccountConfig{
		InitialSize:    5,
		NewAccountRate: 0.5, // 50% new accounts
	}

	pool := registry.NewPool(config)
	accounts := pool.GetAccounts()

	originalAddresses := make(map[common.Address]bool)
	for _, account := range accounts {
		originalAddresses[account.Address] = true
	}

	const iterations = 100
	originalCount := 0
	newCount := 0
	rng := rng.NewSource(1).Rand("accounts:test")

	for i := 0; i < iterations; i++ {
		selectedAccount := pool.NextAccount(rng)
		if originalAddresses[selectedAccount.Address] {
			originalCount++
		} else {
			newCount++
		}
	}

	// Seeded: the split is exact and reproducible, not probabilistic. Re-running
	// the same seeded pool must reproduce these counts. If the frozen derivation
	// changes, these expected values change with it.
	const expectedNew = 51
	require.Equal(t, expectedNew, newCount, "seeded new-account count is not reproducible")
	require.Equal(t, iterations, originalCount+newCount, "Total accounts don't match iterations")
}

func TestAccountPoolConcurrency(t *testing.T) {
	registry := NewAccountRegistry()
	config := &AccountConfig{
		InitialSize:    5,
		NewAccountRate: 0.0, // Pure round-robin for predictable testing
	}

	pool := registry.NewPool(config)
	accounts := pool.GetAccounts()

	const numGoroutines = 50
	const selectionsPerGoroutine = 20

	var wg sync.WaitGroup
	selectedAccounts := make([]common.Address, numGoroutines*selectionsPerGoroutine)

	// Launch concurrent goroutines to select accounts
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			rng := rng.NewSource(1).Rand("types:test")
			for j := 0; j < selectionsPerGoroutine; j++ {
				account := pool.NextAccount(rng)
				selectedAccounts[goroutineID*selectionsPerGoroutine+j] = account.Address
			}
		}(i)
	}

	wg.Wait()

	// Verify all selected accounts are from the original pool
	originalAddresses := make(map[common.Address]bool)
	for _, account := range accounts {
		originalAddresses[account.Address] = true
	}

	for i, address := range selectedAccounts {
		require.True(t, originalAddresses[address],
			"Selection %d: got unexpected address %s", i, address.Hex())
	}
}

func TestCreateTxFromEthTx(t *testing.T) {
	// Create a test account and scenario
	account := NewAccount()

	account.Nonce.Store(42)
	receiver := common.HexToAddress("0x1234567890123456789012345678901234567890")
	scenario := &TxScenario{
		Name:     "TestScenario",
		Sender:   account,
		Receiver: receiver,
	}

	// Create a test transaction using DynamicFeeTx (EIP-1559)
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   big.NewInt(713714), // Sei testnet chain ID
		Nonce:     scenario.Sender.Nonce.Load(),
		GasTipCap: big.NewInt(2000000000),  // 2 Gwei tip
		GasFeeCap: big.NewInt(20000000000), // 20 Gwei max fee
		Gas:       21000,                   // Gas limit
		To:        &scenario.Receiver,
		Value:     big.NewInt(1000000000000000000), // 1 ETH
		Data:      nil,
	})

	// Create LoadTx from the transaction
	loadTx := CreateTxFromEthTx(tx, scenario)

	// Verify LoadTx structure
	require.NotNil(t, loadTx)
	require.Equal(t, tx, loadTx.EthTx)
	require.Equal(t, scenario, loadTx.Scenario)
	require.NotEmpty(t, loadTx.JSONRPCPayload)
	require.NotEmpty(t, loadTx.Payload)

	// Verify JSON-RPC payload is valid JSON
	require.Contains(t, string(loadTx.JSONRPCPayload), `"jsonrpc":"2.0"`)
	require.Contains(t, string(loadTx.JSONRPCPayload), `"method":"eth_sendRawTransaction"`)
	require.Contains(t, string(loadTx.JSONRPCPayload), `"id":0`) // Numeric ID, not string

	// Verify payload matches transaction binary data
	expectedPayload, err := tx.MarshalBinary()
	require.NoError(t, err)
	require.Equal(t, expectedPayload, loadTx.Payload)
}

func TestLoadTxShardID(t *testing.T) {
	// Create more test accounts to ensure better shard distribution
	accounts := GenerateAccounts(50)

	tests := []struct {
		name       string
		numShards  int
		iterations int
	}{
		{"Single shard", 1, 10},
		{"Two shards", 2, 20},
		{"Multiple shards", 5, 50},
		{"Many shards", 16, 200}, // Increased iterations for better distribution
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shardCounts := make(map[int]int)

			for i := 0; i < tt.iterations; i++ {
				account := accounts[i%len(accounts)]
				scenario := &TxScenario{
					Name:     "TestScenario",
					Sender:   account,
					Receiver: common.Address{},
				}

				scenario.Sender.Nonce.Store(uint64(i))
				// Create a simple transaction
				tx := types.NewTx(&types.DynamicFeeTx{
					ChainID:   big.NewInt(713714), // Sei testnet chain ID
					Nonce:     scenario.Sender.Nonce.Load(),
					GasTipCap: big.NewInt(2000000000),  // 2 Gwei tip
					GasFeeCap: big.NewInt(20000000000), // 20 Gwei max fee
					Gas:       21000,                   // Gas limit
					To:        &scenario.Receiver,
					Value:     big.NewInt(0), // 0 ETH
					Data:      nil,
				})
				loadTx := CreateTxFromEthTx(tx, scenario)

				shardID := loadTx.ShardID(tt.numShards)

				// Verify shard ID is in valid range
				require.GreaterOrEqual(t, shardID, 0, "Shard ID should be non-negative")
				require.Less(t, shardID, tt.numShards, "Shard ID should be less than number of shards")

				shardCounts[shardID]++
			}

			// For tests with sufficient iterations and accounts, expect reasonable distribution
			// Note: Hash-based shard distribution can be uneven, so we don't require all shards to be used
			// Instead, we verify that the distribution is reasonable and all shard IDs are valid
			totalCount := 0
			for shardID, count := range shardCounts {
				totalCount += count
				// Verify shard IDs are in valid range
				require.GreaterOrEqual(t, shardID, 0, "Shard ID should be non-negative")
				require.Less(t, shardID, tt.numShards, "Shard ID should be less than number of shards")
			}

			// Verify total count matches iterations
			require.Equal(t, tt.iterations, totalCount, "Total shard counts should match iterations")

			// For large numbers of shards, verify we're using a reasonable number of them
			// (at least 50% of available shards for sufficient iterations)
			if tt.numShards > 4 && tt.iterations >= tt.numShards*8 {
				usedShards := len(shardCounts)
				minExpectedShards := tt.numShards / 2
				require.GreaterOrEqual(t, usedShards, minExpectedShards,
					"Expected at least %d shards to be used, got %d", minExpectedShards, usedShards)
			}
		})
	}
}

func TestLoadTxShardIDConsistency(t *testing.T) {
	// Test that the same sender always maps to the same shard
	account := NewAccount()

	scenario := &TxScenario{
		Name:     "TestScenario",
		Sender:   account,
		Receiver: common.Address{},
	}

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   big.NewInt(713714), // Sei testnet chain ID
		Nonce:     scenario.Sender.Nonce.Load(),
		GasTipCap: big.NewInt(2000000000),  // 2 Gwei tip
		GasFeeCap: big.NewInt(20000000000), // 20 Gwei max fee
		Gas:       21000,                   // Gas limit
		To:        &scenario.Receiver,
		Value:     big.NewInt(0), // 0 ETH
		Data:      nil,
	})
	loadTx := CreateTxFromEthTx(tx, scenario)

	const numShards = 8
	expectedShardID := loadTx.ShardID(numShards)

	// Test multiple times with the same sender
	for i := 0; i < 10; i++ {
		shardID := loadTx.ShardID(numShards)
		require.Equal(t, expectedShardID, shardID,
			"Shard ID should be consistent for the same sender (iteration %d)", i)
	}
}

func TestTxScenario(t *testing.T) {
	account := NewAccount()

	receiver := common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd")

	account.Nonce.Store(123)

	scenario := &TxScenario{
		Name:     "TestScenario",
		Sender:   account,
		Receiver: receiver,
	}

	// Verify all fields are set correctly
	require.Equal(t, "TestScenario", scenario.Name)
	require.Equal(t, 123, scenario.Sender.Nonce.Load())
	require.Equal(t, account, scenario.Sender)
	require.Equal(t, receiver, scenario.Receiver)
}

func TestJSONRPCPayloadFormat(t *testing.T) {
	// Test the internal JSON-RPC payload generation
	testData := []byte{0x01, 0x02, 0x03, 0x04}

	payload, err := toJSONRequestBytes(testData)
	require.NoError(t, err)

	expectedContent := `{"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":["0x01020304"],"id":0}` // Numeric ID, not string
	assert.JSONEq(t, expectedContent, string(payload))
}

func BenchmarkAccountGeneration(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewAccount()
	}
}

func BenchmarkAccountPoolNextAccount(b *testing.B) {
	registry := NewAccountRegistry()
	config := &AccountConfig{
		InitialSize:    100,
		NewAccountRate: 0.0,
	}
	pool := registry.NewPool(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.NextAccount(rng.NewSource(1).Rand("types:test"))
	}
}

func BenchmarkNonceIncrement(b *testing.B) {
	account := NewAccount()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		account.GetAndIncrementNonce()
	}
}

func BenchmarkCreateTxFromEthTx(b *testing.B) {
	account := NewAccount()

	scenario := &TxScenario{
		Name:     "BenchmarkScenario",
		Sender:   account,
		Receiver: common.Address{},
	}

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   big.NewInt(713714), // Sei testnet chain ID
		Nonce:     scenario.Sender.Nonce.Load(),
		GasTipCap: big.NewInt(2000000000),  // 2 Gwei tip
		GasFeeCap: big.NewInt(20000000000), // 20 Gwei max fee
		Gas:       21000,                   // Gas limit
		To:        &scenario.Receiver,
		Value:     big.NewInt(0), // 0 ETH
		Data:      nil,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CreateTxFromEthTx(tx, scenario)
	}
}
