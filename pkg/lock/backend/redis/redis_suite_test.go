package redis_test

import (
	"context"
	"testing"

	"github.com/alexandreLamarre/dlock/pkg/lock"
	"github.com/alexandreLamarre/dlock/pkg/lock/backend/redis"
	"github.com/alexandreLamarre/dlock/pkg/logger"
	"github.com/alexandreLamarre/dlock/pkg/test"
	"github.com/alexandreLamarre/dlock/pkg/test/conformance/integration"
	"github.com/alexandreLamarre/dlock/pkg/util/future"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
)

func TestRedis(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Redis Suite")
}

var lmF = future.New[lock.LockManager]()
var lmSetF = future.New[lo.Tuple3[
	lock.LockManager, lock.LockManager, lock.LockManager,
]]()

var _ = BeforeSuite(func() {
	if Label("integration").MatchesLabelFilter(GinkgoLabelFilter()) {
		env := test.Environment{}
		Expect(env.Start()).To(Succeed())

		conf, err := env.StartRedis()
		GinkgoWriter.Write([]byte("started redis...."))
		Expect(err).NotTo(HaveOccurred())
		pools := redis.AcquireRedisPool(conf)

		lm := redis.NewLockManager(
			context.Background(),
			"test",
			pools,
			logger.NewNop(),
		)
		lmF.Set(lm)

		pool1 := redis.AcquireRedisPool(conf)
		pool2 := redis.AcquireRedisPool(conf)
		pool3 := redis.AcquireRedisPool(conf)

		x := redis.NewLockManager(context.Background(), "test", pool1, logger.NewNop())
		y := redis.NewLockManager(context.Background(), "test", pool2, logger.NewNop())
		z := redis.NewLockManager(context.Background(), "test", pool3, logger.NewNop().WithGroup("redis-lock-pool3"))

		lmSetF.Set(lo.Tuple3[lock.LockManager, lock.LockManager, lock.LockManager]{
			A: x, B: y, C: z,
		})

		DeferCleanup(env.Stop, "Test Suite Finished")
	}
})

var _ = Describe("Redis Lock Manager", Ordered, Label("integration", "slow"), integration.LockManagerTestSuite(lmF, lmSetF))
