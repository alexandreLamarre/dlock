package redis

import (
	"context"

	"github.com/alexandreLamarre/dlock/pkg/lock"
	"github.com/go-redis/redis"
)

type Lock struct {
	client *redis.Client
}

var _ lock.Lock = (*Lock)(nil)

func (l *Lock) Lock(ctx context.Context) (expired <-chan struct{}, err error) {
	l.client.SetNX("key", "value", 0)
	return nil, nil
}

func (l *Lock) TryLock(ctx context.Context) (acquired bool, expired <-chan struct{}, err error) {
	return false, nil, nil
}

func (l *Lock) Unlock() error {
	return nil
}
