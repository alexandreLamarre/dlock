package cassandra

import (
	"log/slog"

	"github.com/alexandreLamarre/dlock/pkg/lock"
	"github.com/gocql/gocql"
	"go.opentelemetry.io/otel/trace"
)

type LockManager struct {
	session *gocql.Session
	prefix  string

	tracer trace.Tracer

	lg *slog.Logger
}

var _ lock.LockManager = (*LockManager)(nil)

func NewLockManager(
	session *gocql.Session,
	prefix string,
	tracer trace.Tracer,
	lg *slog.Logger,
) *LockManager {
	return &LockManager{
		session: session,
		prefix:  prefix,
		tracer:  tracer,
		lg:      lg,
	}
}

func (l *LockManager) NewLock(key string, opts ...lock.LockOption) lock.Lock {
	options := lock.DefaultLockOptions()
	options.Apply(opts...)
	return NewCassandraLock(l.lg, l.session, l.prefix, key, options)
}
