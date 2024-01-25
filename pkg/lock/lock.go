package lock

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrLockActionRequested = errors.New("lock action already requested")
	ErrLockScheduled       = errors.New("nothing scheduled")
)

var (
	DefaultRetryDelay     = 10 * time.Millisecond
	DefaultAcquireTimeout = 100 * time.Millisecond
	DefaultTimeout        = 10 * time.Second
)

// Lock is a distributed lock that can be used to coordinate access to a resource or interest in
// such a resource.
// Locks follow the following liveliness & atomicity guarantees to prevent distributed deadlocks
// and guarantee atomicity in the critical section.
//
// Liveliness A :  A lock is always eventually released when the process holding it crashes or exits unexpectedly.
// Liveliness B : A lock is always eventually released when its backend store is unavailable.
// Atomicity A : No two processes or threads can hold the same lock at the same time.
// Atomicity B : Any call to unlock will always eventually release the lock
type Lock interface {
	// Lock acquires a lock on the key. If the lock is already held, it will block until the lock is acquired or
	// the context fails.
	// Lock returns an error if the context expires or an unrecoverable error occurs when trying to acquire the lock.
	Lock(ctx context.Context) (expired chan struct{}, err error)
	// TryLock tries to acquire the lock on the key and reports whether it succeeded.
	// It blocks until at least one attempt was made to acquired the lock, and returns acquired=false and no error
	// if the lock is known to be held by someone else
	TryLock(ctx context.Context) (acquired bool, expired chan struct{}, err error)
	// Unlock releases the lock on the key in a non-blocking fashion.
	// It spawns a goroutine that will perform the unlock mechanism until it succeeds or the the lock is
	// expired by the server.
	// It immediately signals to the lock's original expired channel that the lock is released.
	Unlock() error
}

type LockManager interface {
	// Instantiates a new Lock instance for the given key, with the given options.
	//
	// Defaults to lock.DefaultOptions if no options are provided.
	NewLock(key string, opts ...LockOption) Lock
}

type LockScheduler struct {
	cond      sync.Cond
	scheduled bool
}

func NewLockScheduler() *LockScheduler {
	return &LockScheduler{
		cond: sync.Cond{
			L: &sync.Mutex{},
		},
		scheduled: false,
	}
}

func (l *LockScheduler) Schedule(f func() error) error {
	l.cond.L.Lock()
	defer l.cond.L.Unlock()

	for l.scheduled {
		l.cond.Wait()
	}

	if err := f(); err != nil {
		return err
	}

	l.scheduled = true
	return nil
}

func (l *LockScheduler) Done(f func() error) error {
	l.cond.L.Lock()
	defer l.cond.L.Unlock()

	if !l.scheduled {
		return ErrLockScheduled
	}

	if err := f(); err != nil {
		return err
	}

	l.scheduled = false
	l.cond.Signal()
	return nil
}

// Modified sync.Once primitive
type OnceErr struct {
	done uint32
	m    sync.Mutex
}

func (o *OnceErr) Do(f func() error) error {
	if atomic.LoadUint32(&o.done) == 0 {
		return o.doSlow(f)
	}
	return ErrLockActionRequested
}

func (o *OnceErr) doSlow(f func() error) error {
	o.m.Lock()
	defer o.m.Unlock()
	if o.done == 0 {
		defer atomic.StoreUint32(&o.done, 1)
		return f()
	}
	return nil
}

type LockOptions struct{}

func DefaultLockOptions() *LockOptions {
	return &LockOptions{}
}

func (o *LockOptions) Apply(opts ...LockOption) {
	for _, op := range opts {
		op(o)
	}
}

type LockOption func(o *LockOptions)
