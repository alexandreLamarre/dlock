package redis

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/alexandreLamarre/dlock/pkg/lock"
	"github.com/alexandreLamarre/dlock/pkg/logger"
	"github.com/go-redsync/redsync/v4/redis"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type redisMutex struct {
	lg       *slog.Logger
	prefix   string
	mutexKey string

	internalDone chan struct{}
	*lock.LockOptions

	quorum int
	pools  []redis.Pool

	uuid string

	// TODO : make better
	until time.Time

	// TODO : all the following are unused
	// expiry time.Duration
	// driftFactor   float64 // nolint:unused
	// timeoutFactor float64
	// fencingToken  string // this should add extra consistency, can be added to genValue instead
	// parentCtx     context.Context
}

func newRedisMutex(
	prefix, key string,
	quorum int,
	pools []redis.Pool,
	lg *slog.Logger,
	opts *lock.LockOptions,
) redisMutex {
	return redisMutex{
		lg:           lg.With("prefix", prefix, "key", key, "quorum", quorum),
		prefix:       prefix,
		mutexKey:     key,
		internalDone: make(chan struct{}),
		LockOptions:  opts,
		quorum:       quorum,
		pools:        pools,
	}
}

func (m *redisMutex) scopedToken() string {
	return uuid.New().String()
}

func (m *redisMutex) actOnPoolsAsync(actFn func(redis.Pool) (bool, error)) (int, error) {
	type result struct {
		Node   int
		Status bool
		Err    error
	}

	ch := make(chan result)
	for node, pool := range m.pools {
		go func(node int, pool redis.Pool) {
			r := result{Node: node}
			r.Status, r.Err = actFn(pool)
			ch <- r
		}(node, pool)
	}
	n := 0
	var taken []int
	var err error
	for range m.pools {
		r := <-ch
		if r.Status {
			n++
		} else if r.Err != nil {
			err = errors.Join(err, &RedisError{Node: r.Node, Err: r.Err})
		} else {
			taken = append(taken, r.Node)
			err = errors.Join(err, &ErrNodeTaken{Node: r.Node})
		}
	}
	if len(taken) >= m.quorum {
		m.lg.With("taken", taken).Debug("consensus reached elsewhere on given operation")
		return n, ErrTaken
	}
	return n, err
}

func (m *redisMutex) key() string {
	return m.prefix + "-" + m.mutexKey
}

func (m *redisMutex) acquire(ctx context.Context, pool redis.Pool, value string) (bool, error) {
	m.lg.With("fenced", value).Debug("acquiring lock...")
	conn, err := pool.Get(ctx)
	if err != nil {
		return false, err
	}
	defer conn.Close()
	reply, err := conn.SetNX(m.key(), value, LockExpiry)
	if err != nil {
		m.lg.With("fenced", value).Error("failed to acquire lock", logger.Err(err))
		return false, err
	}
	m.lg.With("fenced", value).Debug(fmt.Sprintf("acquired lock? %v", reply))
	return reply, nil
}

func (m *redisMutex) lock(ctx context.Context) (<-chan struct{}, error) {
	uuid := m.scopedToken()

	m.uuid = uuid

	start := time.Now()

	n, lockErr := func() (int, error) {
		ctx, ca := context.WithTimeout(ctx, ackTimeoutFactor())
		defer ca()
		return m.actOnPoolsAsync(func(pool redis.Pool) (bool, error) {
			return m.acquire(ctx, pool, uuid)
		})
	}()

	now := time.Now()
	expiredC := lo.Async(m.keepalive)

	until := now.Add(LockExpiry - now.Sub(start) - expiryDriftFactor())
	if n >= m.quorum && now.Before(until) {
		m.lg.Debug("lock acquired and valid")
		m.uuid = uuid
		m.until = until
		return expiredC, nil
	}

	m.lg.Debug("lock not acquired, or lock acquired but already timed out")
	// otherwise, lock should already be expired, due to latency in the system
	func() (int, error) {
		ctx, ca := context.WithTimeout(ctx, LockExpiry)
		defer ca()
		return m.actOnPoolsAsync(func(pool redis.Pool) (bool, error) {
			return m.release(ctx, pool, uuid)
		})
	}()

	return expiredC, lockErr
}

