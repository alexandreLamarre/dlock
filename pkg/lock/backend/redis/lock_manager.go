package redis

import (
	"context"
	"log/slog"

	"github.com/alexandreLamarre/dlock/pkg/lock"
	"github.com/go-redsync/redsync/v4/redis"
)

type LockManager struct {
	ctx    context.Context
	pools  []redis.Pool
	quorum int

	prefix string

	lg *slog.Logger
}

var _ lock.LockManager = (*LockManager)(nil)

func NewLockManager(
	ctx context.Context,
	prefix string,
	pools []redis.Pool,
	lg *slog.Logger,
) *LockManager {
	return &LockManager{
		ctx:    ctx,
		pools:  pools,
		prefix: prefix,
		quorum: len(pools)/2 + 1,
		lg:     lg,
	}
}

func (lm *LockManager) NewLock(key string, opt ...lock.LockOption) lock.Lock {
	options := lock.DefaultLockOptions()
	options.Apply(opt...)
	return NewLock(lm.pools, lm.quorum, lm.prefix, key, lm.lg, options)
}
