package redis

import (
	"context"
	"log/slog"

	rl "github.com/alexandreLamarre/dlock/internal/lock/backend/redis"
	"github.com/alexandreLamarre/dlock/pkg/lock"
	"github.com/go-redsync/redsync/v4/redis"
)

func NewLock(
	pools []redis.Pool,
	quorum int,
	prefix, key string,
	lg *slog.Logger,
	opts *lock.LockOptions,
) lock.Lock {
	return rl.NewLock(
		pools,
		quorum,
		prefix,
		key,
		lg,
		opts,
	)
}

func NewLockManager(
	ctx context.Context,
	prefix string,
	pools []redis.Pool,
	lg *slog.Logger,
) lock.LockManager {
	return rl.NewLockManager(
		ctx,
		prefix,
		pools,
		lg,
	)
}
