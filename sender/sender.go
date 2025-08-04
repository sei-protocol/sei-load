package sender

import (
	"context"
	"github.com/sei-protocol/sei-load/types"
)

type TxSender interface {
	Send(ctx context.Context, tx *types.LoadTx) error
}
