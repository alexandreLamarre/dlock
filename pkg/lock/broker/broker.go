package broker

import (
	"context"
	"log/slog"

	"github.com/alexandreLamarre/dlock/pkg/config/v1alpha1"
	"github.com/alexandreLamarre/dlock/pkg/lock"
	"github.com/alexandreLamarre/dlock/pkg/lock/backend/etcd"
	"github.com/alexandreLamarre/dlock/pkg/lock/backend/jetstream"
	"github.com/alexandreLamarre/dlock/pkg/logger"
)

func NewLockManager(ctx context.Context, lg *slog.Logger, config *v1alpha1.LockServerConfig) lock.LockManager {
	if config.EtcdStorageSpec != nil {
		cli, err := etcd.NewEtcdClient(ctx, config.EtcdStorageSpec)
		if err != nil {
			lg.With(logger.Err(err)).Warn("failed to acquired etcd client")
		}
		return etcd.NewEtcdLockManager(cli, "lock", lg)
	}

	if config.JetStreamStorageSpec != nil {
		cli, err := jetstream.AcquireJetstreamConn(ctx, config.JetStreamStorageSpec, lg)
		if err != nil {
			lg.With(logger.Err(err)).Warn("failed to acquired jetstream client")
		}
		return jetstream.NewLockManager(ctx, cli, "lock", lg)
	}

	return nil
}
