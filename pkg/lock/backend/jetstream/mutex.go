package jetstream

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/alexandreLamarre/dlock/pkg/lock"
	"github.com/alexandreLamarre/dlock/pkg/logger"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func newLease(key string) *nats.StreamConfig {
	return &nats.StreamConfig{
		Name:         key,
		Retention:    nats.InterestPolicy,
		Subjects:     []string{fmt.Sprintf("%s.lease.*", key)},
		MaxConsumers: 1,
	}
}

var (
	LockValidity   = 60 * time.Second
	LockRetryDelay = 100 * time.Millisecond
)

// encapsulates stateful information and tasks requried for holding a lock
type jetstreamMutex struct {
	lg *slog.Logger

	prefix string
	key    string
	uuid   string

	js   nats.JetStreamContext
	msgQ chan *nats.Msg

	sub          *nats.Subscription
	internalDone chan struct{}
	retDone      chan struct{}

	*lock.LockOptions
}

func newJetstreamMutex(
	lg *slog.Logger,
	js nats.JetStreamContext,
	prefix, key string,
	opts *lock.LockOptions,
) jetstreamMutex {
	uuid := uuid.New().String()
	return jetstreamMutex{
		js:           js,
		lg:           lg.With("uuid", uuid),
		prefix:       prefix,
		key:          key,
		uuid:         uuid,
		msgQ:         make(chan *nats.Msg, 16),
		internalDone: make(chan struct{}),
		retDone:      make(chan struct{}),
		LockOptions:  opts,
	}
}

func (j *jetstreamMutex) Key() string {
	return j.prefix + "-" + j.key
}

func (j *jetstreamMutex) tryLock() (<-chan struct{}, error) {
	var err error
	if _, err := j.js.AddStream(newLease(j.Key())); err != nil {
		return nil, err
	}
	cfg := &nats.ConsumerConfig{
		Durable:           j.uuid,
		AckPolicy:         nats.AckExplicitPolicy,
		InactiveThreshold: LockValidity,
		DeliverSubject:    j.uuid,
		Heartbeat:         max(LockRetryDelay, 100*time.Millisecond),
	}
	if _, err := j.js.AddConsumer(j.Key(), cfg); err != nil {
		j.lg.Warn(err.Error())
		return nil, err
	}
	j.sub, err = j.js.ChanSubscribe(j.uuid, j.msgQ, nats.Bind(j.Key(), j.uuid))
	if err != nil {
		j.lg.Warn(err.Error())
		return nil, err
	}
	return lo.Async(j.keepaliveC), nil
}

func (j *jetstreamMutex) keepaliveC() struct{} {
	for {
		select {
		case <-j.internalDone:
			return struct{}{}
		case msg, ok := <-j.msgQ:
			if !ok {
				return struct{}{}
			}
			if err := msg.Ack(); err != nil {
				j.lg.Warn(fmt.Sprintf("failed to ack : %s", err.Error()))
			}
		}
	}
}

func (j *jetstreamMutex) teardown() {
	defer close(j.internalDone)
	select {
	case j.internalDone <- struct{}{}:
	default:
	}
}

func (j *jetstreamMutex) isReleased(err error) bool {
	return err == nil || errors.Is(err, nats.ErrConsumerNotFound)
}

// !!Important : never treat nats closed connections as successful unlocks, this could lead to inconsistent states
func (j *jetstreamMutex) tryUnlock() error {
	unsubErr := j.sub.Unsubscribe()
	if unsubErr != nil {
		j.lg.With(logger.Err(unsubErr)).Warn("failed to unsubscribe to consumer")
	}
	drainErr := j.sub.Drain()
	if drainErr != nil {
		j.lg.With(logger.Err(drainErr)).Warn("failed to drain subscriber")
	}
	consumerErr := j.js.DeleteConsumer(j.Key(), j.uuid)
	if j.isReleased(consumerErr) {
		consumerErr = nil
	} else {
		j.lg.With(logger.Err(consumerErr)).Warn("failed to delete consumer")
	}
	return errors.Join(unsubErr, drainErr, consumerErr)
}

// best effort unlock until context is done, at which point we
// basically disconnect the connection keepalive semantic
// which delegates unlock the key to the KV server-side,
// giving the guarantee that unlock always actually unlocks when called
func (j *jetstreamMutex) unlock() error {
	defer j.teardown()
	ctx := context.Background()
	var span trace.Span
	if j.TracingEnabled() {
		ctx, span = j.Tracer.Start(context.Background(), "Unlock/jetstream-unlock", trace.WithAttributes(
			attribute.KeyValue{
				Key:   "key",
				Value: attribute.StringValue(j.Key()),
			},
		))
		defer span.End()
	}

	ctx, ca := context.WithTimeout(ctx, 60*time.Second)
	defer ca()
	tTicker := time.NewTicker(LockRetryDelay)
	defer tTicker.Stop()

	// always try at least one unlock operation before ctx is done
	if err := j.tryUnlock(); err == nil {
		return nil
	}

	for {
		select {
		case <-tTicker.C:
			err := j.tryUnlock()
			if err == nil {
				return nil
			}
			j.lg.Warn(fmt.Sprintf("failed to unlock : %s, retrying...", err.Error()))
			j.RecordError(span, err)
		case <-ctx.Done():
			err := ctx.Err()
			j.RecordError(span, err)
			return err
		}
	}
}
