package jetstream

import (
	"context"
	"log/slog"

	"github.com/alexandreLamarre/dlock/pkg/constants"
	"github.com/alexandreLamarre/dlock/pkg/lock"
	"github.com/alexandreLamarre/dlock/pkg/lock/broker"
	"github.com/alexandreLamarre/dlock/pkg/logger"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/trace"
)

func init() {
	broker.RegisterLockBroker(
		constants.JetstreamLockManager,
		func(ctx context.Context, l broker.LockBroker) (lock.LockManager, error) {
			l.Lg.Info("acquiring jetstream client...")
			cli, err := AcquireJetstreamConn(ctx, l.Config.JetstreamClientSpec, l.Lg)
			if err != nil {
				l.Lg.With(logger.Err(err)).Warn("failed to acquired jetstream client")
				return nil, err
			}
			l.Lg.Info("acquired jetstream client")
			return NewLockManager(ctx, cli, "lock", l.Tracer, l.Lg), nil
		},
	)
}

// Requires jetstream 2.9+
type LockManager struct {
	ctx    context.Context
	js     nats.JetStreamContext
	tracer trace.Tracer

	lg *slog.Logger

	prefix string
}

var _ lock.LockManager = (*LockManager)(nil)

func NewLockManager(
	ctx context.Context,
	js nats.JetStreamContext,
	prefix string,
	tracer trace.Tracer,
	lg *slog.Logger,
) *LockManager {
	prefix = sanitizePrefix(prefix)
	return &LockManager{
		ctx:    ctx,
		js:     js,
		lg:     lg,
		prefix: prefix,
		tracer: tracer,
	}
}

func (l *LockManager) Health(ctx context.Context) (conditions []string, err error) {
	// We could be using jsm.go here: https://github.com/nats-io/jsm.go
	// for now, we'll count account info as a health check
	_, err = l.js.AccountInfo()
	if err != nil {
		return nil, err
	}
	return []string{}, nil
}

func (l *LockManager) NewLock(key string, opts ...lock.LockOption) lock.Lock {
	options := lock.DefaultLockOptions()
	options.Apply(opts...)
	return NewLock(l.js, l.prefix, key, l.lg, options)
}
