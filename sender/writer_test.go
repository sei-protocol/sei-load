package sender

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/sei-protocol/sei-load/types"
	"github.com/stretchr/testify/require"
)

func TestTxsWriter_Flush(t *testing.T) {
	// two evm transfer txs
	writer := NewTxsWriter(42_000, "/tmp", 1, 3)
	txs := []*types.LoadTx{
		testWriterTx(t, 0),
		testWriterTx(t, 1),
		testWriterTx(t, 2),
	}

	require.NoError(t, writer.Send(t.Context(), txs[0]))
	for inner := range writer.inner.Lock() {
		require.Equal(t, uint64(1), inner.nextHeight)
		require.Equal(t, uint64(21_000), inner.bufferGas)
		require.Len(t, inner.txBuffer, 1)
		require.Equal(t, txs[0], inner.txBuffer[0])
	}

	require.NoError(t, writer.Send(t.Context(), txs[1]))
	for inner := range writer.inner.Lock() {
		require.Equal(t, uint64(1), inner.nextHeight)
		require.Equal(t, uint64(42_000), inner.bufferGas)
		require.Len(t, inner.txBuffer, 2)
		require.Equal(t, txs[1], inner.txBuffer[1])
	}

	require.NoError(t, writer.Send(t.Context(), txs[2]))
	// now should be flushed and have the new tx
	for inner := range writer.inner.Lock() {
		require.Equal(t, uint64(2), inner.nextHeight)
		require.Equal(t, uint64(21_000), inner.bufferGas)
		require.Len(t, inner.txBuffer, 1)
		require.Equal(t, txs[2], inner.txBuffer[0])
	}
}

func testWriterTx(t *testing.T, nonce uint64) *types.LoadTx {
	t.Helper()
	sender := types.NewAccount(true)
	receiver := common.HexToAddress("0x0000000000000000000000000000000000000001")
	return types.CreateTxFromEthTx(ethtypes.NewTx(&ethtypes.DynamicFeeTx{
		Nonce:     nonce,
		To:        &receiver,
		Value:     big.NewInt(1),
		Gas:       21_000,
		GasTipCap: big.NewInt(1),
		GasFeeCap: big.NewInt(1),
	}), &types.TxScenario{
		Name:     "evmtransfer",
		Nonce:    nonce,
		Sender:   sender,
		Receiver: receiver,
	})
}
