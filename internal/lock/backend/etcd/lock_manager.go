package etcd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/alexandreLamarre/dlock/internal/lock/backend/unimplemented"
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

func (e *EtcdLockManager) EXLock(key string, opts ...lock.LockOption) lock.Lock {
	return e.NewLock(key, opts...)
}

func (e *EtcdLockManager) PWLock(key string, opts ...lock.LockOption) lock.Lock {
	return &unimplemented.UnimplementedLock{}
}

func (e *EtcdLockManager) PRLock(key string, opts ...lock.LockOption) lock.Lock {
	return &unimplemented.UnimplementedLock{}
}

func (e *EtcdLockManager) CWLock(key string, opts ...lock.LockOption) lock.Lock {
	return &unimplemented.UnimplementedLock{}
}

func (e *EtcdLockManager) CRLock(key string, opts ...lock.LockOption) lock.Lock {
	return &unimplemented.UnimplementedLock{}
}
