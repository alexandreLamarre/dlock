package etcd_test

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/alexandreLamarre/dlock/internal/lock/backend/etcd"
	"github.com/alexandreLamarre/dlock/pkg/config/v1alpha1"
	"github.com/alexandreLamarre/dlock/pkg/constants"
	"github.com/alexandreLamarre/dlock/pkg/lock"
	"github.com/alexandreLamarre/dlock/pkg/lock/broker"
	"github.com/alexandreLamarre/dlock/pkg/logger"
	"github.com/alexandreLamarre/dlock/pkg/test/conformance/integration"
	"github.com/alexandreLamarre/dlock/pkg/test/container"
	"github.com/alexandreLamarre/dlock/pkg/util/future"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	clientv3 "go.etcd.io/etcd/client/v3"
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
		ctx := context.Background()
		ctxca, ca := context.WithCancel(ctx)
		DeferCleanup(
			func() {
				ca()
			},
		)
		etcdC, err := container.StartEtcdContainer(ctxca)
		Expect(err).NotTo(HaveOccurred())

		DeferCleanup(func() {
			etcdC.Container.Terminate(ctx)
		})
		etcdUrl, err := url.Parse(etcdC.URI)
		Expect(err).NotTo(HaveOccurred())
		conf := &v1alpha1.EtcdClientSpec{
			Endpoints: []string{
				etcdUrl.String(),
			},
		}

		cli, err := etcd.NewEtcdClient(context.Background(), conf)
		Expect(err).To(Succeed())
		ctxT, caT := context.WithTimeout(ctxca, 1*time.Second)
		DeferCleanup(func() {
			caT()
		})
		kapi := clientv3.NewKV(cli)
		resp, err := kapi.Put(ctxT, "/foo", "bar")
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.Header.Revision).To(BeNumerically(">", 0))
		// Set the key foo
		// resp, err := kapi.Set(ctxT), "/foo", "bar")
		// Expect(err).NotTo(HaveOccurred())

		lm := etcd.NewEtcdLockManager(cli, "test", nil, logger.NewNop())
		lmF.Set(lm)

		x, err := etcd.NewEtcdClient(context.Background(), conf)
		Expect(err).To(Succeed())
		lmX := etcd.NewEtcdLockManager(x, "test", nil, logger.NewNop())

		y, err := etcd.NewEtcdClient(context.Background(), conf)
		Expect(err).To(Succeed())
		lmY := etcd.NewEtcdLockManager(y, "test", nil, logger.NewNop())
		Expect(err).To(Succeed())

		z, err := etcd.NewEtcdClient(context.Background(), conf)
		Expect(err).To(Succeed())
		lmZ := etcd.NewEtcdLockManager(z, "test", nil, logger.NewNop())

		lmSet.Set(lo.Tuple3[lock.LockManager, lock.LockManager, lock.LockManager]{
			A: lmX, B: lmY, C: lmZ,
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(err).To(Succeed())
		// DeferCleanup(env.Stop, "Test Suite Finished")
	}
})

var _ = Describe("Etcd Lock Manager", Ordered, Label("integration", "slow"), integration.LockManagerTestSuite(lmF, lmSet))
var _ = Describe("Etcd Broker", Label("unit"), func() {
	When("we register the lock broker", func() {
		It("should register the Etcd lock manager as a broker", func() {
			eBroker, ok := broker.GetLockBroker(constants.EtcdLockManager)
			Expect(ok).To(BeTrue())
			Expect(eBroker).NotTo(BeNil())
		})
	})
})
