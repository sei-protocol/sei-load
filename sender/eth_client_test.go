package sender

import (
	"context"
	"crypto/sha256"
	"math/big"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/sei-protocol/sei-load/stats"
	"github.com/sei-protocol/sei-load/types"
	"github.com/sei-protocol/sei-load/utils"
	"github.com/sei-protocol/sei-load/utils/scope"
	"github.com/stretchr/testify/require"
)

func TestEthClientSendTx_HTTP(t *testing.T) {
	api := newMockEthAPI()
	srv := rpc.NewServer()
	require.NoError(t, srv.RegisterName("eth", api))

	ts := httptest.NewServer(srv)
	defer ts.Close()

	tx := testLoadTx(t)
	client := newEthClient(&ethClientConfig{
		ChainID:   "test-chain",
		ID:        7,
		Endpoint:  ts.URL,
		Tasks:     1,
		Collector: stats.NewCollector(),
	})

	err := scope.Run(t.Context(), func(ctx context.Context, s scope.Scope) error {
		s.SpawnBg(func() error { return utils.IgnoreCancel(client.Run(ctx)) })
		return client.Send(ctx, tx)
	})
	require.NoError(t, err)
	require.Equal(t, [][]byte{tx.Payload}, api.RawTransactions())
}

func TestEthClientSendTx_WS(t *testing.T) {
	api := newMockEthAPI()
	srv := rpc.NewServer()
	require.NoError(t, srv.RegisterName("eth", api))

	ts := httptest.NewServer(srv.WebsocketHandler([]string{"*"}))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	tx := testLoadTx(t)
	client := newEthClient(&ethClientConfig{
		ChainID:   "test-chain",
		ID:        8,
		Endpoint:  wsURL,
		Tasks:     1,
		Collector: stats.NewCollector(),
	})

	err := scope.Run(t.Context(), func(ctx context.Context, s scope.Scope) error {
		s.SpawnBg(func() error { return utils.IgnoreCancel(client.Run(ctx)) })
		return client.Send(ctx, tx)
	})
	require.NoError(t, err)
	require.Equal(t, [][]byte{tx.Payload}, api.RawTransactions())
}

type mockEthAPI struct {
	rawTxs utils.Mutex[*[][]byte]
}

func newMockEthAPI() *mockEthAPI {
	rawTxs := [][]byte{}
	return &mockEthAPI{rawTxs: utils.NewMutex(&rawTxs)}
}

func (m *mockEthAPI) SendRawTransaction(_ context.Context, rawTx hexutil.Bytes) (common.Hash, error) {
	for rawTxs := range m.rawTxs.Lock() {
		*rawTxs = append(*rawTxs, slices.Clone(rawTx))
	}
	sum := sha256.Sum256(rawTx)
	return common.BytesToHash(sum[:]), nil
}

func (m *mockEthAPI) RawTransactions() [][]byte {
	for rawTxs := range m.rawTxs.Lock() {
		return slices.Clone(*rawTxs)
	}
	panic("unreachable")
}

func testLoadTx(t *testing.T) *types.LoadTx {
	t.Helper()

	account, err := types.NewAccount()
	require.NoError(t, err)

	to := common.HexToAddress("0x0000000000000000000000000000000000000001")
	tx := ethtypes.NewTx(&ethtypes.LegacyTx{
		Nonce:    1,
		To:       &to,
		Value:    big.NewInt(1),
		Gas:      21_000,
		GasPrice: big.NewInt(1),
	})
	signedTx, err := ethtypes.SignTx(tx, ethtypes.LatestSignerForChainID(big.NewInt(1)), account.PrivKey)
	require.NoError(t, err)

	return types.CreateTxFromEthTx(signedTx, &types.TxScenario{
		Name:   "test-scenario",
		Sender: account,
	})
}
