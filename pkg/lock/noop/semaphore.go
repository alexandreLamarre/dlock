package noop

import (
	"context"
	"fmt"

	"github.com/alexandreLamarre/dlock/pkg/lock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type noopWeightedSemaphore struct {
	backend string
}

func NewNoopWeightedSemaphore(backend string) lock.WeightedSemaphore {
	return &noopWeightedSemaphore{backend: backend}
}

var _ lock.WeightedSemaphore = &noopWeightedSemaphore{}

func (n *noopWeightedSemaphore) Acquire(_ context.Context) (<-chan int, error) {
	return nil, status.Error(codes.Unimplemented, fmt.Sprintf("%s backend does not support weighted semaphores", n.backend))
}

func (n *noopWeightedSemaphore) TryAcquire(_ context.Context) (acquired bool, expired <-chan int, err error) {
	return false, nil, status.Error(codes.Unimplemented, fmt.Sprintf("%s backend does not support weighted semaphores", n.backend))
}

func (n *noopWeightedSemaphore) Release() error {
	return status.Error(codes.Unimplemented, fmt.Sprintf("%s backend does not support weighted semaphores", n.backend))
}
