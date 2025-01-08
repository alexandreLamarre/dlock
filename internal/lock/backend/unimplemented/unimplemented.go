package unimplemented

import (
	"context"

	"github.com/alexandreLamarre/dlock/pkg/lock"
)

type UnimplementedLock struct{}

var _ lock.Lock = (*UnimplementedLock)(nil)

func (u *UnimplementedLock) Lock(ctx context.Context) (expired <-chan struct{}, err error) {
	return nil, lock.ErrLockTypeNotImplemented
}

func (u *UnimplementedLock) TryLock(ctx context.Context) (acquired bool, expired <-chan struct{}, err error) {
	return false, nil, lock.ErrLockTypeNotImplemented
}

func (u *UnimplementedLock) Unlock() error {
	return lock.ErrLockTypeNotImplemented
}