func (m *redisMutex) teardown() {
	defer close(m.internalDone)
	select {
	case m.internalDone <- struct{}{}:
	default:
	}
}

func (m *redisMutex) unlock() (bool, error) {
	defer m.teardown()
	m.lg.Debug("unlock requested")
	ctx := context.Background()
	var span trace.Span
	if m.TracingEnabled() {
		ctx, span = m.Tracer.Start(context.Background(), "Unlock/redis-unlock", trace.WithAttributes(
			attribute.KeyValue{
				Key:   "key",
				Value: attribute.StringValue(m.key()),
			},
		))
		defer span.End()
	}

	ctx, ca := context.WithTimeout(context.Background(), LockExpiry)
	defer ca()

	n, err := m.actOnPoolsAsync(func(pool redis.Pool) (bool, error) {
		return m.release(ctx, pool, m.uuid)
	})
	if n < m.quorum {
		m.lg.With(logger.Err(err)).Warn("failed to release lock no consensus : ")
		m.RecordError(span, err)
		return false, err
	}
	return true, nil
}

var deleteScript = redis.NewScript(1, `
	if redis.call("GET", KEYS[1]) == ARGV[1] then
		return redis.call("DEL", KEYS[1])
	else
		return 0
	end
`)

func (m *redisMutex) release(ctx context.Context, pool redis.Pool, value string) (bool, error) {
	m.lg.With("fenced", m.uuid).Debug("release lock requested...")
	conn, err := pool.Get(ctx)
	if err != nil {
		return false, err
	}
	defer conn.Close()
	status, err := conn.Eval(deleteScript, m.key(), value)
	if err != nil {
		return false, err
	}
	m.lg.With("fenced", m.uuid).Debug(fmt.Sprintf("release lock status : %d", status))
	return status != int64(0), nil
}

var touchScript = redis.NewScript(1, `
	if redis.call("GET", KEYS[1]) == ARGV[1] then
		return redis.call("PEXPIRE", KEYS[1], ARGV[2])
	else
		return 0
	end
`)

func (m *redisMutex) touch(ctx context.Context, pool redis.Pool, value string, expiry int) (bool, error) {
	conn, err := pool.Get(ctx)
	if err != nil {
		return false, nil
	}
	defer conn.Close()
	status, err := conn.Eval(touchScript, m.key(), value)
	if err != nil {
		return false, err
	}
	m.lg.With("fenced", m.uuid).Debug(fmt.Sprintf("touch lock status : %d", status))
	return status != int64(0), nil
}

func expiryDriftFactor() time.Duration {
	return time.Duration(int64(float64(LockExpiry) * LockDriftFactor))
}

func ackTimeoutFactor() time.Duration {
	return time.Duration(int64(float64(LockExpiry) * LockTimeoutFactor))
}

func (m *redisMutex) extend(ctx context.Context) (bool, error) {
	m.lg.Debug("extending lock expiry...")
	start := time.Now()
	n, err := m.actOnPoolsAsync(func(pool redis.Pool) (bool, error) {
		// cast to milliseconds
		return m.touch(ctx, pool, m.uuid, int(LockExpiry/time.Millisecond))
	})
	if n < m.quorum {
		m.lg.With(logger.Err(err)).Warn("failed to extend lock expiry : ")
		return false, err
	}
	now := time.Now()
	until := now.Add(LockExpiry - now.Sub(start) - expiryDriftFactor())
	if now.Before(until) {
		m.until = until
		return true, nil
	}
	m.lg.Warn("failed to extend lock expiry : lock already expired")
	return false, ErrExtendFailed
}

func (m *redisMutex) keepalive() struct{} {
	// TODO : maybe replace with unused parentCtx
	ctx := context.TODO()
	t := time.NewTicker(LockExtendDelay)
	defer t.Stop()
	for {
		select {
		case <-m.internalDone:
			return struct{}{}
		case <-ctx.Done():
			return struct{}{}
		case <-t.C:
			extended, err := m.extend(ctx)
			if err != nil {
				m.lg.With(logger.Err(err), "extended", extended).Warn("failed to extend lock")
			}
			now := time.Now()
			if now.After(m.until) {
				return struct{}{}
			}
		}
	}
}
