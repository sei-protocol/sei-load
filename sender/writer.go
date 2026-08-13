package sender

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/sei-protocol/sei-load/types"
	"github.com/sei-protocol/sei-load/utils"
)

type txsWriterInner struct {
	nextHeight      uint64
	blocksGenerated uint64

	bufferGas uint64
	txBuffer  []*types.LoadTx
	nonces    map[common.Address]uint64
}

type TxsWriter struct {
	txsDir      string
	gasPerBlock uint64
	numBlocks   uint64
	inner       utils.Mutex[*txsWriterInner]
}

func NewTxsWriter(gasPerBlock uint64, txsDir string, startHeight uint64, numBlocks uint64) *TxsWriter {
	// what height to start at?
	return &TxsWriter{
		gasPerBlock: gasPerBlock,
		txsDir:      txsDir,
		numBlocks:   numBlocks,
		inner: utils.NewMutex(&txsWriterInner{
			nextHeight:      startHeight,
			blocksGenerated: 0,
			bufferGas:       0,
			txBuffer:        make([]*types.LoadTx, 0),
			nonces:          map[common.Address]uint64{},
		}),
	}
}

// Send writes the transaction to the writer
func (w *TxsWriter) Send(ctx context.Context, tx *types.LoadTx) error {
	for inner := range w.inner.Lock() {
		// if bwe would exceed gasPerBlock, flush
		if inner.bufferGas+tx.EthTx.Gas() > w.gasPerBlock {
			if err := w.flush(inner); err != nil {
				return err
			}
		}
		// add to buffer
		inner.txBuffer = append(inner.txBuffer, tx)
		inner.bufferGas += tx.EthTx.Gas()
		if tx.Scenario.Sender.Tracked {
			inner.nonces[tx.Scenario.Sender.Address] += 1
		}
	}
	return nil
}

func (w *TxsWriter) Nonce(acc types.Account) uint64 {
	for inner := range w.inner.Lock() {
		return inner.nonces[acc.Address]
	}
	panic("unreachable")
}

type TxWriteData struct {
	TxPayloads [][]byte `json:"tx_payloads"`
}

func (w *TxsWriter) Flush(ctx context.Context) error {
	for inner := range w.inner.Lock() {
		return w.flush(inner)
	}
	panic("unreachable")
}

func (w *TxsWriter) flush(inner *txsWriterInner) error {
	defer func() {
		// clear buffer and reset bufferGas and increment nextHeight
		inner.txBuffer = nil
		inner.bufferGas = 0
		inner.nextHeight++
		inner.blocksGenerated++
	}()
	// write to dir `~/load_txs`
	// make dir if it doesn't exist
	err := os.MkdirAll(w.txsDir, 0755)
	if err != nil {
		return err
	}
	txsFile := filepath.Join(w.txsDir, fmt.Sprintf("%d_txs.json", inner.nextHeight))
	txData := TxWriteData{
		TxPayloads: make([][]byte, 0),
	}
	for _, tx := range inner.txBuffer {
		payload, err := tx.EthTx.MarshalBinary()
		if err != nil {
			return fmt.Errorf("tx.EthTx.MarshalBinary(): %w", err)
		}
		txData.TxPayloads = append(txData.TxPayloads, payload)
	}

	txDataJSON, err := json.Marshal(txData)
	if err != nil {
		return err
	}

	if err := os.WriteFile(txsFile, txDataJSON, 0644); err != nil {
		return err
	}

	log.Printf("Flushed %d transactions to %s", len(inner.txBuffer), txsFile)

	if inner.blocksGenerated >= w.numBlocks {
		return fmt.Errorf("reached max number of blocks: %d", w.numBlocks)
	}

	return nil
}
