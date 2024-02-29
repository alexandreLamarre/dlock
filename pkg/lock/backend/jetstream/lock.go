package jetstream

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/alexandreLamarre/dlock/pkg/lock"
	backoffv2 "github.com/lestrrat-go/backoff/v2"
	"github.com/nats-io/nats.go"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/trace"
)

type Lock struct {
	prefix string
	key    string

	js nats.JetStreamContext
	*lock.LockOptions

	scheduler *lock.LockScheduler
	mutex     *jetstreamMutex

	lg *slog.Logger
}

var _ lock.Lock = (*Lock)(nil)

func NewLock(js nats.JetStreamContext, prefix, key string, lg *slog.Logger, options *lock.LockOptions) *Lock {
	return &Lock{
		prefix:      prefix,
		key:         key,
		js:          js,
		lg:          lg.With("key", key),
		LockOptions: options,
		scheduler:   lock.NewLockScheduler(),
	}
}

func (l *Lock) acquire(ctx context.Context, retrier *backoffv2.Policy) (<-chan struct{}, error) {
	var curErr error
	mutex := newJetstreamMutex(l.lg, l.js, l.prefix, l.key, l.LockOptions)
	done, err := mutex.tryLock()
	curErr = err
	if err == nil {
		l.mutex = &mutex
		return done, nil
	}
	if retrier != nil {
		ret := *retrier
		acq := ret.Start(ctx)
		for backoffv2.Continue(acq) {
			done, err := mutex.tryLock()
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

func (l *Lock) lock(ctx context.Context, retrier *backoffv2.Policy) (<-chan struct{}, error) {
	if l.TracingEnabled() {
		ctxSpan, span := l.Tracer.Start(ctx, "Lock/jetstream-lock", trace.WithAttributes())
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

func (l *Lock) Lock(ctx context.Context) (<-chan struct{}, error) {
	retry := lo.ToPtr(backoffv2.Constant(
		backoffv2.WithMaxRetries(0),
		backoffv2.WithInterval(LockRetryDelay),
		backoffv2.WithJitterFactor(0.1),
	))
	return l.lock(ctx, retry)
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

func (l *Lock) TryLock(ctx context.Context) (acquired bool, done <-chan struct{}, err error) {
	closureDone, err := l.lock(ctx, nil)
	if err != nil {
		// hack : jetstream client does not have a stronly typed error for : maxium consumers limit reached
		if strings.Contains(err.Error(), "maximum consumers limit reached") {
			// the request has gone through but someone else has the lock
			return false, nil, nil
		}
		return false, nil, err
	}
	return true, closureDone, nil
}
