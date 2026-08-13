package offline

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

type transferScenario struct {
	cfg    Config
	signer ethtypes.Signer
}

func newTransferScenario(cfg Config) *transferScenario {
	return &transferScenario{
		cfg:    cfg,
		signer: ethtypes.LatestSignerForChainID(cfg.ChainID),
	}
}

func (*transferScenario) SetupGenesis(GenesisWriter) error {
	return nil
}

func (s *transferScenario) SeedSender(state GenesisWriter, sender common.Address) {
	state.SetBalance(sender, s.cfg.SenderBalance)
}

func (s *transferScenario) BuildTransaction(key *ecdsa.PrivateKey, nonce uint64, recipient common.Address) (*ethtypes.Transaction, error) {
	if key == nil {
		return nil, fmt.Errorf("sender private key is required")
	}
	tx := ethtypes.NewTx(&ethtypes.LegacyTx{
		Nonce:    nonce,
		GasPrice: new(big.Int).Set(s.cfg.GasPrice),
		Gas:      s.cfg.GasLimit,
		To:       &recipient,
		Value:    new(big.Int).Set(s.cfg.TransferValue),
	})
	return ethtypes.SignTx(tx, s.signer, key)
}
