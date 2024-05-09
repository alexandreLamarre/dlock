package jetstream_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/alexandreLamarre/dlock/pkg/config/v1alpha1"
	"github.com/alexandreLamarre/dlock/pkg/lock"
	"github.com/alexandreLamarre/dlock/pkg/lock/backend/jetstream"
	"github.com/alexandreLamarre/dlock/pkg/logger"
	"github.com/alexandreLamarre/dlock/pkg/test/conformance/integration"
	"github.com/alexandreLamarre/dlock/pkg/test/container"
	"github.com/alexandreLamarre/dlock/pkg/util/future"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
)

func TestJetStream(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "JetStream Storage Suite")
}

var lmF = future.New[lock.LockManager]()
var lmSetF = future.New[lo.Tuple3[
	lock.LockManager, lock.LockManager, lock.LockManager,
]]()

var _ = BeforeSuite(func() {
	if Label("integration").MatchesLabelFilter(GinkgoLabelFilter()) {
		ctxca, ca := context.WithCancel(context.Background())
		DeferCleanup(func() {
			ca()
		})

		natsC, err := container.StartNatsContainer(ctxca)
		Expect(err).To(Succeed())
		DeferCleanup(func() {
			natsC.Container.Terminate(ctxca)
		})

		natsUrl, err := url.Parse(natsC.URI)
		Expect(err).To(Succeed())
		conf := &v1alpha1.JetstreamClientSpec{
			Endpoint: natsUrl.Host,
		}

		js, err := jetstream.AcquireJetstreamConn(
			context.Background(),
			conf,
			logger.NewNop(),
		)
		Expect(err).NotTo(HaveOccurred())

		lm := jetstream.NewLockManager(
			context.Background(),
			js,
			"test",
			nil,
			logger.NewNop(),
		)
		lmF.Set(lm)

		js1, err := jetstream.AcquireJetstreamConn(
			context.Background(),
			conf,
			logger.NewNop(),
		)
		Expect(err).NotTo(HaveOccurred())

		js2, err := jetstream.AcquireJetstreamConn(
			context.Background(),
			conf,
			logger.NewNop(),
		)
		Expect(err).NotTo(HaveOccurred())

		js3, err := jetstream.AcquireJetstreamConn(
			context.Background(),
			conf,
			logger.NewNop(),
		)
		Expect(err).NotTo(HaveOccurred())

		x := jetstream.NewLockManager(context.Background(), js1, "test", nil, logger.NewNop())
		y := jetstream.NewLockManager(context.Background(), js2, "test", nil, logger.NewNop())
		z := jetstream.NewLockManager(context.Background(), js3, "test", nil, logger.NewNop())

		lmSetF.Set(lo.Tuple3[lock.LockManager, lock.LockManager, lock.LockManager]{
			A: x, B: y, C: z,
		})
	}
})

var _ = Describe("Jetstream Lock Manager", Ordered, Label("integration", "slow"), integration.LockManagerTestSuite(lmF, lmSetF))
