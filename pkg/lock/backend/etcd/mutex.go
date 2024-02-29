package etcd

import (
	"context"
	"errors"
	"log/slog"
	"path"
	"time"

	"github.com/alexandreLamarre/dlock/pkg/lock"
	"github.com/samber/lo"
	"go.etcd.io/etcd/client/v3/concurrency"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// encapsulates stateful information and tasks requried for holding a lock
type etcdMutex struct {
	lg *slog.Logger

	prefix string
	key    string

	session *concurrency.Session

	mutex *concurrency.Mutex

	internalDone chan struct{}
	*lock.LockOptions
}

func NewEtcdMutex(
	lg *slog.Logger,
	prefix, key string,
	session *concurrency.Session,
	opts *lock.LockOptions,
) etcdMutex {
	return etcdMutex{
		lg:      lg,
		session: session,

		key:    key,
		prefix: prefix,
		// mu:           sync.Mutex{},
		internalDone: make(chan struct{}),
		LockOptions:  opts,
	}
}

func (e *etcdMutex) lock(ctx context.Context) (<-chan struct{}, error) {
	mutex := concurrency.NewMutex(
		e.session,
		path.Join(e.prefix, e.key),
	)
	if err := mutex.Lock(ctx); err != nil {
		return nil, err
	}
	e.mutex = mutex
	return lo.Async(e.keepalive), nil
}

func (e *etcdMutex) tryLock(ctx context.Context) (<-chan struct{}, error) {
	mutex := concurrency.NewMutex(
		e.session,
		path.Join(e.prefix, e.key),
	)
	if err := mutex.TryLock(ctx); err != nil {
		return nil, err
	}
	e.mutex = mutex

	return lo.Async(e.keepalive), nil
}

func (e *etcdMutex) keepalive() struct{} {
	select {
	case <-e.internalDone:
		return struct{}{}
	case <-e.session.Done():
		e.lg.Warn("releasing lock early, etcd session is done")
		return struct{}{}
	}
}

func (e *etcdMutex) teardown() {
	defer close(e.internalDone)
	select {
	case e.internalDone <- struct{}{}:
	default:
	}
	// sessions must be forcibly orphaned in order to make the guarantee that non-blocking calls
	// to unlock always unlock
	e.session.Orphan()
}

// best effort unlock until context is done, at which point we
// basically disconnect the connection keepalive semantic by orphany the mutex's session
// which delegates unlock the key to the KV server-side,
// giving the guarantee that unlock always actually unlocks when called
func (e *etcdMutex) unlock() error {
	var span trace.Span
	ctx := context.Background()
	if e.TracingEnabled() {
		ctxSpan, span := e.Tracer.Start(ctx, "Lock/etcd-unlock", trace.WithAttributes(
			attribute.KeyValue{
				Key:   "key",
				Value: attribute.StringValue(e.key),
			},
		))
		defer span.End()
		ctx = ctxSpan
	}
	if e.mutex == nil {
		err := errors.New("mutex not acquired")
		e.RecordError(span, err)
		return err
	}
	defer e.teardown()

	mutex := *e.mutex
	e.mutex = nil
	go func() {
		ctxca, ca := context.WithTimeout(ctx, 60*time.Second)
		defer ca()
		if err := mutex.Unlock(ctxca); err != nil {
			e.lg.Warn("failed to unlock mutex", "err", err.Error())
			e.RecordError(span, err)
		}
	}()
	return nil
}
