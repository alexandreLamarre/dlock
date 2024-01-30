package embedded

import (
	"context"

	"github.com/alexandreLamarre/dlock/pkg/lock"
)

type Lock struct {
}

var _ lock.Lock = (*Lock)(nil)

func NewLock() *Lock {
	return &Lock{}
}

func (l *Lock) Key() string {
	return ""
}

func (l *Lock) Lock(ctx context.Context) (expired <-chan struct{}, err error) {
	panic("implement me")
}

func (l *Lock) TryLock(ctx context.Context) (acquired bool, expired <-chan struct{}, err error) {
	panic("implement me")
}

func (l *Lock) Unlock() error {
	panic("implement me")
}
