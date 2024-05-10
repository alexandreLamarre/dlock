package broker

import (
	"context"
	"sync"

	"github.com/alexandreLamarre/dlock/pkg/lock"
	"github.com/samber/lo"
)

type lockBroker = func(context.Context, LockBroker) (lock.LockManager, error)

var (
	brokerMu    sync.RWMutex
	brokerCache map[string]lockBroker
)

func init() {
	brokerMu.Lock()
	defer brokerMu.Unlock()
	brokerCache = map[string]lockBroker{}
}

func RegisterLockBroker(name string, broker lockBroker) {
	brokerMu.Lock()
	defer brokerMu.Unlock()
	brokerCache[name] = broker
}

func GetLockBroker(name string) (broker lockBroker, ok bool) {
	brokerMu.RLock()
	defer brokerMu.RUnlock()
	broker, ok = brokerCache[name]
	return
}

func brokerKeys() []string {
	brokerMu.RLock()
	defer brokerMu.RUnlock()
	return lo.Keys(brokerCache)
}
