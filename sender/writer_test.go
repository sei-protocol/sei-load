package sender

import (
	"context"
	"testing"

	"github.com/sei-protocol/sei-load/config"
	"github.com/sei-protocol/sei-load/generator"
	"github.com/sei-protocol/sei-load/generator/scenarios"
	"github.com/sei-protocol/sei-load/types"
	"github.com/stretchr/testify/require"
)

func TestTxsWriter_Flush(t *testing.T) {
	// two evm transfer txs
	writer := NewTxsWriter(42000, "/tmp")

	loadConfig := &config.LoadConfig{
		ChainID: 7777,
	}

	sharedAccounts := types.NewAccountPool(&types.AccountConfig{
		Accounts:       types.GenerateAccounts(10),
		NewAccountRate: 0.0,
	})

	evmScenario := scenarios.CreateScenario(config.Scenario{
		Name:   "EVMTransfer",
		Weight: 1,
	})
	evmScenario.Deploy(loadConfig, sharedAccounts.NextAccount())

	generator := generator.NewScenarioGenerator(sharedAccounts, evmScenario)

	txs := generator.GenerateN(3)

	writer.Send(context.Background(), txs[0])
	require.Equal(t, uint64(1), writer.nextHeight)
	require.Equal(t, uint64(21000), writer.bufferGas)
	require.Len(t, writer.txBuffer, 1)
	require.Equal(t, txs[0], writer.txBuffer[0])

	writer.Send(context.Background(), txs[1])
	require.Equal(t, uint64(1), writer.nextHeight)
	require.Equal(t, uint64(42000), writer.bufferGas)
	require.Len(t, writer.txBuffer, 2)
	require.Equal(t, txs[1], writer.txBuffer[1])

	writer.Send(context.Background(), txs[2])
	// now should be flushed and have the new tx
	require.Equal(t, uint64(2), writer.nextHeight)
	require.Equal(t, uint64(21000), writer.bufferGas)
	require.Len(t, writer.txBuffer, 1)

}
