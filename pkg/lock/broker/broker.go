package broker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/alexandreLamarre/dlock/pkg/config/v1alpha1"
	"github.com/alexandreLamarre/dlock/pkg/lock"
	"github.com/alexandreLamarre/dlock/pkg/lock/backend/etcd"
	"github.com/alexandreLamarre/dlock/pkg/lock/backend/jetstream"
	"github.com/alexandreLamarre/dlock/pkg/lock/backend/redis"
	"github.com/alexandreLamarre/dlock/pkg/logger"
	"github.com/alexandreLamarre/dlock/pkg/util"
	goredislib "github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/trace"
)

type LockBroker struct {
	lg     *slog.Logger
	config *v1alpha1.LockServerConfig
	tracer trace.Tracer
}

func NewLockBroker(
	lg *slog.Logger,
	cfg *v1alpha1.LockServerConfig,
	tracer trace.Tracer,
) LockBroker {
	return LockBroker{
		config: cfg,
		lg:     lg,
		tracer: tracer,
	}
}

// LockManager blocks until it acquires the client connection, or returns an error
// when an unrecoverable error is hit
func (l LockBroker) LockManager(ctx context.Context) (lock.LockManager, error) {
	if l.config.EtcdClientSpec != nil {
		l.lg.Info("acquiring etcd client...")
		cli, err := etcd.NewEtcdClient(ctx, l.config.EtcdClientSpec)
		if err != nil {
			l.lg.With(logger.Err(err)).Warn("failed to acquired etcd client")
			return nil, err
		}
		errs := []error{}
		for _, endp := range l.config.EtcdClientSpec.Endpoints {
			ctxT, caT := context.WithTimeout(ctx, 1*time.Second)
			defer caT()
			_, err := cli.Status(ctxT, endp)
			if err != nil {
				errs = append(errs, err)
			}
		}
		if len(errs) > 0 {
			return nil, errors.Join(errs...)
		}
		l.lg.Info("acquired etcd client")
		return etcd.NewEtcdLockManager(cli, "lock", l.tracer, l.lg), nil
	}

	if l.config.JetstreamClientSpec != nil {
		l.lg.Info("acquiring jetstream client...")
		cli, err := jetstream.AcquireJetstreamConn(ctx, l.config.JetstreamClientSpec, l.lg)
		if err != nil {
			l.lg.With(logger.Err(err)).Warn("failed to acquired jetstream client")
			return nil, err
		}
		l.lg.Info("acquired jetstream client")
		return jetstream.NewLockManager(ctx, cli, "lock", l.tracer, l.lg), nil
	}

	if l.config.RedisClientSpec != nil {
		l.lg.Info("acquiring redis client...")
		cli := redis.AcquireRedisPool([]*goredislib.Options{
			{
				Addr:    l.config.RedisClientSpec.Addr,
				Network: l.config.RedisClientSpec.Network,
			},
		})
		// TODO : ping redis pool for health before starting
		l.lg.Info("acquired redis client")
		return redis.NewLockManager(ctx, "lock", cli, l.lg), nil
	}
	return nil, fmt.Errorf("unknown lock manager type in config : %s", util.Must(json.Marshal(l.config)))
}
