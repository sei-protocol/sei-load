package seievmtx

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/sei-protocol/sei-chain/x/evm/types"
	"github.com/sei-protocol/sei-chain/x/evm/types/ethtx"
)

// Re-export the TxData interface and MsgEVMTransaction message so callers can
// use them without importing the internal subpackage.
type (
	TxData            = ethtx.TxData
	MsgEVMTransaction = types.MsgEVMTransaction
)

var (
	RegisterInterfaces           = types.RegisterInterfaces
	NewMsgEVMTransaction         = types.NewMsgEVMTransaction
	MustGetEVMTransactionMessage = types.MustGetEVMTransactionMessage
	GetEVMTransactionMessage     = types.GetEVMTransactionMessage
	PackTxData                   = types.PackTxData
	UnpackTxData                 = types.UnpackTxData
)

// Ensure the aliases satisfy the same interfaces as the original types.
var _ = func() func(codectypes.InterfaceRegistry) {
	return RegisterInterfaces
}

var _ sdk.Msg = (*MsgEVMTransaction)(nil)
