package service

import (
	"context"

	"github.com/sei-protocol/sei-load/utils/scope"
)

type Scope = scope.Scope

type JoinHandle[R any] = scope.JoinHandle[R]

func Spawn1[R any](s Scope, t func() (R, error)) JoinHandle[R] {
	return scope.Spawn1(s, t)
}

func Run(ctx context.Context, main func(context.Context, Scope) error) error {
	return scope.Run(ctx, main)
}

func Run1[R any](ctx context.Context, main func(context.Context, Scope) (R, error)) (R, error) {
	return scope.Run1(ctx, main)
}
