package sender

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"

	"github.com/sei-protocol/sei-load/config"
	"github.com/sei-protocol/sei-load/stats"
	"github.com/sei-protocol/sei-load/types"
	"github.com/sei-protocol/sei-load/utils"
	"github.com/sei-protocol/sei-load/utils/scope"
)

func TestShardDistributionVerification(t *testing.T) {
	client := &ethClient{cfg: &ethClientConfig{
		Endpoints: []string{
			"http://localhost:8545",
			"http://localhost:8546",
		},
	}}

	addr := common.HexToAddress("0x1234567890123456789012345678901234567890")
	for range 10 {
		shardID := client.shardID(addr)
		require.GreaterOrEqual(t, shardID, 0)
		require.Less(t, shardID, len(client.cfg.Endpoints))
	}
}

func TestShardDistribution(t *testing.T) {
	client := &ethClient{cfg: &ethClientConfig{
		Endpoints: []string{
			"http://localhost:8545",
			"http://localhost:8546",
		},
	}}

	accounts := types.GenerateAccounts(100, true)
	seen := map[int]int{}
	for _, account := range accounts {
		scenario := &types.TxScenario{Name: "test", Sender: account}
		tx := types.CreateTxFromEthTx(ethtypes.NewTx(&ethtypes.DynamicFeeTx{
			Nonce:     0,
			To:        &common.Address{},
			Value:     big.NewInt(0),
			Gas:       21_000,
			GasTipCap: big.NewInt(1),
			GasFeeCap: big.NewInt(1),
		}), scenario)
		shardID := client.shardID(tx.Scenario.Sender.Address)
		require.GreaterOrEqual(t, shardID, 0)
		require.Less(t, shardID, len(client.cfg.Endpoints))
		seen[shardID]++
	}

	require.NotZero(t, seen[0])
	require.NotZero(t, seen[1])
}

func TestShardedSender_TrackedReset(t *testing.T) {
	state := newFakeChainState()
	account := types.NewAccount(true)
	state.SetNonce(account.Address, 5)
	endpoints := newFakeRPCEndpoints(t, state, 2)
	ss := newTestShardedSender(endpoints)

	require.NoError(t, scope.Run(t.Context(), func(ctx context.Context, s scope.Scope) error {
		s.SpawnBg(func() error { return utils.IgnoreCancel(ss.Run(ctx)) })

		if err := ss.Send(ctx, signedLoadTx(t, account, 0)); err != nil {
			return fmt.Errorf("send tracked tx nonce 0: %w", err)
		}
		if err := ss.Send(ctx, signedLoadTx(t, account, 1)); err != nil {
			return fmt.Errorf("send tracked tx nonce 1: %w", err)
		}
		if err := ss.Flush(ctx); err != nil {
			return fmt.Errorf("flush tracked stale txs: %w", err)
		}
		if got := ss.Nonce(account); got != 5 {
			return fmt.Errorf("tracked nonce after reset = %d, want 5", got)
		}

		if err := ss.Send(ctx, signedLoadTx(t, account, 5)); err != nil {
			return fmt.Errorf("send tracked tx nonce 5: %w", err)
		}
		if err := state.WaitForNonce(ctx, account.Address, 6); err != nil {
			return fmt.Errorf("wait for tracked chain nonce advance: %w", err)
		}
		if err := ss.Flush(ctx); err != nil {
			return fmt.Errorf("flush tracked fresh tx: %w", err)
		}
		return nil
	}))
}

func TestShardedSender_UntrackedReset(t *testing.T) {
	state := newFakeChainState()
	account := types.NewAccount(false)
	state.SetNonce(account.Address, 5)
	endpoints := newFakeRPCEndpoints(t, state, 2)
	ss := newTestShardedSender(endpoints)

	require.NoError(t, scope.Run(t.Context(), func(ctx context.Context, s scope.Scope) error {
		s.SpawnBg(func() error { return utils.IgnoreCancel(ss.Run(ctx)) })

		if err := ss.Send(ctx, signedLoadTx(t, account, 0)); err != nil {
			return fmt.Errorf("send untracked tx nonce 0: %w", err)
		}
		if err := ss.Send(ctx, signedLoadTx(t, account, 1)); err != nil {
			return fmt.Errorf("send untracked tx nonce 1: %w", err)
		}

		if err := ss.Flush(ctx); err != nil {
			return fmt.Errorf("flush untracked stale txs: %w", err)
		}

		if got := ss.Nonce(account); got != 0 {
			return fmt.Errorf("untracked queue nonce = %d, want 0", got)
		}
		if got := state.Nonce(account.Address); got != 5 {
			return fmt.Errorf("untracked chain nonce = %d, want 5", got)
		}
		return nil
	}))
}

