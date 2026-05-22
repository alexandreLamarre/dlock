package jetstream

import (
	"context"

	"github.com/alexandreLamarre/dlock/pkg/lock"
)

type cwLock struct {
}

var _ lock.Lock = (*cwLock)(nil)

func (l *cwLock) Lock(ctx context.Context) (expired <-chan struct{}, err error) {
	return nil, lock.ErrLockTypeNotImplemented
}

func (l *cwLock) TryLock(ctx context.Context) (acquired bool, expired <-chan struct{}, err error) {
	return false, nil, lock.ErrLockTypeNotImplemented
}

func (l *cwLock) Unlock() error {
	return lock.ErrLockTypeNotImplemented
}
