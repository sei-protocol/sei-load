package scenarios_test

import (
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