type fakeChainState struct {
	inner        utils.Watch[*fakeChainStateInner]
}

type fakeChainStateInner struct {
	nonces map[common.Address]uint64
}

func newFakeChainState() *fakeChainState {
	return &fakeChainState{
		inner: utils.NewWatch(&fakeChainStateInner{
			nonces: map[common.Address]uint64{},
		}),
	}
}

func (s *fakeChainState) SetNonce(addr common.Address, nonce uint64) {
	for inner, ctrl := range s.inner.Lock() {
		inner.nonces[addr] = nonce
		ctrl.Updated()
	}
}

func (s *fakeChainState) Nonce(addr common.Address) uint64 {
	for inner := range s.inner.Lock() {
		return inner.nonces[addr]
	}
	panic("unreachable")
}

func (s *fakeChainState) WaitForNonce(ctx context.Context, addr common.Address, want uint64) error {
	for inner, ctrl := range s.inner.Lock() {
		return ctrl.WaitUntil(ctx, func() bool { return inner.nonces[addr] == want })
	}
	panic("unreachable")
}

func (s *fakeChainState) HandleSendRawTransaction(rawTx hexutil.Bytes) (common.Hash, error) {
	tx := new(ethtypes.Transaction)
	if err := tx.UnmarshalBinary(rawTx); err != nil {
		return common.Hash{}, err
	}
	signer := ethtypes.LatestSignerForChainID(tx.ChainId())
	sender, err := ethtypes.Sender(signer, tx)
	if err != nil {
		return common.Hash{}, err
	}

	for inner, ctrl := range s.inner.Lock() {
		wantNonce := inner.nonces[sender]
		if tx.Nonce() != wantNonce {
			return common.Hash{}, errors.New("nonce too low")
		}
		inner.nonces[sender] = wantNonce + 1
		ctrl.Updated()
		return tx.Hash(), nil
	}
	panic("unreachable")
}

func (s *fakeChainState) HandleGetTransactionCount(addr common.Address) hexutil.Uint64 {
	for inner := range s.inner.Lock() {
		return hexutil.Uint64(inner.nonces[addr])
	}
	panic("unreachable")
}

type fakeEthAPI struct{ state *fakeChainState }

func (a fakeEthAPI) SendRawTransaction(_ context.Context, rawTx hexutil.Bytes) (common.Hash, error) {
	return a.state.HandleSendRawTransaction(rawTx)
}

func (a fakeEthAPI) GetTransactionCount(_ context.Context, addr common.Address, _ string) (hexutil.Uint64, error) {
	return a.state.HandleGetTransactionCount(addr), nil
}

func newFakeRPCEndpoints(t *testing.T, state *fakeChainState, n int) []string {
	t.Helper()
	endpoints := make([]string, 0, n)
	for range n {
		srv := rpc.NewServer()
		require.NoError(t, srv.RegisterName("eth", fakeEthAPI{state: state}))
		ts := httptest.NewServer(srv)
		t.Cleanup(ts.Close)
		endpoints = append(endpoints, ts.URL)
	}
	return endpoints
}

func newTestShardedSender(endpoints []string) *ShardedSender {
	settings := config.DefaultSettings()
	settings.MaxInFlight = 16
	return NewShardedSender(&config.LoadConfig{
		SeiChainID: "test-chain",
		Endpoints:  endpoints,
		Settings:   &settings,
	}, rate.NewLimiter(rate.Inf, 1), stats.NewCollector(), utils.None[*stats.InclusionTracker]())
}

func signedLoadTx(t *testing.T, account types.Account, nonce uint64) *types.LoadTx {
	t.Helper()
	to := common.HexToAddress("0x0000000000000000000000000000000000000001")
	tx := ethtypes.NewTx(&ethtypes.DynamicFeeTx{
		ChainID:   big.NewInt(1),
		Nonce:     nonce,
		To:        &to,
		Value:     big.NewInt(1),
		Gas:       21_000,
		GasTipCap: big.NewInt(1),
		GasFeeCap: big.NewInt(1),
	})
	signed, err := signTx(tx, account.PrivKey)
	require.NoError(t, err)
	return types.CreateTxFromEthTx(signed, &types.TxScenario{
		Name:     "test",
		Nonce:    nonce,
		Sender:   account,
		Receiver: to,
	})
}

func signTx(tx *ethtypes.Transaction, key *ecdsa.PrivateKey) (*ethtypes.Transaction, error) {
	return ethtypes.SignTx(tx, ethtypes.LatestSignerForChainID(big.NewInt(1)), key)
}
