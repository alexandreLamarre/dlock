package broker

import (
	"context"
	"log/slog"

	"github.com/alexandreLamarre/dlock/pkg/config/v1alpha1"
	"github.com/alexandreLamarre/dlock/pkg/lock"
	"github.com/alexandreLamarre/dlock/pkg/lock/backend/etcd"
	"github.com/alexandreLamarre/dlock/pkg/lock/backend/jetstream"
	"github.com/alexandreLamarre/dlock/pkg/lock/backend/redis"
	"github.com/alexandreLamarre/dlock/pkg/logger"
	goredislib "github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/trace"
)

func NewLockManager(ctx context.Context, tracer trace.Tracer, lg *slog.Logger, config *v1alpha1.LockServerConfig) lock.LockManager {
	if config.EtcdClientSpec != nil {
		lg.Info("acquiring etcd client ...")
		cli, err := etcd.NewEtcdClient(ctx, config.EtcdClientSpec)
		if err != nil {
			lg.With(logger.Err(err)).Warn("failed to acquired etcd client")
		}
		return etcd.NewEtcdLockManager(cli, "lock", tracer, lg)
	}

	if config.JetstreamClientSpec != nil {
		lg.Info("acquiring jetstream client ...")
		cli, err := jetstream.AcquireJetstreamConn(ctx, config.JetstreamClientSpec, lg)
		if err != nil {
			lg.With(logger.Err(err)).Warn("failed to acquired jetstream client")
		}
		return jetstream.NewLockManager(ctx, cli, "lock", tracer, lg)
	}

	if config.RedisClientSpec != nil {
		lg.Info("acquiring redis client ...")
		cli := redis.AcquireRedisPool([]*goredislib.Options{
			{
				Addr:    config.RedisClientSpec.Addr,
				Network: config.RedisClientSpec.Network,
			},
		})
		return redis.NewLockManager(ctx, "lock", cli, lg)
	}

	return nil
}
