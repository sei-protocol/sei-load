package scenarios_test

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/stretchr/testify/require"

	"github.com/sei-protocol/sei-load/config"
	"github.com/sei-protocol/sei-load/generator/bindings"
	"github.com/sei-protocol/sei-load/generator/scenarios"
	"github.com/sei-protocol/sei-load/types"
	"github.com/sei-protocol/sei-load/utils/rng"
)

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
	contractAddr := types.GenerateAccounts(1)[0].Address
	require.NoError(t, gen.Attach(cfg, contractAddr))

	// Build the tx scenario the way the weighted generator does: a funded sender.
	sender := types.GenerateAccounts(1)[0]
	txScenario := &types.TxScenario{
		Name:   scenarios.StorageRW,
		Sender: sender,
	}

	loadTx := gen.Generate(txScenario)
	require.NotNil(t, loadTx)
	require.NotNil(t, loadTx.EthTx)

	// The produced tx must target the deployed contract...
	require.NotNil(t, loadTx.EthTx.To())
	require.Equal(t, contractAddr, *loadTx.EthTx.To())

	// ...and carry rmw calldata against the fixed slot 0.
	data := loadTx.EthTx.Data()
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

// storageRWABI returns the parsed StorageRWv1 ABI for calldata decoding.
func storageRWABI(t *testing.T) *abi.ABI {
	t.Helper()
	parsed, err := bindings.StorageRWv1MetaData.GetAbi()
	require.NoError(t, err)
	return parsed
}

// decodeStorageRWTx decodes one produced tx into (method name, slot, pad-length).
// It identifies the method by its 4-byte selector and unpacks the operands so a
// test can assert on the slot draw and pad size without reimplementing ABI rules.
func decodeStorageRWTx(t *testing.T, parsed *abi.ABI, data []byte) (string, uint64, int) {
	t.Helper()
	require.GreaterOrEqual(t, len(data), 4)
	method, err := parsed.MethodById(data[:4])
	require.NoError(t, err)
	args, err := method.Inputs.Unpack(data[4:])
	require.NoError(t, err)
	slot := args[0].(*big.Int)
	pad := args[len(args)-1].([]byte) // _pad is always the trailing operand.
	return method.Name, slot.Uint64(), len(pad)
}

// newConfiguredStorageRW builds a mock-deployed StorageRW scenario from cfg and
// binds its distribution streams the way generator.bindDistributionStreams does,
// so the produced txs are seeded deterministically.
func newConfiguredStorageRW(t *testing.T, seed uint64, cfg config.Scenario) scenarios.TxGenerator {
	t.Helper()
	cfg.Name = scenarios.StorageRW
	src := rng.NewSource(seed)
	if cfg.KeyDistribution != nil {
		cfg.KeyDistribution.SetStream(src.Stream(rng.KeyDistributionStream(0)))
	}
	if cfg.SizeDistribution != nil {
		cfg.SizeDistribution.SetStream(src.Stream(rng.SizeDistributionStream(0)))
	}
	if cfg.Operations != nil {
		cfg.Operations.SetStream(src.Stream(rng.OpDistributionStream(0)))
	}
	gen := scenarios.CreateScenario(cfg)
	loadCfg := &config.LoadConfig{ChainID: 7777, MockDeploy: true, Endpoints: []string{"http://localhost:8545"}}
	require.NoError(t, gen.Attach(loadCfg, types.GenerateAccounts(1)[0].Address))
	return gen
}

// drawSlots produces count txs and returns the slot drawn for each.
func drawSlots(t *testing.T, gen scenarios.TxGenerator, parsed *abi.ABI, count int) []uint64 {
	t.Helper()
	sender := types.GenerateAccounts(1)[0]
	slots := make([]uint64, count)
	for i := range slots {
		tx := gen.Generate(&types.TxScenario{Name: scenarios.StorageRW, Sender: sender})
		_, slot, _ := decodeStorageRWTx(t, parsed, tx.EthTx.Data())
		slots[i] = slot
	}
	return slots
}

