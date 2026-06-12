package sender

import (
	"context"

	"github.com/sei-protocol/sei-load/utils"
)

type queueSlot struct {
	id   queueID
	slot int
}

type queueID int

type queueState struct {
	first int
	next  int
}

type queuePoolState[T any] struct {
	mem    map[queueSlot]T
	queues []queueState
}

type QueuePool[T any] struct {
	state utils.Mutex[*queuePoolState[T]]
	size  chan struct{}
}

type Queue[T any] struct {
	id   queueID
	pool *QueuePool[T]
	size chan struct{}
}

func (q *Queue[T]) Len() int { return len(q.size) }

func NewQueuePool[T any](capacity int) *QueuePool[T] {
	return &QueuePool[T]{
		state: utils.NewMutex(&queuePoolState[T]{
			mem: make(map[queueSlot]T, capacity),
		}),
		size: make(chan struct{}, capacity),
	}
}

func (p *QueuePool[T]) NewQueue() *Queue[T] {
	for state := range p.state.Lock() {
		id := queueID(len(state.queues))
		state.queues = append(state.queues, queueState{})
		return &Queue[T]{
			id:   id,
			pool: p,
			size: make(chan struct{}, cap(p.size)),
		}
	}
	panic("unreachable")
}

func (q *Queue[T]) Send(ctx context.Context, v T) error {
	if err := utils.Send(ctx, q.pool.size, struct{}{}); err != nil {
		return err
	}
	for state := range q.pool.state.Lock() {
		s := &state.queues[q.id]
		state.mem[queueSlot{q.id, s.next}] = v
		s.next += 1
	}
	q.size <- struct{}{}
	return nil
}

func (q *Queue[T]) Recv(ctx context.Context) (T, error) {
	if _, err := utils.Recv(ctx, q.size); err != nil {
		return utils.Zero[T](), err
	}
	var res T
	for state := range q.pool.state.Lock() {
		s := &state.queues[q.id]
		slot := queueSlot{q.id, s.first}
		s.first += 1
		res = state.mem[slot]
		delete(state.mem, slot)
	}
	<-q.pool.size
	return res, nil
}
