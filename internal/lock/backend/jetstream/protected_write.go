package jetstream

import (
	"context"

	"github.com/alexandreLamarre/dlock/pkg/lock"
)

type pwLock struct {
}

var _ lock.Lock = (*pwLock)(nil)

func (l *pwLock) Lock(ctx context.Context) (expired <-chan struct{}, err error) {
	return nil, lock.ErrLockTypeNotImplemented
}

func (l *pwLock) TryLock(ctx context.Context) (acquired bool, expired <-chan struct{}, err error) {
	return false, nil, lock.ErrLockTypeNotImplemented
}

func (l *pwLock) Unlock() error {
	return lock.ErrLockTypeNotImplemented
}
