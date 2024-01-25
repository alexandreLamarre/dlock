package lock

import (
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