func uniformDist(t *testing.T) *config.Distribution {
	t.Helper()
	var d config.Distribution
	require.NoError(t, d.UnmarshalJSON([]byte(`{"Name":"uniform"}`)))
	return &d
}

// TestStorageRWContentionSweep pins the contention continuum at its two ends: a
// uniform draw over a huge keyspace collides almost never, and the default
// single-slot config collides always. The assertion is on the generator's slot
// draw, not on-chain state.
func TestStorageRWContentionSweep(t *testing.T) {
	t.Parallel()
	parsed := storageRWABI(t)
	const draws = 2000

	t.Run("huge_uniform_keyspace_near_zero_collision", func(t *testing.T) {
		t.Parallel()
		gen := newConfiguredStorageRW(t, 1, config.Scenario{
			KeyDistribution: uniformDist(t),
			RecordCount:     1_000_000,
		})
		slots := drawSlots(t, gen, parsed, draws)
		seen := make(map[uint64]int, draws)
		for _, s := range slots {
			seen[s]++
		}
		// Birthday collisions over a 1e6 keyspace with 2000 draws are a handful;
		// require >99% distinct so a regression that collapsed the keyspace fails.
		require.Greater(t, len(seen), int(float64(draws)*0.99),
			"uniform over a huge keyspace must barely collide; got %d distinct of %d", len(seen), draws)
	})

	t.Run("single_slot_full_collision", func(t *testing.T) {
		t.Parallel()
		// No key distribution => scaffold single-slot default: every draw is slot 0.
		gen := newConfiguredStorageRW(t, 1, config.Scenario{})
		slots := drawSlots(t, gen, parsed, draws)
		for _, s := range slots {
			require.Zero(t, s, "default config must target the single fixed slot")
		}
	})
}

// TestStorageRWSizeBuckets proves the size distribution selects calldata pad
// lengths from the configured bucket histogram and only from those buckets.
func TestStorageRWSizeBuckets(t *testing.T) {
	t.Parallel()
	parsed := storageRWABI(t)
	buckets := []int{0, 64, 256, 1024}
	allowed := make(map[int]bool, len(buckets))
	for _, b := range buckets {
		allowed[b] = true
	}

	gen := newConfiguredStorageRW(t, 7, config.Scenario{
		SizeDistribution: uniformDist(t),
		SizeBuckets:      buckets,
	})

	sender := types.GenerateAccounts(1)[0]
	const draws = 4000
	hits := make(map[int]int, len(buckets))
	for i := 0; i < draws; i++ {
		tx := gen.Generate(&types.TxScenario{Name: scenarios.StorageRW, Sender: sender})
		_, _, padLen := decodeStorageRWTx(t, parsed, tx.EthTx.Data())
		require.True(t, allowed[padLen], "pad length %d not in configured buckets", padLen)
		hits[padLen]++
	}
	// Uniform over 4 buckets: each should be well-represented (no empty bucket),
	// proving the size draw actually spans the histogram.
	for _, b := range buckets {
		require.Positive(t, hits[b], "bucket %d never selected under uniform size dist", b)
	}
}

// TestStorageRWKeySizeIndependence is the core trap guard: changing the size
// distribution must not perturb the key sequence. Same seed + same key config
// must yield an identical slot sequence regardless of the size config.
func TestStorageRWKeySizeIndependence(t *testing.T) {
	t.Parallel()
	parsed := storageRWABI(t)
	const seed, draws = 42, 500

	keyOnly := newConfiguredStorageRW(t, seed, config.Scenario{
		KeyDistribution: uniformDist(t),
		RecordCount:     100_000,
	})
	withSize := newConfiguredStorageRW(t, seed, config.Scenario{
		KeyDistribution:  uniformDist(t),
		RecordCount:      100_000,
		SizeDistribution: uniformDist(t),
		SizeBuckets:      []int{0, 128, 512},
	})

	require.Equal(t,
		drawSlots(t, keyOnly, parsed, draws),
		drawSlots(t, withSize, parsed, draws),
		"adding a size distribution must not change the key draw sequence")
}

