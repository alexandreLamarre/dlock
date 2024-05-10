package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/alexandreLamarre/dlock/pkg/config/v1alpha1"
	"github.com/alexandreLamarre/dlock/pkg/constants"
	"github.com/alexandreLamarre/dlock/pkg/lock"
	"github.com/alexandreLamarre/dlock/pkg/util"
	"go.opentelemetry.io/otel/trace"
)

type LockBroker struct {
	Lg     *slog.Logger
	Config *v1alpha1.LockServerConfig
	Tracer trace.Tracer
}

func NewLockBroker(
	lg *slog.Logger,
	cfg *v1alpha1.LockServerConfig,
	tracer trace.Tracer,
) LockBroker {
	return LockBroker{
		Config: cfg,
		Lg:     lg,
		Tracer: tracer,
	}
}

// LockManager blocks until it acquires the client connection, or returns an error
// when an unrecoverable error is hit
func (l LockBroker) LockManager(ctx context.Context) (lock.LockManager, error) {
	backends := strings.Join(brokerKeys(), ",")
	l.Lg.Info(fmt.Sprintf("Available lock managers : %s", backends))
	if l.Config.EtcdClientSpec != nil {
		broker, ok := GetLockBroker(constants.EtcdLockManager)
		if !ok {
			return nil, fmt.Errorf("etcd lock manager not registered")
		}
		return broker(ctx, l)
	}
	if l.Config.JetstreamClientSpec != nil {
		broker, ok := GetLockBroker(constants.JetstreamLockManager)
		if !ok {
			return nil, fmt.Errorf("jetstream lock manager not registered")
		}
		return broker(ctx, l)
	}

	if l.Config.RedisClientSpec != nil {
		broker, ok := GetLockBroker(constants.RedisLockManager)
		if !ok {
			return nil, fmt.Errorf("redis lock manager not registered")
		}
		return broker(ctx, l)
	}

	return nil, fmt.Errorf("unknown lock manager type in config : %s", util.Must(json.Marshal(l.Config)))
}
