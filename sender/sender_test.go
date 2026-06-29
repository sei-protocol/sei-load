package sender

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/sei-protocol/sei-load/types"
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
