package sender

import (
	"context"
	"log"

	"github.com/ethereum/go-ethereum/common"
	"github.com/sei-protocol/sei-load/types"
	"github.com/sei-protocol/sei-load/utils"
)

type addrNonce struct {
	addr  common.Address
	nonce uint64
}

type accState struct {
	firstNonce uint64
	nextNonce  uint64
	track      bool
}

type queue[T any] struct {
	first, next uint64
	q           map[uint64]T
}

func newQueue[T any]() *queue[T] {
	return &queue[T]{q: map[uint64]T{}}
}

func (q *queue[T]) Len() uint64 {
	return q.next-q.first
}

func (q *queue[T]) Push(v T) {
	q.q[q.next] = v
	q.next += 1
}

func (q *queue[T]) Pop() T {
	if q.first >= q.next {
		panic("empty queue")
	}
	v := q.q[q.first]
	delete(q.q, q.first)
	q.first += 1
	return v
}

type txsQueueInner struct {
	txs    map[addrNonce]*types.LoadTx
	byAddr map[common.Address]*accState
	ready  *queue[common.Address]
}

type TxsQueue struct {
	capacity int
	inner    utils.Watch[*txsQueueInner]
}

func NewTxsQueue(capacity int) *TxsQueue {
	return &TxsQueue{
		capacity: capacity,
		inner: utils.NewWatch(&txsQueueInner{
			txs:    map[addrNonce]*types.LoadTx{},
			byAddr: map[common.Address]*accState{},
			ready:  newQueue[common.Address](),
		}),
	}
}

func (q *TxsQueue) PopSent(addr common.Address) {
	for inner, ctrl := range q.inner.Lock() {
		state, ok := inner.byAddr[addr]
		if !ok || state.firstNonce == state.nextNonce {
			return
		}
		ctrl.Updated()
		delete(inner.txs, addrNonce{addr, state.firstNonce})
		state.firstNonce += 1
		if state.firstNonce < state.nextNonce {
			inner.ready.Push(addr)
		} else if !state.track {
			delete(inner.byAddr, addr)
		}
	}
}

func (q *TxsQueue) Reset(addr common.Address, nonce uint64) {
	for inner, ctrl := range q.inner.Lock() {
		state, ok := inner.byAddr[addr]
		if !ok {
			return
		}
		ctrl.Updated()
		for state.firstNonce < state.nextNonce {
			delete(inner.txs, addrNonce{addr, state.firstNonce})
			state.firstNonce += 1
		}
		if state.track {
			state.firstNonce = nonce
			state.nextNonce = nonce
		} else {
			delete(inner.byAddr, addr)
		}
	}
}

func (q *TxsQueue) PopReady(ctx context.Context) (*types.LoadTx, error) {
	for inner, ctrl := range q.inner.Lock() {
		if err := ctrl.WaitUntil(ctx, func() bool { return inner.ready.Len()>0 }); err != nil {
			return nil, err
		}
		addr := inner.ready.Pop()
		state := inner.byAddr[addr]
		an := addrNonce{addr, state.firstNonce}
		tx := inner.txs[an]
		ctrl.Updated()
		return tx, nil
	}
	panic("unreachable")
}

func (q *TxsQueue) Push(ctx context.Context, tx *types.LoadTx) error {
	for inner, ctrl := range q.inner.Lock() {
		if err := ctrl.WaitUntil(ctx, func() bool { return len(inner.txs) < q.capacity }); err != nil {
			return err
		}
		addr := tx.Scenario.Sender.Address
		nonce := tx.EthTx.Nonce()
		state, ok := inner.byAddr[addr]
		if !ok {
			state = &accState{track: tx.Scenario.Sender.Tracked}
		}
		if nonce != state.nextNonce {
			// It is expected in case of send failure.
			log.Printf("bad nonce for %v: got %v, want %v", addr, nonce, state.nextNonce)
			return nil
		}
		state.nextNonce += 1
		inner.byAddr[addr] = state
		inner.txs[addrNonce{addr, nonce}] = tx
		if state.firstNonce == nonce {
			inner.ready.Push(addr)
			ctrl.Updated()
		}
	}
	return nil
}

func (q *TxsQueue) WaitUntilEmpty(ctx context.Context) error {
	for inner, ctrl := range q.inner.Lock() {
		return ctrl.WaitUntil(ctx, func() bool { return len(inner.txs) == 0 })
	}
	panic("unreachable")
}

func (q *TxsQueue) Nonce(acc types.Account) uint64 {
	for inner := range q.inner.Lock() {
		if state, ok := inner.byAddr[acc.Address]; ok {
			return state.nextNonce
		}
	}
	return 0
}
