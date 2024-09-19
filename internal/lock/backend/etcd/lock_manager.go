package etcd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/alexandreLamarre/dlock/pkg/constants"
	"github.com/alexandreLamarre/dlock/pkg/lock"
	"github.com/alexandreLamarre/dlock/pkg/lock/broker"
	"github.com/alexandreLamarre/dlock/pkg/logger"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.opentelemetry.io/otel/trace"
)

func init() {
	broker.RegisterLockBroker(
		constants.EtcdLockManager,
		func(ctx context.Context, l broker.LockBroker) (lock.LockManager, error) {
			l.Lg.Info("acquiring etcd client...")
			cli, err := NewEtcdClient(ctx, l.Config.EtcdClientSpec)
			if err != nil {
				l.Lg.With(logger.Err(err)).Warn("failed to acquired etcd client")
				return nil, err
			}
			errs := []error{}
			for _, endp := range l.Config.EtcdClientSpec.Endpoints {
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
			l.Lg.Info("acquired etcd client")
			return NewEtcdLockManager(cli, "lock", l.Tracer, l.Lg), nil
		})
}

type EtcdLockManager struct {
	client *clientv3.Client
	prefix string

	tracer trace.Tracer

	lg *slog.Logger
}

func NewEtcdLockManager(
	client *clientv3.Client,
	prefix string,
	tracer trace.Tracer,
	lg *slog.Logger,
) *EtcdLockManager {
	lm := &EtcdLockManager{
		client: client,
		prefix: prefix,
		tracer: tracer,
		lg:     lg,
	}
	return lm
}

func (e *EtcdLockManager) Health(ctx context.Context) (conditions []string, err error) {
	conditions = make([]string, 0)
	remoteEndpoints := e.client.Endpoints()
	for _, endp := range remoteEndpoints {
		resp, stErr := e.client.Status(ctx, endp)
		if stErr != nil {
			err = errors.Join(err, stErr)
			continue
		}
		if len(resp.Errors) > 0 {
			conditions = append(conditions, fmt.Sprintf("%s : %s", endp, strings.Join(resp.Errors, ",")))
		}
	}
	return
}

// !! Cannot reuse *concurrency.Session across multiple locks since it will break liveliness guarantee A
// locks will share their sessions and therefore keepalives will be sent for all locks, not just a specific lock.
// In the current implementation sessions are forcibly orphaned when the non-blocking call to unlock is
// made so we cannot re-use sessions in that case either -- since the session  will be orphaned for all locks
// if the session is re-used.
func (e *EtcdLockManager) NewLock(key string, opts ...lock.LockOption) lock.Lock {
	options := lock.DefaultLockOptions()
	options.Apply(opts...)
	return NewEtcdLock(
		e.lg,
		e.client,
		e.prefix,
		key,
		options,
	)
}
