package redis

import (
	"context"
	"strings"
	"time"

	"github.com/go-redsync/redsync/v4/redis"
	redsyncgoredis "github.com/go-redsync/redsync/v4/redis/goredis/v9"
	goredislib "github.com/redis/go-redis/v9"

	"github.com/redis/rueidis"
	"github.com/redis/rueidis/rueidiscompat"
)

func AcquireRedisPool(
	clients []*goredislib.Options,
) []redis.Pool {
	pools := make([]redis.Pool, len(clients))
	for i, clientOps := range clients {
		client := goredislib.NewClient(clientOps)
		pools[i] = redsyncgoredis.NewPool(client)
	}
	return pools
}

// The following code is copied from:
// https://github.com/go-redsync/redsync/blob/master/redis/rueidis/rueidis.go

type pool struct {
	delegate rueidiscompat.Cmdable
}

func (p *pool) Get(ctx context.Context) (redis.Conn, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	return &conn{p.delegate, ctx}, nil
}

// NewPool returns a rueidis-based pool implementation.
func NewPool(delegate rueidiscompat.Cmdable) redis.Pool {
	return &pool{delegate}
}

type conn struct {
	delegate rueidiscompat.Cmdable
	ctx      context.Context
}

func (c *conn) Get(name string) (string, error) {
	value, err := c.delegate.Get(c.ctx, name).Result()
	return value, noErrNil(err)
}

func (c *conn) Set(name string, value string) (bool, error) {
	reply, err := c.delegate.Set(c.ctx, name, value, 0).Result()
	return reply == "OK", err
}

func (c *conn) SetNX(name string, value string, expiry time.Duration) (bool, error) {
	return c.delegate.SetNX(c.ctx, name, value, expiry).Result()
}

func (c *conn) PTTL(name string) (time.Duration, error) {
	return c.delegate.PTTL(c.ctx, name).Result()
}

func (c *conn) Eval(script *redis.Script, keysAndArgs ...interface{}) (interface{}, error) {
	keys := make([]string, script.KeyCount)
	args := keysAndArgs

	if script.KeyCount > 0 {
		for i := 0; i < script.KeyCount; i++ {
			keys[i] = keysAndArgs[i].(string)
		}
		args = keysAndArgs[script.KeyCount:]
	}

	v, err := c.delegate.EvalSha(c.ctx, script.Hash, keys, args...).Result()
	if err != nil && strings.Contains(err.Error(), "NOSCRIPT ") {
		v, err = c.delegate.Eval(c.ctx, script.Src, keys, args...).Result()
	}
	return v, noErrNil(err)
}

func (c *conn) Close() error {
	// Not needed for this library
	return nil
}

func noErrNil(err error) error {
	if err == rueidis.Nil {
		return nil
	}
	return err
}
