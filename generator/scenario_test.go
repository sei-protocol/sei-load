package generator

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/sei-protocol/sei-load/config"
	"github.com/sei-protocol/sei-load/generator/scenarios"
	"github.com/sei-protocol/sei-load/types"
)

func TestScenarioGenerator_FixedReceiverUsesOnePoolAccount(t *testing.T) {
	accs := types.GenerateAccounts(2)
	pool := types.NewAccountPool(&types.AccountConfig{
		Accounts:         accs,
		NewAccountRate:   0,
		SingleUseSenders: true,
	})
	evm := scenarios.NewEVMTransferScenario(config.Scenario{})
	cfg := &config.LoadConfig{ChainID: 713714}
	evm.Deploy(cfg, accs[0])

	fixed := "0x00000000000000000000000000000000000000aa"
	want := common.HexToAddress(fixed)
	gen := NewScenarioGenerator(pool, evm, config.Scenario{FixedReceiver: fixed})

	tx1, ok := gen.Generate()
	require.True(t, ok)
	require.NotNil(t, tx1)
	require.Equal(t, want, tx1.Scenario.Receiver)

	tx2, ok := gen.Generate()
	require.True(t, ok)
	require.NotNil(t, tx2)

	_, ok = gen.Generate()
	require.False(t, ok)
}
