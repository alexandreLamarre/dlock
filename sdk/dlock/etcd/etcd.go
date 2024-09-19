package etcd

import (
	"log/slog"

	"github.com/alexandreLamarre/dlock/internal/lock/backend/etcd"
	"github.com/alexandreLamarre/dlock/pkg/lock"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.opentelemetry.io/otel/trace"
)

func NewLock(
	lg *slog.Logger,
	client *clientv3.Client,
	prefix, key string,
	options *lock.LockOptions,
) lock.Lock {
	return etcd.NewEtcdLock(lg, client, prefix, key, options)
}

func NewLockManager(
	lg *slog.Logger,
	client *clientv3.Client,
	prefix string,
	tracer trace.Tracer,
) lock.LockManager {
	return etcd.NewEtcdLockManager(client, prefix, tracer, lg)
}
