package seievmtx

import (
	"sync"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdkstd "github.com/cosmos/cosmos-sdk/std"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/sei-protocol/sei-chain/x/evm/types/ethtx"
)

var (
	defaultTxConfig     client.TxConfig
	defaultTxConfigOnce sync.Once
)

func getDefaultTxConfig() client.TxConfig {
	defaultTxConfigOnce.Do(func() {
		registry := codectypes.NewInterfaceRegistry()
		sdkstd.RegisterInterfaces(registry)
		RegisterInterfaces(registry)

		protoCodec := codec.NewProtoCodec(registry)
		defaultTxConfig = authtx.NewTxConfig(protoCodec, authtx.DefaultSignModes)
	})
	return defaultTxConfig
}

// DefaultTxConfig returns the lazily constructed TxConfig used by this
// package. Callers may reuse it across multiple EncodeCosmosTxFromEthTx calls
// to avoid rebuilding the interface registry.
func DefaultTxConfig() client.TxConfig {
	return getDefaultTxConfig()
}

func EncodeCosmosTxFromEthTx(ethTx *ethtypes.Transaction) ([]byte, error) {
	if ethTx == nil {
		return nil, sdkerrors.ErrInvalidRequest.Wrap("nil ethereum tx")
	}
	return EncodeCosmosTxFromEthTxWithConfig(DefaultTxConfig(), ethTx)
}

func EncodeCosmosTxFromEthTxWithConfig(txConfig client.TxConfig, ethTx *ethtypes.Transaction) ([]byte, error) {
	if ethTx == nil {
		return nil, sdkerrors.ErrInvalidRequest.Wrap("nil ethereum tx")
	}

	txData, err := ethtx.NewTxDataFromTx(ethTx)
	if err != nil {
		return nil, err
	}

	msg, err := NewMsgEVMTransaction(txData)
	if err != nil {
		return nil, err
	}

	builder := txConfig.NewTxBuilder()
	if err := builder.SetMsgs(msg); err != nil {
		return nil, err
	}

	if ethTx.Gas() > 0 {
		builder.SetGasLimit(ethTx.Gas())
	}

	return txConfig.TxEncoder()(builder.GetTx())
}
