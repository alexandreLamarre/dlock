package jetstream

import (
	"context"

	"github.com/alexandreLamarre/dlock/pkg/lock"
)

type prLock struct {
}

var _ lock.Lock = (*prLock)(nil)

func (l *prLock) Lock(ctx context.Context) (expired <-chan struct{}, err error) {
	return nil, lock.ErrLockTypeNotImplemented
}

func (l *prLock) TryLock(ctx context.Context) (acquired bool, expired <-chan struct{}, err error) {
	return false, nil, lock.ErrLockTypeNotImplemented
}

func (l *prLock) Unlock() error {
	return lock.ErrLockTypeNotImplemented
}
