package scenarios_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	mrand "math/rand/v2"
	"testing"

	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
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

	// Pin the default calldata: rmw(uint256 slot, bytes _pad) with
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
// single-slot keyspace is total conflict, and a large keyspace spreads draws
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
		// 512 uniform draws over 10k slots collide rarely; anything near a single
		// slot would mean the keyspace is not being indexed.
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
		requireGasCoversFloor(t, tx)
	}
	require.Len(t, seen, len(buckets), "every bucket should be drawn over 256 samples")
}

// TestStorageRWOpMix proves the operation selector honors the configured mix: a
// single-weighted op is selected exclusively, and a balanced mix reaches all
// three methods.
func TestStorageRWOpMix(t *testing.T) {
	t.Run("single weight is exclusive", func(t *testing.T) {
		gen, txs := newAttachedStorageRW(t, config.Scenario{
			Operations: config.OperationMix{config.OpWrite: 1},
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
			Operations: config.OperationMix{config.OpRead: 1, config.OpWrite: 1, config.OpRmw: 1},
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
		requireGasCoversFloor(t, tx)
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

// requireGasCoversFloor asserts the declared limit clears both admission costs a
// transaction has to satisfy: the intrinsic cost the ante checks, and the
// EIP-7623 calldata floor it does not. The floor is the one that matters here —
// Sei admits a transaction whose limit is below it, then fails execution with
// GasUsed equal to the limit, so the failure lands in a block as an included tx
// and inflates the gas-used metric a run reports.
func requireGasCoversFloor(t *testing.T, tx *ethtypes.Transaction) {
	t.Helper()

	intrinsic, err := core.IntrinsicGas(tx.Data(), tx.AccessList(), nil, false, true, true, true)
	require.NoError(t, err)
	require.GreaterOrEqualf(t, tx.Gas(), intrinsic,
		"gas limit %d is below the intrinsic cost %d for %d bytes of calldata",
		tx.Gas(), intrinsic, len(tx.Data()))

	floor, err := core.FloorDataGas(tx.Data())
	require.NoError(t, err)
	require.GreaterOrEqualf(t, tx.Gas(), floor,
		"gas limit %d is below the EIP-7623 floor %d for %d bytes of calldata",
		tx.Gas(), floor, len(tx.Data()))
}

// TestStorageRWGasClearsFloorAcrossPadSizes sweeps the pad from empty to the
// configured cap and asserts every transaction is fundable. The pre-Prague rate
// of 4 gas per pad byte fails this from roughly 4.6 KiB upward, which is inside
// the range sizeBuckets invites.
func TestStorageRWGasClearsFloorAcrossPadSizes(t *testing.T) {
	for _, pad := range []int{0, 1, 31, 32, 1024, 4096, 4544, 4609, 8192, 65536, 128 << 10} {
		t.Run(fmt.Sprintf("pad=%d", pad), func(t *testing.T) {
			for _, mix := range []config.OperationMix{{config.OpRmw: 1}, {config.OpRead: 1}, {config.OpWrite: 1}} {
				gen, txs := newAttachedStorageRW(t, config.Scenario{
					SizeDistribution: uniformDist(t),
					SizeBuckets:      []int{pad},
					Operations:       mix,
				})
				tx, err := gen.Generate(newTestRng(1), txs)
				require.NoError(t, err)
				requireGasCoversFloor(t, tx)
			}
		})
	}
}

// decodeMix builds a mix the way a profile does. Insertion order matters to the
// guard this feeds: a small map iterates in insertion order most of the time, so
// a mix built in the set's declared order would let a picker that wrongly walked
// the mix still reproduce the golden on most runs. The keys below are
// deliberately alphabetical, which for storagerw inverts the declared order
// (rmw, read, write) and turns that mutation into a reliable failure.
//
// json.Unmarshal inserts in document order, so the ordering lives in the literal
// rather than in the decoding.
func decodeMix(t *testing.T, raw string) config.OperationMix {
	t.Helper()
	var mix config.OperationMix
	require.NoError(t, json.Unmarshal([]byte(raw), &mix))
	return mix
}

// TestStorageRWDrawOrderIsStable pins the documented draw order — slot, then
// pad, then operation — with all three axes live. Reordering the picks changes
// this sequence, which is what makes a saved workload replayable; without a
// golden the order is documented but unguarded.
func TestStorageRWDrawOrderIsStable(t *testing.T) {
	scenario := config.Scenario{
		KeyDistribution:  uniformDist(t),
		RecordCount:      64,
		SizeDistribution: uniformDist(t),
		SizeBuckets:      []int{0, 32, 96},
		Operations:       decodeMix(t, `{"read":1,"rmw":1,"write":1}`),
	}
	want := []struct {
		method string
		slot   uint64
		pad    int
	}{
		{"rmw", 50, 96},
		{"read", 30, 32},
		{"rmw", 18, 96},
		{"rmw", 46, 32},
		{"rmw", 31, 96},
		{"rmw", 10, 0},
		{"read", 12, 96},
		{"write", 24, 96},
	}

	draw := func() []string {
		gen, txs := newAttachedStorageRW(t, scenario)
		rng := newTestRng(2026)
		out := make([]string, len(want))
		for i := range out {
			tx, err := gen.Generate(rng, txs)
			require.NoError(t, err)
			method, slot, pad := decodeStorageRW(t, tx.Data())
			out[i] = fmt.Sprintf("%s/%d/%d", method, slot, pad)
		}
		return out
	}

	golden := make([]string, len(want))
	for i, w := range want {
		golden[i] = fmt.Sprintf("%s/%d/%d", w.method, w.slot, w.pad)
	}
	require.Equal(t, golden, draw())

	// Same seed, a fresh scenario: the sequence repeats.
	require.Equal(t, golden, draw())
}

// TestDeployTimeoutIsNotAContextSentinel: the deploy budget must not reach the
// caller as context.DeadlineExceeded. main treats those sentinels as a clean
// shutdown so a signalled or duration-bounded run exits zero, and a deployment
// that never mined would otherwise be reported as a successful run that did
// nothing.
func TestDeployTimeoutIsNotAContextSentinel(t *testing.T) {
	cfg := &config.LoadConfig{
		ChainID: 7777,
		// A blackhole address: dialing is lazy, so the deploy reaches its wait and
		// the budget expires there rather than at dial.
		Endpoints: []string{"http://198.51.100.1:8545"},
	}
	gen := scenarios.CreateScenario(config.Scenario{Name: scenarios.StorageRW})

	_, err := gen.Deploy(t.Context(), cfg, types.GenerateAccounts(1, true)[0])
	require.Error(t, err)
	require.NotErrorIs(t, err, context.DeadlineExceeded,
		"a deploy budget that escapes as a context sentinel is read by main as a clean shutdown")
}

// TestStorageRWCoversItsDeclaredOperations closes the seam between the operation
// names config declares for this scenario and the contract methods the scenario
// calls: weighting one name alone must produce calldata for the method of that
// name. A name added to the basket with no method here fails this test rather
// than every transaction of a run.
func TestStorageRWCoversItsDeclaredOperations(t *testing.T) {
	for _, name := range config.StorageRWOperations.Names() {
		t.Run(name, func(t *testing.T) {
			gen, txs := newAttachedStorageRW(t, config.Scenario{
				Operations: config.OperationMix{name: 1},
			})
			tx, err := gen.Generate(newTestRng(1), txs)
			require.NoError(t, err)
			method, _, _ := decodeStorageRW(t, tx.Data())
			require.Equal(t, name, method)
		})
	}
}

// TestStorageRWStampsTheDrawnOperation asserts that the operation recorded on
// the TxScenario is the contract method the calldata calls. It covers every draw
// across a balanced read/write/rmw mix.
func TestStorageRWStampsTheDrawnOperation(t *testing.T) {
	gen, txs := newAttachedStorageRW(t, config.Scenario{
		Operations: config.OperationMix{config.OpRead: 1, config.OpWrite: 1, config.OpRmw: 1},
	})

	rng := newTestRng(5)
	seen := map[string]int{}
	for i := 0; i < 600; i++ {
		txs.Operation = ""
		tx, err := gen.Generate(rng, txs)
		require.NoError(t, err)
		method, _, _ := decodeStorageRW(t, tx.Data())
		require.Equal(t, method, txs.Operation)
		seen[txs.Operation]++
	}
	for _, op := range config.StorageRWOperations.Names() {
		require.Positive(t, seen[op])
	}
}

// TestStorageRWDefaultStampsItsDefaultOperation covers a scenario that
// configures no mix. The recorded operation is the set's default, rmw, so the
// dimension is never empty for StorageRW. The fallback consumes no randomness —
// see config.TestOperationMixAbsentDrawsNoRandomness.
func TestStorageRWDefaultStampsItsDefaultOperation(t *testing.T) {
	gen, txs := newAttachedStorageRW(t, config.Scenario{})

	rng := newTestRng(9)
	for i := 0; i < 16; i++ {
		txs.Operation = ""
		tx, err := gen.Generate(rng, txs)
		require.NoError(t, err)
		method, _, _ := decodeStorageRW(t, tx.Data())
		require.Equal(t, method, txs.Operation)
		require.Equal(t, config.OpRmw, txs.Operation)
	}
}
