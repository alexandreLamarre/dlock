package broker

import (
	"context"
	"log/slog"

	"github.com/alexandreLamarre/dlock/pkg/config/v1alpha1"
	"github.com/alexandreLamarre/dlock/pkg/lock"
	"github.com/alexandreLamarre/dlock/pkg/lock/backend/etcd"
	"github.com/alexandreLamarre/dlock/pkg/lock/backend/jetstream"
)

func NewLockManager(ctx context.Context, lg *slog.Logger, config *v1alpha1.LockServerConfig) lock.LockManager {
	if config.EtcdStorageSpec != nil {
		cli, _ := etcd.NewEtcdClient(ctx, config.EtcdStorageSpec)
		return etcd.NewEtcdLockManager(cli, "lock", lg)
	}

	if config.JetStreamStorageSpec != nil {
		cli, _ := jetstream.AcquireJetstreamConn(ctx, config.JetStreamStorageSpec, lg)
		return jetstream.NewLockManager(ctx, cli, "lock", lg)
	}

	return nil
}
