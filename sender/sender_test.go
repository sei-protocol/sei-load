package sender

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	mrand "math/rand/v2"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"

	"github.com/sei-protocol/sei-load/config"
	"github.com/sei-protocol/sei-load/generator"
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
	state := newChainState(chainConfig{})
	account := types.NewAccount(true)
	state.SetNonce(account.Address, 5)
	endpoints := state.newRPCServers(t, 2)
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
	state := newChainState(chainConfig{})
	account := types.NewAccount(false)
	state.SetNonce(account.Address, 5)
	endpoints := state.newRPCServers(t, 2)
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

func TestShardedSender_WithGeneratorAndNonceRewinds(t *testing.T) {
	tests := []struct {
		name           string
		accountCount   int
		newAccountRate float64
	}{
		{
			name:           "tracked_only",
			accountCount:   5,
			newAccountRate: 0,
		},
		{
			name:           "mixed_tracked_untracked",
			accountCount:   5,
			newAccountRate: 0.25,
		},
		{
			name:           "only_untracked",
			accountCount:   0,
			newAccountRate: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := newChainState(chainConfig{
				failNonceRPCEvery: 3,
				resetNonceAfter:   10,
				resetNonceRange:   5,
			})
			endpoints := state.newRPCServers(t, 2)

			cfg := testGeneratorConfigWithAccounts(endpoints, tt.accountCount, tt.newAccountRate)
			rng := mrand.New(mrand.NewPCG(1, 2))
			gen, err := generator.NewGenerator(rng, cfg)
			require.NoError(t, err)
			ss := newTestShardedSender(endpoints)

			require.NoError(t, scope.Run(t.Context(), func(ctx context.Context, s scope.Scope) error {
				s.SpawnBg(func() error { return utils.IgnoreCancel(ss.Run(ctx)) })
				s.SpawnBg(func() error { return utils.IgnoreCancel(gen.Run(ctx, rng, ss)) })
				for inner, ctrl := range state.inner.Lock() {
					return ctrl.WaitUntil(ctx, func() bool { return inner.nonceSum > 200 })
				}
				panic("unreachable")
			}))
		})
	}
}

type chainState struct {
	cfg   chainConfig
	inner utils.Watch[*chainStateInner]
}

type accountState struct {
	nonce       uint64
	nextResetAt uint64
}

type chainStateInner struct {
	nonceRPCs uint64
	nonceSum  uint64
	accounts  map[common.Address]*accountState
}

type chainConfig struct {
	failNonceRPCEvery uint64
	resetNonceAfter   uint64
	resetNonceRange   uint64
}

func newChainState(cfg chainConfig) *chainState {
	return &chainState{
		cfg: cfg,
		inner: utils.NewWatch(&chainStateInner{
			accounts: map[common.Address]*accountState{},
		}),
	}
}

func (s *chainState) ensure(inner *chainStateInner, addr common.Address) *accountState {
	acc, ok := inner.accounts[addr]
	if !ok {
		acc = &accountState{}
		if s.cfg.resetNonceAfter > 0 {
			acc.nextResetAt = s.cfg.resetNonceAfter
		}
		inner.accounts[addr] = acc
	}
	return acc
}

func (s *chainState) SetNonce(addr common.Address, nonce uint64) {
	for inner, ctrl := range s.inner.Lock() {
		acc := s.ensure(inner, addr)
		inner.nonceSum -= acc.nonce
		acc.nonce = nonce
		acc.nextResetAt = nonce + s.cfg.resetNonceAfter
		inner.nonceSum += acc.nonce
		ctrl.Updated()
	}
}

func (s *chainState) Nonce(addr common.Address) uint64 {
	for inner := range s.inner.Lock() {
		return s.ensure(inner, addr).nonce
	}
	panic("unreachable")
}

func (s *chainState) AccountCount() int {
	for inner := range s.inner.Lock() {
		return len(inner.accounts)
	}
	panic("unreachable")
}

func (s *chainState) WaitForNonce(ctx context.Context, addr common.Address, want uint64) error {
	for inner, ctrl := range s.inner.Lock() {
		return ctrl.WaitUntil(ctx, func() bool { return s.ensure(inner, addr).nonce == want })
	}
	panic("unreachable")
}

func (s *chainState) SendRawTransaction(rawTx hexutil.Bytes) (common.Hash, error) {
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
		acc := s.ensure(inner, sender)
		if acc.nonce != tx.Nonce() {
			return common.Hash{}, errors.New("nonce too low")
		}
		inner.nonceSum -= acc.nonce
		acc.nonce += 1
		if acc.nonce == acc.nextResetAt {
			acc.nonce -= s.cfg.resetNonceRange
			acc.nextResetAt = acc.nonce + s.cfg.resetNonceAfter
		}
		inner.nonceSum += acc.nonce
		ctrl.Updated()
		return tx.Hash(), nil
	}
	panic("unreachable")
}

func (s *chainState) GetTransactionCount(_ context.Context, addr common.Address, _ string) (hexutil.Uint64, error) {
	for inner := range s.inner.Lock() {
		inner.nonceRPCs += 1
		if s.cfg.failNonceRPCEvery > 0 && inner.nonceRPCs%s.cfg.failNonceRPCEvery == 0 {
			return 0, fmt.Errorf("internal error")
		}
		return hexutil.Uint64(s.ensure(inner, addr).nonce), nil
	}
	panic("unreachable")
}

func (s *chainState) newRPCServers(t *testing.T, n int) []string {
	t.Helper()
	endpoints := make([]string, 0, n)
	for range n {
		srv := rpc.NewServer()
		require.NoError(t, srv.RegisterName("eth", s))
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

func testGeneratorConfigWithAccounts(endpoints []string, accountCount int, newAccountRate float64) *config.LoadConfig {
	settings := config.DefaultSettings()
	settings.MaxInFlight = 10
	return &config.LoadConfig{
		ChainID:    1,
		SeiChainID: "test-chain",
		Endpoints:  endpoints,
		Accounts: &config.AccountConfig{
			Accounts:       accountCount,
			NewAccountRate: newAccountRate,
		},
		Scenarios: []config.Scenario{{
			Name:   "evmtransfer",
			Weight: 1,
		}},
		MockDeploy: true,
		Settings:   &settings,
	}
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
