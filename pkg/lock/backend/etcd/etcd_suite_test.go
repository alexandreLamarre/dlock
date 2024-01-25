package etcd_test

import (
	"context"
	"testing"

	"github.com/alexandreLamarre/dlock/pkg/lock"
	"github.com/alexandreLamarre/dlock/pkg/lock/backend/etcd"
	"github.com/alexandreLamarre/dlock/pkg/logger"
	"github.com/alexandreLamarre/dlock/pkg/test"
	"github.com/alexandreLamarre/dlock/pkg/test/conformance/integration"
	"github.com/alexandreLamarre/dlock/pkg/util/future"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
)

func TestEtcd(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Etcd Storage Suite")
}

var lmF = future.New[lock.LockManager]()
var lmSet = future.New[lo.Tuple3[
	lock.LockManager, lock.LockManager, lock.LockManager,
]]()

var _ = BeforeSuite(func() {
	if Label("integration").MatchesLabelFilter(GinkgoLabelFilter()) {
		env := test.Environment{}
		Expect(env.Start()).To(Succeed())

		conf, err := env.StartEtcd()
		Expect(err).To(Succeed())

		cli, err := etcd.NewEtcdClient(context.Background(), conf)
		Expect(err).To(Succeed())

		lm := etcd.NewEtcdLockManager(cli, "test", logger.NewNop())
		lmF.Set(lm)

		x, err := etcd.NewEtcdClient(context.Background(), conf)
		Expect(err).To(Succeed())
		lmX := etcd.NewEtcdLockManager(x, "test", logger.NewNop())

		y, err := etcd.NewEtcdClient(context.Background(), conf)
		Expect(err).To(Succeed())
		lmY := etcd.NewEtcdLockManager(y, "test", logger.NewNop())
		Expect(err).To(Succeed())

		z, err := etcd.NewEtcdClient(context.Background(), conf)
		Expect(err).To(Succeed())
		lmZ := etcd.NewEtcdLockManager(z, "test", logger.NewNop())

		lmSet.Set(lo.Tuple3[lock.LockManager, lock.LockManager, lock.LockManager]{
			A: lmX, B: lmY, C: lmZ,
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(err).To(Succeed())
		DeferCleanup(env.Stop, "Test Suite Finished")
	}
})

var _ = Describe("Etcd Lock Manager", Ordered, Label("integration", "slow"), integration.LockManagerTestSuite(lmF, lmSet))
