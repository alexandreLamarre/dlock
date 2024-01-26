package redis

import (
	"github.com/alexandreLamarre/dlock/pkg/lock"
	"github.com/go-redis/redis"
)

type LockManager struct {
	client *redis.Client
}

var _ lock.LockManager = (*LockManager)(nil)

func (lm *LockManager) NewLock(key string, opt ...lock.LockOption) lock.Lock {
	return &Lock{
		client: lm.client,
	}
}