// TestStorageRWOpIndependence guards that op selection rides its own sub-stream:
// configuring an op mix must not change the key draw sequence.
func TestStorageRWOpIndependence(t *testing.T) {
	t.Parallel()
	parsed := storageRWABI(t)
	const seed, draws = 42, 500

	keyOnly := newConfiguredStorageRW(t, seed, config.Scenario{
		KeyDistribution: uniformDist(t),
		RecordCount:     100_000,
	})
	withOps := newConfiguredStorageRW(t, seed, config.Scenario{
		KeyDistribution: uniformDist(t),
		RecordCount:     100_000,
		Operations:      &config.OperationMix{Read: 1, Write: 1, Rmw: 1},
	})

	require.Equal(t,
		drawSlots(t, keyOnly, parsed, draws),
		drawSlots(t, withOps, parsed, draws),
		"adding an operation mix must not change the key draw sequence")
}

// TestStorageRWOpMix proves the operation selector honors the configured mix:
// all three methods appear when all three are weighted, and a single-op mix
// produces only that op.
func TestStorageRWOpMix(t *testing.T) {
	t.Parallel()
	parsed := storageRWABI(t)
	sender := types.GenerateAccounts(1)[0]

	countOps := func(gen scenarios.TxGenerator, draws int) map[string]int {
		out := map[string]int{}
		for i := 0; i < draws; i++ {
			tx := gen.Generate(&types.TxScenario{Name: scenarios.StorageRW, Sender: sender})
			name, _, _ := decodeStorageRWTx(t, parsed, tx.EthTx.Data())
			out[name]++
		}
		return out
	}

	t.Run("all_three_appear", func(t *testing.T) {
		t.Parallel()
		gen := newConfiguredStorageRW(t, 3, config.Scenario{
			Operations: &config.OperationMix{Read: 1, Write: 1, Rmw: 1},
		})
		got := countOps(gen, 3000)
		require.Positive(t, got["read"])
		require.Positive(t, got["write"])
		require.Positive(t, got["rmw"])
	})

	t.Run("single_op_only", func(t *testing.T) {
		t.Parallel()
		gen := newConfiguredStorageRW(t, 3, config.Scenario{
			Operations: &config.OperationMix{Write: 1},
		})
		got := countOps(gen, 500)
		require.Equal(t, 500, got["write"])
		require.Len(t, got, 1, "single-op mix must produce only that op")
	})
}

// TestStorageRWDefaultPathByteIdentical pins the additive guarantee: a scenario
// with no distribution config produces calldata byte-identical to the PLT-461
// scaffold (fixed slot 0, empty pad, rmw) for a fixed sender/nonce.
func TestStorageRWDefaultPathByteIdentical(t *testing.T) {
	t.Parallel()
	gen := newConfiguredStorageRW(t, 99, config.Scenario{})
	sender := types.GenerateAccounts(1)[0]
	tx := gen.Generate(&types.TxScenario{Name: scenarios.StorageRW, Sender: sender})

	data := tx.EthTx.Data()
	require.Equal(t, rmwSelector, data[:4])
	body := data[4:]
	require.Len(t, body, 96)
	want := make([]byte, 96)
	want[63] = 0x40 // offset to the empty _pad bytes argument.
	require.Equal(t, want, body)
}

// TestStorageRWScenarioConfigAdditive proves the new fields are omitempty: a
// scenario carrying none of them round-trips without introducing their keys.
func TestStorageRWScenarioConfigAdditive(t *testing.T) {
	t.Parallel()
	out, err := json.Marshal(config.Scenario{Name: scenarios.StorageRW})
	require.NoError(t, err)
	for _, key := range []string{"recordCount", "sizeBuckets", "operations"} {
		require.NotContains(t, string(out), key)
	}
}
