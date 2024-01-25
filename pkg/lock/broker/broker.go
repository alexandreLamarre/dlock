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
		return etcd.NewEtcdLockManager(nil, "lock", lg)
	}

	if config.JetStreamStorageSpec != nil {
		return jetstream.NewLockManager(ctx, nil, "lock", lg)
	}

	return nil
}
