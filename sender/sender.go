package sender

import (
	"context"
	"github.com/sei-protocol/sei-load/types"
)

type TxSender interface {
	Run(ctx context.Context, q *types.TxsQueue) error
	Send(ctx context.Context, tx *types.LoadTx) error
}
