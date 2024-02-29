package cassandra

import (
	"context"
	"errors"
	"log/slog"

	"github.com/alexandreLamarre/dlock/pkg/lock"
	"github.com/gocql/gocql"
	backoffv2 "github.com/lestrrat-go/backoff/v2"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/trace"
)

type Lock struct {
	lg *slog.Logger

	prefix string
	key    string

	*lock.LockOptions

	scheduler *lock.LockScheduler
	session   *gocql.Session
	mutex     *mutex
}

var _ lock.Lock = (*Lock)(nil)

func NewCassandraLock(
	lg *slog.Logger,
	session *gocql.Session,
	prefix, key string,
	options *lock.LockOptions,
) *Lock {
	return &Lock{
		lg:          lg,
		session:     session,
		prefix:      prefix,
		key:         key,
		LockOptions: options,
		scheduler:   lock.NewLockScheduler(),
	}
}

func (l *Lock) Lock(ctx context.Context) (expired <-chan struct{}, err error) {
	retry := lo.ToPtr(backoffv2.Constant(
		backoffv2.WithMaxRetries(0),
		backoffv2.WithInterval(LockRetryDelay),
		backoffv2.WithJitterFactor(0.1),
	))
	return l.lock(ctx, retry)
}

func (l *Lock) lock(ctx context.Context, retrier *backoffv2.Policy) (expired <-chan struct{}, err error) {
	if l.TracingEnabled() {
		ctxSpan, span := l.Tracer.Start(ctx, "lock/cassandra-lock", trace.WithAttributes())
		defer span.End()
		ctx = ctxSpan
	}
	// https://github.com/lestrrat-go/backoff/issues/31
	ctxca, ca := context.WithCancel(ctx)
	defer ca()

	var closureDone <-chan struct{}
	if err := l.scheduler.Schedule(func() error {
		done, err := l.acquireLock(ctxca, retrier)
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

func (l *Lock) acquireLock(ctx context.Context, retrier *backoffv2.Policy) (<-chan struct{}, error) {
	var curErr error
	mutex := newMutex(l.lg, l.prefix, l.key, l.session, l.LockOptions)
	done, err := mutex.tryLock(ctx)
	curErr = err
	if err == nil {
		l.mutex = &mutex
		return done, nil
	}
	if retrier != nil {
		ret := *retrier
		acq := ret.Start(ctx)
		for backoffv2.Continue(acq) {
			done, err := mutex.tryLock(ctx)
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

func (l *Lock) TryLock(ctx context.Context) (acquired bool, expired <-chan struct{}, err error) {
	closureDone, err := l.lock(ctx, nil)
	if err != nil {
		// TODO : differentiate between error and lock already acquired
		return false, nil, nil
	}
	return true, closureDone, nil
}

func (l *Lock) Unlock() error {
	if err := l.scheduler.Done(func() error {
		if l.mutex == nil {
			panic("never acquired")
		}
		mutex := *l.mutex
		go func() {
			if err := mutex.unlock(); err != nil {
				l.lg.Error(err.Error())
			}
		}()
		l.mutex = nil
		return nil
	}); err != nil {
		return err
	}
	return nil
}
