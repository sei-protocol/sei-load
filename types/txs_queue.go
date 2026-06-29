package types

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	"log"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/sei-protocol/sei-load/utils"
	"time"
)

type addrNonce struct {
	addr common.Address
  nonce uint64
}

type accState struct {
	firstNonce uint64
	nextNonce uint64
	track bool
}

type queue[T any] struct {
	first,next uint64
	q map[uint64]T
}

func newQueue[T any]() *queue[T] {
	return &queue[T]{q:map[uint64]T{}}
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
	delete(q.q,q.first)
	q.first += 1
	return v
}

type txsQueueInner struct {
	capacity int
	txs map[addrNonce]*LoadTx
	byAddr map[common.Address]*accState
	ready queue[common.Address]
}

type TxsQueue struct {
	inner utils.Watch[*txsQueueInner]
}

func NewTxsQueue(capacity int) *TxsQueue {
	return &TxsQueue{
		inner: utils.NewWatch(&txsQueueInner {
			capacity: capacity,
			txs: map[addrNonce]*LoadTx{},	
			byAddr: map[common.Address]*accState{},
		}),
	}
}

func (q *TxsQueue) ack(an addrNonce, resetNonce utils.Option[uint64]) {
	for inner,ctrl := range q.inner.Lock() {
		state := inner.byAddr[an.addr]
		if an.nonce != state.firstNonce {
			// It can happen if ack returned by Pop is called twice.
			panic("bad nonce acknowledged")
		}
		ctrl.Updated()
		if resetNonce,ok := resetNonce.Get(); ok {
			for state.firstNonce<state.nextNonce {
				delete(inner.txs, addrNonce{an.addr,state.firstNonce})
			}
			state.firstNonce = resetNonce
			state.nextNonce = resetNonce
		} else {
			delete(inner.txs, addrNonce{an.addr,state.firstNonce})
			state.firstNonce += 1	
		}
		if state.firstNonce < state.nextNonce {
			inner.ready.Push(an.addr)
		} else if !state.track {
			delete(inner.byAddr, an.addr)
		}
	}
}

func (q *TxsQueue) Pop(ctx context.Context) (tx *LoadTx, ack func(resetNonce utils.Option[uint64]), err error) {
	for inner,ctrl := range q.inner.Lock() {
		if err:=ctrl.WaitUntil(ctx,func() bool { return len(inner.txs) == 0 }); err!=nil {
			return nil,nil,err
		}
		addr := inner.ready.Pop()
		state := inner.byAddr[addr]
		an := addrNonce{addr,state.firstNonce}
		tx := inner.txs[an]
		ctrl.Updated()
		return tx, func(resetNonce utils.Option[uint64]){ q.ack(an,resetNonce) }, nil
	}
	panic("unreachable")
}

func (q *TxsQueue) Push(ctx context.Context, scenario *TxScenario, tx *ethtypes.Transaction) error {
	for inner,ctrl := range q.inner.Lock() {
		if err:=ctrl.WaitUntil(ctx,func() bool { return len(inner.txs) < inner.capacity }); err!=nil {
			return err
		}
		addr := scenario.Sender.Address
		nonce := tx.Nonce()
		ltx := &LoadTx{EthTx: tx, IntendedSendTime: time.Now(), Scenario: scenario}
		state,ok := inner.byAddr[addr]
		if !ok {
			state = &accState{track: scenario.Sender.Tracked}
		}
		if nonce!=state.nextNonce {
			// It is expected in case of send failure.
			log.Printf("bad nonce for %v: got %v, want %v",addr,tx.Nonce(),state.nextNonce)
			return nil
		}
		state.nextNonce += 1
		inner.byAddr[addr] = state
		inner.txs[addrNonce{addr,nonce}] = ltx
		if state.firstNonce == nonce {
			inner.ready.Push(addr)
			ctrl.Updated()
		}
	}
	return nil
}

func (q *TxsQueue) WaitUntilEmpty(ctx context.Context) error {
	for inner,ctrl := range q.inner.Lock() {
		return ctrl.WaitUntil(ctx, func() bool { return len(inner.txs)==0 })
	}
	panic("unreachable")
}

func (q *TxsQueue) Nonce(addr common.Address) uint64 {
	for inner := range q.inner.Lock() {
		if state, ok := inner.byAddr[addr]; ok {
			return state.nextNonce
		}
	}
	return 0
}
