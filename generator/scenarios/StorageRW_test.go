package scenarios_test

import (
	"encoding/json"
	"math/big"
	mrand "math/rand/v2"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sei-protocol/sei-load/config"
	"github.com/sei-protocol/sei-load/generator/bindings"
	"github.com/sei-protocol/sei-load/generator/scenarios"
	"github.com/sei-protocol/sei-load/types"
)

func newTestRng(seed uint64) *mrand.Rand {
	return mrand.New(mrand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
}

// rmwSelector is the 4-byte function selector for StorageRWv1.rmw(uint256,bytes).
// It is the ABI-derived discriminator the produced calldata must start with.
var rmwSelector = []byte{0x22, 0x74, 0x6b, 0x07}

// TestStorageRWFactoryRegistration proves the scenario is reachable by name
// through the factory.
func TestStorageRWFactoryRegistration(t *testing.T) {
	gen := scenarios.CreateScenario(config.Scenario{Name: scenarios.StorageRW})
	require.NotNil(t, gen)
	require.Equal(t, scenarios.StorageRW, gen.Name())
}

// TestStorageRWDeployAndGenerate proves the deploy + send path end-to-end under
// mock deploy: the scenario binds StorageRWv1, attaches at a known address, and
// produces a valid fixed rmw transaction targeting that contract.
func TestStorageRWDeployAndGenerate(t *testing.T) {
	cfg := &config.LoadConfig{
		ChainID:    7777,
		MockDeploy: true,
		Endpoints:  []string{"http://localhost:8545"},
	}

	gen := scenarios.CreateScenario(config.Scenario{Name: scenarios.StorageRW})

	// Mirror generator.mockDeployAll: attach the bound contract at a known address.
	contractAddr := types.GenerateAccounts(1, false)[0].Address
	require.NoError(t, gen.Attach(cfg, contractAddr))

	// Build the tx scenario the way the weighted generator does: a funded sender.
	sender := types.GenerateAccounts(1, true)[0]
	txScenario := &types.TxScenario{
		Name:   scenarios.StorageRW,
		Nonce:  0,
		Sender: sender,
	}

	tx, err := gen.Generate(newTestRng(1), txScenario)
	require.NoError(t, err)
	require.NotNil(t, tx)

	// The produced tx must target the deployed contract...
	require.NotNil(t, tx.To())
	require.Equal(t, contractAddr, *tx.To())

	// ...and carry rmw calldata against the fixed slot 0.
	data := tx.Data()
	require.GreaterOrEqual(t, len(data), 4)
	require.Equal(t, rmwSelector, data[:4])

	// Pin the fixed scaffold calldata: rmw(uint256 slot, bytes _pad) with
	// slot == 0 and an empty pad. ABI head is the slot operand (32B) then the
	// bytes offset (0x40); the tail is the bytes length (0). All zero except the
	// 0x40 offset, so the full body is 96 bytes.
	body := data[4:]
	require.Len(t, body, 96)
	wantBody := make([]byte, 96)
	wantBody[63] = 0x40 // offset to the _pad bytes argument
	require.Equal(t, wantBody, body)

	// Sanity: the selector we assert against matches the binding's ABI.
	parsed, err := bindings.StorageRWv1MetaData.GetAbi()
	require.NoError(t, err)
	require.Equal(t, rmwSelector, parsed.Methods["rmw"].ID)
}

// newAttachedStorageRW builds a StorageRW scenario from sc and attaches it at a
// known address under mock deploy, mirroring generator.mockDeployAll. It returns
// the generator and a tx scenario carrying a funded sender.
func newAttachedStorageRW(t *testing.T, sc config.Scenario) (scenarios.TxGenerator, *types.TxScenario) {
	t.Helper()
	sc.Name = scenarios.StorageRW
	cfg := &config.LoadConfig{
		ChainID:    7777,
		MockDeploy: true,
		Endpoints:  []string{"http://localhost:8545"},
	}
	gen := scenarios.CreateScenario(sc)
	require.NoError(t, gen.Attach(cfg, types.GenerateAccounts(1, false)[0].Address))
	return gen, &types.TxScenario{
		Name:   scenarios.StorageRW,
		Nonce:  0,
		Sender: types.GenerateAccounts(1, true)[0],
	}
}

// decodeStorageRW unpacks StorageRW calldata through the binding's own ABI, so
// the assertions below read against method names rather than byte offsets. The
// slot is the first argument of every method and the pad is the last.
func decodeStorageRW(t *testing.T, data []byte) (method string, slot uint64, padLen int) {
	t.Helper()
	parsed, err := bindings.StorageRWv1MetaData.GetAbi()
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(data), 4)
	m, err := parsed.MethodById(data[:4])
	require.NoError(t, err)
	args, err := m.Inputs.Unpack(data[4:])
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(args), 2)
	return m.Name, args[0].(*big.Int).Uint64(), len(args[len(args)-1].([]byte))
}

// uniformDist unmarshals a uniform Distribution the way a profile would, so the
// tests exercise the same wire path operators use.
func uniformDist(t *testing.T) *config.Distribution {
	t.Helper()
	var d config.Distribution
	require.NoError(t, json.Unmarshal([]byte(`{"Name":"uniform"}`), &d))
	return &d
}

