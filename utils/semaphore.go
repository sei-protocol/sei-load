package utils

import (
	"context"
)

// Semaphore provides a way to bound concurrenct access to a resource.
type Semaphore struct {
	ch chan struct{}
}

// NewSemaphore constructs a new semaphore with n permits.
func NewSemaphore(n int) *Semaphore {
	return &Semaphore{ch: make(chan struct{}, n)}
}

// Acquire acquires a permit from the semaphore.
// Blocks until a permit is available.
func (s *Semaphore) Acquire(ctx context.Context) (release func(), err error) {
	if err := Send(ctx, s.ch, struct{}{}); err != nil {
		return nil, err
	}
	return func() { <-s.ch }, nil
}

// TryAcquire acquires a permit without blocking. It returns the release func
// and true if a permit was available, or nil and false if all permits are held.
// Used by callers that must never block waiting for capacity (e.g. an open-loop
// scheduler that drops rather than throttling its clock).
func (s *Semaphore) TryAcquire() (release func(), ok bool) {
	select {
	case s.ch <- struct{}{}:
		return func() { <-s.ch }, true
	default:
		return nil, false
	}
}
