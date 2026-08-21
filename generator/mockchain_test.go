package generator_test

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/sei-protocol/sei-load/utils"
)

// mockChainConfig shapes one mock chain.
type mockChainConfig struct {
	// baseNonce is the nonce an address has already reached before the run, so a
	// test can give a deployer on-chain history.
	baseNonce map[common.Address]uint64
	// revertDeployments mines every contract creation with a failed status.
	revertDeployments bool
}

// mockChain serves the smallest eth JSON-RPC surface a deployment needs. It
// mines every transaction on arrival, reports a sender's pending nonce as its
// base plus the transactions that sender has sent, and keeps what it received.
// That makes the nonce a deployment actually used observable.
type mockChain struct {
	url   string
	cfg   mockChainConfig
	state utils.Mutex[*mockChainState]
}

type mockChainState struct {
	mined  []minedTx
	byHash map[common.Hash]minedTx
}

// minedTx is one transaction the chain accepted. contract is the created
// address, and is zero for anything but a contract creation.
type minedTx struct {
	from     common.Address
	tx       *ethtypes.Transaction
	contract common.Address
}

// newMockChain starts a mock chain on a local HTTP endpoint for the test's
// lifetime.
func newMockChain(t *testing.T, cfg mockChainConfig) *mockChain {
	t.Helper()
	chain := &mockChain{
		cfg:   cfg,
		state: utils.NewMutex(&mockChainState{byHash: map[common.Hash]minedTx{}}),
	}
	srv := rpc.NewServer()
	require.NoError(t, srv.RegisterName("eth", chain))
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	chain.url = ts.URL
	return chain
}

func (m *mockChain) SendRawTransaction(_ context.Context, raw hexutil.Bytes) (common.Hash, error) {
	tx := new(ethtypes.Transaction)
	if err := tx.UnmarshalBinary(raw); err != nil {
		return common.Hash{}, err
	}
	from, err := ethtypes.Sender(ethtypes.LatestSignerForChainID(tx.ChainId()), tx)
	if err != nil {
		return common.Hash{}, err
	}
	mined := minedTx{from: from, tx: tx}
	if tx.To() == nil {
		mined.contract = crypto.CreateAddress(from, tx.Nonce())
	}
	for state := range m.state.Lock() {
		state.mined = append(state.mined, mined)
		state.byHash[tx.Hash()] = mined
	}
	return tx.Hash(), nil
}

func (m *mockChain) GetTransactionCount(_ context.Context, addr common.Address, _ rpc.BlockNumber) (hexutil.Uint64, error) {
	return hexutil.Uint64(m.cfg.baseNonce[addr] + uint64(len(m.txsFrom(addr)))), nil
}

func (m *mockChain) GetTransactionReceipt(_ context.Context, hash common.Hash) (*ethtypes.Receipt, error) {
	var (
		mined minedTx
		found bool
	)
	for state := range m.state.Lock() {
		mined, found = state.byHash[hash]
	}
	if !found {
		// A null receipt is how a chain reports a transaction it has not mined.
		return nil, nil
	}
	status := uint64(ethtypes.ReceiptStatusSuccessful)
	if m.cfg.revertDeployments && mined.contract != (common.Address{}) {
		status = ethtypes.ReceiptStatusFailed
	}
	return &ethtypes.Receipt{
		Type:              mined.tx.Type(),
		Status:            status,
		CumulativeGasUsed: 21_000,
		Logs:              []*ethtypes.Log{},
		TxHash:            hash,
		ContractAddress:   mined.contract,
		GasUsed:           21_000,
		BlockNumber:       big.NewInt(1),
	}, nil
}

func (m *mockChain) GetBalance(_ context.Context, _ common.Address, _ rpc.BlockNumberOrHash) (*hexutil.Big, error) {
	return (*hexutil.Big)(new(big.Int)), nil
}

func (m *mockChain) GetCode(_ context.Context, _ common.Address, _ rpc.BlockNumberOrHash) (hexutil.Bytes, error) {
	return hexutil.Bytes{0x60, 0x00}, nil
}

func (m *mockChain) EstimateGas(_ context.Context, _ json.RawMessage, _ *rpc.BlockNumberOrHash) (hexutil.Uint64, error) {
	return hexutil.Uint64(1_000_000), nil
}

// txCount returns how many transactions the chain has accepted.
func (m *mockChain) txCount() int {
	for state := range m.state.Lock() {
		return len(state.mined)
	}
	panic("unreachable")
}

// txsFrom returns the transactions addr sent, in arrival order.
func (m *mockChain) txsFrom(addr common.Address) []*ethtypes.Transaction {
	var txs []*ethtypes.Transaction
	for state := range m.state.Lock() {
		for _, mined := range state.mined {
			if mined.from == addr {
				txs = append(txs, mined.tx)
			}
		}
	}
	return txs
}

// noncesFrom returns the nonces addr used, in arrival order.
func (m *mockChain) noncesFrom(addr common.Address) []uint64 {
	var nonces []uint64
	for _, tx := range m.txsFrom(addr) {
		nonces = append(nonces, tx.Nonce())
	}
	return nonces
}
