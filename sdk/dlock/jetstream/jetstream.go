package jetstream

import (
	"context"
	"log/slog"

	"github.com/alexandreLamarre/dlock/internal/lock/backend/jetstream"
	"github.com/alexandreLamarre/dlock/pkg/lock"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/trace"
)

func NewLock(
	lg *slog.Logger,
	js nats.JetStreamContext,
	prefix, key string,
	options *lock.LockOptions,
) lock.Lock {
	return jetstream.NewLock(js, prefix, key, lg, options)
}

func NewLockManager(
	ctx context.Context,
	lg *slog.Logger,
	js nats.JetStreamContext,
	prefix string,
	tracer trace.Tracer,
) lock.LockManager {
	return jetstream.NewLockManager(ctx, js, prefix, tracer, lg)
}
