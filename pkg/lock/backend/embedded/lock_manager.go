package embedded

import (
	"context"

	"github.com/alexandreLamarre/dlock/pkg/lock"
)

type LockManager struct {
}

var _ lock.LockManager = (*LockManager)(nil)

func NewLockManager() *LockManager {
	return &LockManager{}
}

func (lm *LockManager) Health(ctx context.Context) ([]string, error) {
	panic("implement me")
}

func (lm *LockManager) NewLock(key string, opts ...lock.LockOption) lock.Lock {
	panic("implement me")
}
