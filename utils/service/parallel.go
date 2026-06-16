package service

import "github.com/sei-protocol/sei-load/utils/scope"

type ParallelScope = scope.ParallelScope

func Parallel(main func(ParallelScope) error) error {
	return scope.Parallel(main)
}
