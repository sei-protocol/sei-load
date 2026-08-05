package offline

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

type testGenesis struct {
	balances map[common.Address]*big.Int
	code     map[common.Address][]byte
	storage  map[common.Address]map[common.Hash]common.Hash
}

func newTestGenesis() *testGenesis {
	return &testGenesis{
		balances: map[common.Address]*big.Int{},
		code:     map[common.Address][]byte{},
		storage:  map[common.Address]map[common.Hash]common.Hash{},
	}
}

func (s *testGenesis) SetBalance(address common.Address, balance *big.Int) {
	s.balances[address] = new(big.Int).Set(balance)
}

func (s *testGenesis) SetCode(address common.Address, code []byte) {
	s.code[address] = append([]byte(nil), code...)
}

func (s *testGenesis) SetState(address common.Address, key, value common.Hash) {
	if s.storage[address] == nil {
		s.storage[address] = map[common.Hash]common.Hash{}
	}
	s.storage[address][key] = value
}

func testConfig() Config {
	return Config{
		ChainID:       big.NewInt(713_715),
		GasPrice:      new(big.Int),
		SenderBalance: new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil),
		TransferValue: big.NewInt(17),
		GasLimit:      100_000,
		ERC20Contract: common.HexToAddress("0x1000"),
	}
}

func TestTransferScenarioBuildsAndSeeds(t *testing.T) {
	cfg := testConfig()
	scenario, err := NewScenario(Transfer, cfg)
	require.NoError(t, err)
	state := newTestGenesis()
	require.NoError(t, scenario.SetupGenesis(state))

	key, err := crypto.HexToECDSA("0000000000000000000000000000000000000000000000000000000000000001")
	require.NoError(t, err)
	sender := crypto.PubkeyToAddress(key.PublicKey)
	recipient := common.HexToAddress("0x2000")
	scenario.SeedSender(state, sender)
	tx, err := scenario.BuildTransaction(key, 3, recipient)
	require.NoError(t, err)

	recovered, err := ethtypes.Sender(ethtypes.LatestSignerForChainID(cfg.ChainID), tx)
	require.NoError(t, err)
	require.Equal(t, sender, recovered)
	require.Equal(t, uint64(3), tx.Nonce())
	require.Equal(t, recipient, *tx.To())
	require.Zero(t, cfg.TransferValue.Cmp(tx.Value()))
	require.Zero(t, cfg.SenderBalance.Cmp(state.balances[sender]))
}

func TestERC20TransferScenarioUsesCompiledRuntimeAndBalanceSlot(t *testing.T) {
	cfg := testConfig()
	scenario, err := NewScenario(ERC20Transfer, cfg)
	require.NoError(t, err)
	genesis := newTestGenesis()
	require.NoError(t, scenario.SetupGenesis(genesis))
	require.NotEmpty(t, genesis.code[cfg.ERC20Contract])

	key, err := crypto.HexToECDSA("0000000000000000000000000000000000000000000000000000000000000002")
	require.NoError(t, err)
	sender := crypto.PubkeyToAddress(key.PublicKey)
	recipient := common.HexToAddress("0x3000")
	scenario.SeedSender(genesis, sender)
	require.Equal(t, common.BigToHash(cfg.TransferValue), genesis.storage[cfg.ERC20Contract][ERC20BalanceSlot(sender)])

	tx, err := scenario.BuildTransaction(key, 0, recipient)
	require.NoError(t, err)
	require.Equal(t, cfg.ERC20Contract, *tx.To())
	require.Len(t, tx.Data(), 4+32+32)

	db, err := state.New(ethtypes.EmptyRootHash, state.NewDatabaseForTesting())
	require.NoError(t, err)
	db.CreateAccount(cfg.ERC20Contract)
	db.SetCode(cfg.ERC20Contract, genesis.code[cfg.ERC20Contract])
	db.SetState(cfg.ERC20Contract, ERC20BalanceSlot(sender), common.BigToHash(cfg.TransferValue))
	_, _, err = runtime.Call(cfg.ERC20Contract, tx.Data(), &runtime.Config{
		ChainConfig: params.AllEthashProtocolChanges,
		Origin:      sender,
		BlockNumber: big.NewInt(1),
		Time:        1_700_000_000,
		GasLimit:    cfg.GasLimit,
		GasPrice:    new(big.Int),
		Value:       new(big.Int),
		BaseFee:     new(big.Int),
		State:       db,
	})
	require.NoError(t, err)
	require.Equal(t, common.Hash{}, db.GetState(cfg.ERC20Contract, ERC20BalanceSlot(sender)))
	require.Equal(t, common.BigToHash(cfg.TransferValue), db.GetState(cfg.ERC20Contract, ERC20BalanceSlot(recipient)))
}

func TestNewScenarioRejectsInvalidConfig(t *testing.T) {
	_, err := NewScenario(Transfer, Config{})
	require.ErrorContains(t, err, "chain ID")
	_, err = NewScenario("unknown", testConfig())
	require.ErrorContains(t, err, "unsupported")
}
