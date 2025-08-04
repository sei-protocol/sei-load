package sender

import "github.com/sei-protocol/sei-load/types"

type TxSender interface {
	Send(tx *types.LoadTx) error
}
