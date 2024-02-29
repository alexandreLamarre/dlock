package cassandra_test

import (
	"testing"

	"github.com/alexandreLamarre/dlock/pkg/lock"
	"github.com/alexandreLamarre/dlock/pkg/lock/backend/cassandra"
	"github.com/alexandreLamarre/dlock/pkg/logger"
	"github.com/alexandreLamarre/dlock/pkg/test"
	"github.com/alexandreLamarre/dlock/pkg/test/conformance/integration"
	"github.com/alexandreLamarre/dlock/pkg/util/future"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
)

func TestCassandra(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cassandra Suite")
}

var lmF = future.New[lock.LockManager]()
var lmSet = future.New[lo.Tuple3[
	lock.LockManager, lock.LockManager, lock.LockManager,
]]()

const (
	cassandraHost = "127.0.0.1:3000"
)

var _ = BeforeSuite(func() {
	if Label("integration").MatchesLabelFilter(GinkgoLabelFilter()) {
		env := test.Environment{}
		Expect(env.Start()).To(Succeed())

		session, err := cassandra.NewCassandraSession(cassandraHost)
		Expect(err).To(Succeed())

		lm := cassandra.NewLockManager(
			session, "test", nil, logger.NewNop(),
		)
		lmF.Set(lm)

		x, err := cassandra.NewCassandraSession(cassandraHost)
		Expect(err).To(Succeed())
		lmX := cassandra.NewLockManager(x, "test", nil, logger.NewNop())

		y, err := cassandra.NewCassandraSession(cassandraHost)
		Expect(err).To(Succeed())
		lmY := cassandra.NewLockManager(y, "test", nil, logger.NewNop())

		z, err := cassandra.NewCassandraSession(cassandraHost)
		Expect(err).To(Succeed())
		lmZ := cassandra.NewLockManager(z, "test", nil, logger.NewNop())

		lmSet.Set(lo.Tuple3[lock.LockManager, lock.LockManager, lock.LockManager]{
			A: lmX, B: lmY, C: lmZ,
		})

		DeferCleanup(env.Stop, "Test Suite Finished")

		Expect(err).NotTo(HaveOccurred())
		Expect(err).To(Succeed())
		DeferCleanup(env.Stop, "Test Suite Finished")
	}
})

var _ = Describe("Cassandra Lock Manager", Ordered, Label("integration", "slow"), integration.LockManagerTestSuite(lmF, lmSet))
