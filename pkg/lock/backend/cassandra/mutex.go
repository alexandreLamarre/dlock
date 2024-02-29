package cassandra

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/alexandreLamarre/dlock/pkg/lock"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	leasesTable   = "leases"
	cqlInsertLock = `INSERT INTO ` + leasesTable + ` (name, owner) VALUES (?,?) IF NOT EXISTS USING TTL ?;`
	cqlUpdateLock = `UPDATE ` + leasesTable + ` USING TTL ? SET owner = ? WHERE name = ? IF owner = ?;`
	cqlDeleteLock = `DELETE FROM ` + leasesTable + ` WHERE name = ? IF owner = ?;`
)

var (
	errLockOwnership = errors.New("this host does not own the resource lock")
	LockValidity     = 60 * time.Second
	LockRetryDelay   = 100 * time.Millisecond
)

type mutex struct {
	lg     *slog.Logger
	prefix string
	key    string

	muuid   string
	session *gocql.Session

	internalDone chan struct{}
	*lock.LockOptions
}

func newMutex(
	lg *slog.Logger,
	prefix, key string,
	session *gocql.Session,
	opts *lock.LockOptions,
) mutex {
	return mutex{
		lg:           lg,
		session:      session,
		prefix:       prefix,
		key:          key,
		internalDone: make(chan struct{}, 1),
		LockOptions:  opts,
		muuid:        uuid.New().String(),
	}
}

func (m *mutex) Key() string {
	return fmt.Sprintf("%s.%s", m.prefix, m.key)
}

func (m *mutex) tryLock(_ context.Context) (<-chan struct{}, error) {
	ttlSec := LockValidity.Seconds()
	var name, owner string
	applied, err := m.session.Query(cqlInsertLock, m.Key(), m.muuid, ttlSec).ScanCAS(&name, &owner)
	if err != nil {
		return nil, err
	}
	if applied {
		return lo.Async(m.keepaliveC), nil
	}
	return nil, nil
}

func (m *mutex) tryUnlock() error {
	var name, owner string

	applied, err := m.session.Query(cqlDeleteLock, m.Key(), m.muuid).ScanCAS(name, owner)
	if err != nil {
		return err
	}
	if applied {
		return nil
	}
	return fmt.Errorf("failed to delete remote lock : %w", errLockOwnership)
}

func (m *mutex) unlock() error {
	defer m.teardown()
	ctx := context.Background()
	var span trace.Span
	if m.Tracer != nil {
		ctx, span = m.Tracer.Start(context.Background(), "Unlock/cassandra", trace.WithAttributes(
			attribute.KeyValue{
				Key:   "key",
				Value: attribute.StringValue(m.Key()),
			}),
		)
		defer span.End()
	}
	ctx, ca := context.WithTimeout(ctx, 60*time.Second)
	defer ca()
	tTicker := time.NewTicker(LockRetryDelay)
	defer tTicker.Stop()

	// always try at least one unlock operation immediately
	if err := m.tryUnlock(); err == nil {
		return nil
	}

	for {
		select {
		case <-tTicker.C:
			err := m.tryUnlock()
			if err == nil {
				return nil
			}
			m.lg.Warn(fmt.Sprintf("failed to unlock : %s, retrying", err.Error()))
			m.traceError(span, err)
		case <-ctx.Done():
			err := ctx.Err()
			m.traceError(span, err)
			return err
		}
	}
}

func (m *mutex) traceError(span trace.Span, err error) {
	if span != nil {
		span.RecordError(err)
	}
}

func (m *mutex) extendLease() error {
	ttlSec := LockValidity.Seconds()
	var owner string
	applied, err := m.session.Query(cqlUpdateLock, ttlSec, m.muuid, m.Key(), m.muuid).ScanCAS(&owner)
	if err != nil {
		return err
	}
	if applied {
		return nil
	}
	return errLockOwnership
}

func (m *mutex) teardown() {
	defer close(m.internalDone)
	select {
	case m.internalDone <- struct{}{}:
	default:
	}
}

func (m *mutex) keepaliveC() struct{} {
	for {
		select {
		case <-m.internalDone:
			return struct{}{}
		default:
			// TODO : potentially spawn this as a background task, but :
			// need to be careful to never run it after <-internalDone receives a value
			if err := m.extendLease(); err != nil {
				return struct{}{}
			}
		}
	}
}
