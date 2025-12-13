package redis

import (
	"context"
	"errors"
	"log/slog"

	"github.com/alexandreLamarre/dlock/pkg/constants"
	"github.com/alexandreLamarre/dlock/pkg/lock"
	"github.com/alexandreLamarre/dlock/pkg/lock/broker"
	"github.com/go-redsync/redsync/v4/redis"
	goredislib "github.com/redis/go-redis/v9"
)

var pingScript = redis.NewScript(0, `
	local info = redis.call("INFO")
	return info
`, "")

func init() {
	broker.RegisterLockBroker(
		constants.RedisLockManager,
		func(ctx context.Context, l broker.LockBroker) (lock.LockManager, error) {
			l.Lg.Info("acquiring redis client...")
			cli := AcquireRedisPool([]*goredislib.Options{
				{
					Addr:    l.Config.RedisClientSpec.Addr,
					Network: l.Config.RedisClientSpec.Network,
				},
			})
			// TODO : ping redis pool for health before starting
			l.Lg.Info("acquired redis client")
			return NewLockManager(ctx, "lock", cli, l.Lg), nil
		},
	)
}

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

func (lm *LockManager) Health(ctx context.Context) (conditions []string, err error) {
	for i, pool := range lm.pools {
		conn, poolErr := pool.Get(ctx)
		if poolErr != nil {
			err = errors.Join(err, poolErr)
			continue
		}
		info, evalErr := conn.Eval(pingScript)
		if evalErr != nil {
			err = errors.Join(err, evalErr)
			continue
		}
		lm.lg.With("pool", i).Info("got status : %s", info)
	}
	return []string{}, nil
}

func (lm *LockManager) NewLock(key string, opt ...lock.LockOption) lock.Lock {
	options := lock.DefaultLockOptions()
	options.Apply(opt...)
	return NewLock(lm.pools, lm.quorum, lm.prefix, key, lm.lg, options)
}
