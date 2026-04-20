package jetstream

import (
	"context"

	"github.com/alexandreLamarre/dlock/pkg/lock"
)

type crLock struct {
}

var _ lock.Lock = (*crLock)(nil)

func (l *crLock) Lock(ctx context.Context) (expired <-chan struct{}, err error) {
	return nil, lock.ErrLockTypeNotImplemented
}

func (l *crLock) TryLock(ctx context.Context) (acquired bool, expired <-chan struct{}, err error) {
	return false, nil, lock.ErrLockTypeNotImplemented
}

func (l *crLock) Unlock() error {
	return lock.ErrLockTypeNotImplemented
}
