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
	backoffv2 "github.com/lestrrat-go/backoff/v2"
	"github.com/samber/lo"
)

var (
	LockExpiry        time.Duration = 60 * time.Second
	LockRetryDelay    time.Duration = 100 * time.Millisecond
	LockExtendDelay   time.Duration = 333 * time.Millisecond
	LockDriftFactor   float64       = 0.01
	LockTimeoutFactor float64       = 0.05
)

type Lock struct {
	pools  []redis.Pool
	quorum int

	prefix string
	key    string
	lg     *slog.Logger

	scheduler *lock.LockScheduler
	mutex     *redisMutex

	*lock.LockOptions
}

func NewLock(
	pools []redis.Pool,
	quorum int,
	prefix, key string,
	lg *slog.Logger,
	opts *lock.LockOptions,
) *Lock {
	return &Lock{
		prefix:      prefix,
		key:         key,
		pools:       pools,
		quorum:      quorum,
		lg:          lg,
		scheduler:   lock.NewLockScheduler(),
		LockOptions: opts,
	}
}

var _ lock.Lock = (*Lock)(nil)

func (l *Lock) Lock(ctx context.Context) (expired <-chan struct{}, err error) {
	retry := lo.ToPtr(
		backoffv2.Constant(
			backoffv2.WithMaxRetries(0),
			backoffv2.WithInterval(LockRetryDelay),
			backoffv2.WithJitterFactor(0.1),
		),
	)
	return l.lock(ctx, retry)
}

func (l *Lock) TryLock(ctx context.Context) (acquired bool, expired <-chan struct{}, err error) {
	closureDone, err := l.lock(ctx, nil)
	if err != nil {
		if errors.Is(err, ErrTaken) {
			l.lg.Debug(
				fmt.Sprintf(
					"lock already acquired by someone else : >= quorum(%d)",
					l.quorum,
				),
			)
			return false, nil, nil
		}
		l.lg.With(logger.Err(err)).Error("failed to acquire lock")
		return false, nil, err
	}
	return true, closureDone, nil
}

func (l *Lock) lock(ctx context.Context, retrier *backoffv2.Policy) (expired <-chan struct{}, err error) {
	if l.TracingEnabled() {
		ctxSpan, span := l.Tracer.Start(ctx, "Lock/redis-lock")
		defer span.End()
		ctx = ctxSpan
	}
	// https://github.com/lestrrat-go/backoff/issues/31
	ctxca, ca := context.WithCancel(ctx)
	defer ca()

	var closureDone <-chan struct{}
	if err := l.scheduler.Schedule(func() error {
		done, err := l.acquire(ctxca, retrier)
		if err != nil {
			return err
		}
		closureDone = done
		return nil
	}); err != nil {
		return nil, err
	}

	return closureDone, nil
}

func (l *Lock) acquire(ctx context.Context, retrier *backoffv2.Policy) (<-chan struct{}, error) {
	var curErr error
	mutex := newRedisMutex(l.prefix, l.key, l.quorum, l.pools, l.lg, l.LockOptions)
	done, err := mutex.lock(ctx)
	curErr = err
	if err == nil {
		l.mutex = &mutex
		return done, nil
	}
	if retrier != nil {
		ret := *retrier
		acq := ret.Start(ctx)
		for backoffv2.Continue(acq) {
			done, err := mutex.lock(ctx)
			curErr = err
			if err == nil {
				l.mutex = &mutex
				return done, nil
			}
		}
		return nil, errors.Join(ctx.Err(), curErr)
	}
	return nil, curErr
}

func (l *Lock) Unlock() error {
	if err := l.scheduler.Done(func() error {
		if l.mutex == nil {
			return nil
		}
		mutex := *l.mutex
		go func() {
			if unlocked, err := mutex.unlock(); err != nil {
				l.lg.With(logger.Err(err), "unlocked", unlocked).Warn("failed to unlock")
			}
		}()
		l.mutex = nil
		return nil
	}); err != nil {
		return err
	}
	return nil
}