// TestStorageRWContentionSweep pins the contention continuum at both ends: a
// single-slot keyspace is 100% conflict, and a large keyspace spreads draws
// across many distinct slots.
func TestStorageRWContentionSweep(t *testing.T) {
	t.Run("single slot keyspace is total conflict", func(t *testing.T) {
		gen, txs := newAttachedStorageRW(t, config.Scenario{
			KeyDistribution: uniformDist(t),
			RecordCount:     1,
		})
		rng := newTestRng(7)
		for i := 0; i < 64; i++ {
			tx, err := gen.Generate(rng, txs)
			require.NoError(t, err)
			_, slot, _ := decodeStorageRW(t, tx.Data())
			require.Zero(t, slot)
		}
	})

	t.Run("large keyspace spreads draws", func(t *testing.T) {
		const keyspace = 10000
		gen, txs := newAttachedStorageRW(t, config.Scenario{
			KeyDistribution: uniformDist(t),
			RecordCount:     keyspace,
		})
		rng := newTestRng(7)
		seen := map[uint64]struct{}{}
		for i := 0; i < 512; i++ {
			tx, err := gen.Generate(rng, txs)
			require.NoError(t, err)
			_, slot, _ := decodeStorageRW(t, tx.Data())
			require.Less(t, slot, uint64(keyspace))
			seen[slot] = struct{}{}
		}
		// 512 uniform draws over 10k slots collide rarely; anything near 1 slot
		// would mean the keyspace is not being indexed.
		require.Greater(t, len(seen), 400)
	})
}

// TestStorageRWSizeBuckets proves the size distribution selects calldata pad
// lengths from the configured histogram, and that gas scales with the pad.
func TestStorageRWSizeBuckets(t *testing.T) {
	buckets := []int{0, 128, 1024}
	gen, txs := newAttachedStorageRW(t, config.Scenario{
		SizeDistribution: uniformDist(t),
		SizeBuckets:      buckets,
	})

	rng := newTestRng(11)
	seen := map[int]struct{}{}
	for i := 0; i < 256; i++ {
		tx, err := gen.Generate(rng, txs)
		require.NoError(t, err)
		_, _, padLen := decodeStorageRW(t, tx.Data())
		require.Contains(t, buckets, padLen)
		seen[padLen] = struct{}{}
		// Gas covers the base plus the pad's intrinsic calldata cost.
		require.Equal(t, uint64(50000+padLen*4), tx.Gas())
	}
	require.Len(t, seen, len(buckets), "every bucket should be drawn over 256 samples")
}

// TestStorageRWOpMix proves the operation selector honors the configured mix: a
// single-weighted op is selected exclusively, and a balanced mix reaches all
// three methods.
func TestStorageRWOpMix(t *testing.T) {
	t.Run("single weight is exclusive", func(t *testing.T) {
		gen, txs := newAttachedStorageRW(t, config.Scenario{
			Operations: &config.OperationMix{Write: 1},
		})
		rng := newTestRng(3)
		for i := 0; i < 64; i++ {
			tx, err := gen.Generate(rng, txs)
			require.NoError(t, err)
			method, _, _ := decodeStorageRW(t, tx.Data())
			require.Equal(t, "write", method)
		}
	})

	t.Run("balanced mix reaches every method", func(t *testing.T) {
		gen, txs := newAttachedStorageRW(t, config.Scenario{
			Operations: &config.OperationMix{Read: 1, Write: 1, Rmw: 1},
		})
		rng := newTestRng(3)
		seen := map[string]int{}
		for i := 0; i < 600; i++ {
			tx, err := gen.Generate(rng, txs)
			require.NoError(t, err)
			method, _, _ := decodeStorageRW(t, tx.Data())
			seen[method]++
		}
		require.Positive(t, seen["read"])
		require.Positive(t, seen["write"])
		require.Positive(t, seen["rmw"])
	})
}

// TestStorageRWDefaultPathUnchanged pins the additive guarantee: a scenario with
// no distribution config produces exactly the pre-existing fixed rmw transaction
// and draws no randomness, so adding these fields cannot perturb an existing
// profile's workload.
func TestStorageRWDefaultPathUnchanged(t *testing.T) {
	gen, txs := newAttachedStorageRW(t, config.Scenario{})

	rng := newTestRng(42)
	for i := 0; i < 64; i++ {
		tx, err := gen.Generate(rng, txs)
		require.NoError(t, err)

		method, slot, padLen := decodeStorageRW(t, tx.Data())
		require.Equal(t, "rmw", method)
		require.Zero(t, slot)
		require.Zero(t, padLen)
		require.Equal(t, uint64(50000), tx.Gas())
	}

	// The default path must consume no randomness: an untouched RNG at the same
	// seed is still in lockstep with the one the generator was handed.
	require.Equal(t, newTestRng(42).Uint64(), rng.Uint64())
}

// TestStorageRWScenarioConfigAdditive proves the new fields are omitempty, so a
// profile that does not set them round-trips without gaining keys.
func TestStorageRWScenarioConfigAdditive(t *testing.T) {
	encoded, err := json.Marshal(config.Scenario{Name: scenarios.StorageRW, Weight: 1})
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "recordCount")
	require.NotContains(t, string(encoded), "sizeBuckets")
	require.NotContains(t, string(encoded), "operations")
}
