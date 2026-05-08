package scenarios

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/sei-protocol/sei-load/config"
	types2 "github.com/sei-protocol/sei-load/types"
)

const EVMTransferStress = "evmtransferstress"

// EVMTransferStressScenario is a high-fee simple ETH transfer stress pattern
// (1000 gwei cap, 1 gwei tip, 10^12+1 wei value). For parity with a one-shot
// multi-sender / single-recipient genesis setup, use profiles/evm_transfer_stress.json:
// deterministicEvmStressKeys, singleUseSenders, and fixedReceiver (types.EvmStressRecipientAddress).
type EVMTransferStressScenario struct {
	*ScenarioBase
	fixedReceiver *common.Address
}

func NewEVMTransferStressScenario(cfg config.Scenario) TxGenerator {
	s := &EVMTransferStressScenario{}
	if cfg.FixedReceiver != "" {
		addr := common.HexToAddress(cfg.FixedReceiver)
		s.fixedReceiver = &addr
	}
	s.ScenarioBase = NewScenarioBase(s, cfg)
	return s
}

func (s *EVMTransferStressScenario) Name() string {
	return EVMTransferStress
}

func (s *EVMTransferStressScenario) DeployScenario(_ *config.LoadConfig, _ *types2.Account) common.Address {
	return common.Address{}
}

func (s *EVMTransferStressScenario) AttachScenario(_ *config.LoadConfig, _ common.Address) common.Address {
	return common.Address{}
}

func (s *EVMTransferStressScenario) CreateTransaction(cfg *config.LoadConfig, scenario *types2.TxScenario) (*ethtypes.Transaction, error) {
	to := scenario.Receiver
	if s.fixedReceiver != nil {
		to = *s.fixedReceiver
	}

	tx := &ethtypes.DynamicFeeTx{
		Nonce:     scenario.Sender.GetAndIncrementNonce(),
		To:        &to,
		Value:     big.NewInt(1_000_000_000_001), // 10^12+1 wei: touches both usei balance and wei remainder
		Gas:       21_000,
		GasTipCap: big.NewInt(1_000_000_000),     // 1 gwei
		GasFeeCap: big.NewInt(1_000_000_000_000), // 1000 gwei
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

	signer := ethtypes.NewCancunSigner(cfg.GetChainID())
	return ethtypes.SignTx(ethtypes.NewTx(tx), signer, scenario.Sender.PrivKey)
}
