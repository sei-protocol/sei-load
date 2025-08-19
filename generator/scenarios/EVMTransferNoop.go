package scenarios

import (
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"math/big"

	"github.com/sei-protocol/sei-load/config"
	types2 "github.com/sei-protocol/sei-load/types"
)

const EVMTransferNoop = "evmtransfernoop"

// EVMTransferNoopScenario implements the TxGenerator interface for simple ETH transfers of 0 value
type EVMTransferNoopScenario struct {
	*ScenarioBase
}

// NewEVMTransferNoopScenario creates a new ETH transfer scenario
func NewEVMTransferNoopScenario() TxGenerator {
	scenario := &EVMTransferNoopScenario{}
	scenario.ScenarioBase = NewScenarioBase(scenario)
	return scenario
}

// Name returns the name of the scenario.
func (s *EVMTransferNoopScenario) Name() string {
	return EVMTransfer
}

// DeployScenario implements ScenarioDeployer interface - no deployment needed for ETH transfers
func (s *EVMTransferNoopScenario) DeployScenario(config *config.LoadConfig, deployer *types2.Account) common.Address {
	// No deployment needed for simple ETH transfers
	// Return zero address to indicate no contract deployment
	return common.Address{}
}

// AttachScenario implements ScenarioDeployer interface - no attachment needed for ETH transfers.
func (s *EVMTransferNoopScenario) AttachScenario(config *config.LoadConfig, address common.Address) common.Address {
	// No attachment needed for simple ETH transfers
	// Return zero address to indicate no contract deployment
	return common.Address{}
}

// CreateTransaction implements ScenarioDeployer interface - creates ETH transfer transaction
func (s *EVMTransferNoopScenario) CreateTransaction(config *config.LoadConfig, scenario *types2.TxScenario) (*ethtypes.Transaction, error) {
	// Create transaction with value transfer
	tx := &ethtypes.DynamicFeeTx{
		Nonce:     scenario.Nonce,
		To:        &scenario.Sender.Address,
		Value:     big.NewInt(0),
		Gas:       21000,                   // Standard gas limit for ETH transfer
		GasTipCap: big.NewInt(2000000000),  // 2 gwei
		GasFeeCap: big.NewInt(20000000000), // 20 gwei
		Data:      nil,                     // No data for simple transfer
	}

	// Sign the transaction
	signer := ethtypes.NewCancunSigner(config.GetChainID())
	signedTx, err := ethtypes.SignTx(ethtypes.NewTx(tx), signer, scenario.Sender.PrivKey)
	if err != nil {
		return nil, err
	}

	return signedTx, nil
}
