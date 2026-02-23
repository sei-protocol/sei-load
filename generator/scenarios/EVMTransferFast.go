package scenarios

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/sei-protocol/sei-load/config"
	types2 "github.com/sei-protocol/sei-load/types"
)

const EVMTransferFast = "evmtransferfast"

// EVMTransferFastScenario implements the TxGenerator interface for simple ETH transfers
// that only involve values that are multiples of 10^12 and no tipping.
type EVMTransferFastScenario struct {
	*ScenarioBase
}

// NewEVMTransferScenario creates a new ETH transfer scenario
func NewEVMTransferFastScenario(cfg config.Scenario) TxGenerator {
	scenario := &EVMTransferFastScenario{}
	scenario.ScenarioBase = NewScenarioBase(scenario, cfg)
	return scenario
}

// Name returns the name of the scenario.
func (s *EVMTransferFastScenario) Name() string {
	return EVMTransfer
}

// DeployScenario implements ScenarioDeployer interface - no deployment needed for ETH transfers
func (s *EVMTransferFastScenario) DeployScenario(config *config.LoadConfig, deployer *types2.Account) common.Address {
	// No deployment needed for simple ETH transfers
	// Return zero address to indicate no contract deployment
	return common.Address{}
}

// AttachScenario implements ScenarioDeployer interface - no attachment needed for ETH transfers.
func (s *EVMTransferFastScenario) AttachScenario(config *config.LoadConfig, address common.Address) common.Address {
	// No attachment needed for simple ETH transfers
	// Return zero address to indicate no contract deployment
	return common.Address{}
}

// CreateTransaction EVMTransferFastScenario ScenarioDeployer interface - creates ETH transfer transaction
func (s *EVMTransferFastScenario) CreateTransaction(config *config.LoadConfig, scenario *types2.TxScenario) (*ethtypes.Transaction, error) {
	// Create transaction with value transfer
	tx := &ethtypes.DynamicFeeTx{
		Nonce:     scenario.Sender.GetAndIncrementNonce(),
		To:        &scenario.Receiver,
		Value:     big.NewInt(1_000_000_000_000),
		Gas:       21000,                    // Standard gas limit for ETH transfer
		GasTipCap: big.NewInt(0),            // 2 gwei
		GasFeeCap: big.NewInt(200000000000), // 200 gwei
		Data:      nil,                      // No data for simple transfer
	}

	if s.scenarioConfig.GasPicker != nil {
		var err error
		tx.Gas, err = s.scenarioConfig.GasPicker.GenerateGas()
		if err != nil {
			return nil, err
		}
	}
	if s.scenarioConfig.GasTipCapPicker != nil {
		gasTipCap, err := s.scenarioConfig.GasTipCapPicker.GenerateGas()
		if err != nil {
			return nil, err
		}
		tx.GasTipCap = big.NewInt(int64(gasTipCap))
	}
	if s.scenarioConfig.GasFeeCapPicker != nil {
		gasFeeCap, err := s.scenarioConfig.GasFeeCapPicker.GenerateGas()
		if err != nil {
			return nil, err
		}
		tx.GasFeeCap = big.NewInt(int64(gasFeeCap))
	}

	// Sign the transaction
	signer := ethtypes.NewCancunSigner(config.GetChainID())
	signedTx, err := ethtypes.SignTx(ethtypes.NewTx(tx), signer, scenario.Sender.PrivKey)
	if err != nil {
		return nil, err
	}

	return signedTx, nil
}
