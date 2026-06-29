package sender

import (
	"encoding/json"
	"fmt"
	"math/big"
	mrand "math/rand/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/sei-protocol/sei-load/generator"
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

func TestTxsWriter_WithGeneratorFinalFiles(t *testing.T) {
	tests := []struct {
		name           string
		accountCount   int
		newAccountRate float64
	}{
		{name: "tracked_only", accountCount: 5, newAccountRate: 0},
		{name: "mixed_tracked_untracked", accountCount: 5, newAccountRate: 0.25},
		{name: "only_untracked", accountCount: 0, newAccountRate: 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const (
				totalTxs             = 500
				gasPerTx      uint64 = 21_000
				txsPerFile           = 10
				expectedFiles        = totalTxs / txsPerFile
			)

			cfg := testGeneratorConfigWithAccounts(nil, tt.accountCount, tt.newAccountRate)
			rng := mrand.New(mrand.NewPCG(5, 6))
			gen, err := generator.NewGenerator(rng, cfg)
			require.NoError(t, err)

			outDir := t.TempDir()
			writer := NewTxsWriter(gasPerTx*txsPerFile, outDir, 1, expectedFiles-1)

			err = gen.Run(t.Context(), rng, writer)
			require.EqualError(t, err, fmt.Sprintf("reached max number of blocks: %d", expectedFiles-1))

			entries, err := os.ReadDir(outDir)
			require.NoError(t, err)
			require.Len(t, entries, expectedFiles)

			totalPayloads := 0
			for i := 1; i <= expectedFiles; i++ {
				path := filepath.Join(outDir, fileNameForHeight(uint64(i)))
				data, err := os.ReadFile(path)
				require.NoError(t, err)

				var txData TxWriteData
				require.NoError(t, json.Unmarshal(data, &txData))
				require.Len(t, txData.TxPayloads, txsPerFile)
				totalPayloads += len(txData.TxPayloads)
			}
			require.Equal(t, totalTxs, totalPayloads)
		})
	}
}

func fileNameForHeight(height uint64) string {
	return fmt.Sprintf("%d_txs.json", height)
}
