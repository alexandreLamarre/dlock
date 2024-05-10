package redis_test

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/alexandreLamarre/dlock/pkg/constants"
	"github.com/alexandreLamarre/dlock/pkg/lock"
	"github.com/alexandreLamarre/dlock/pkg/lock/backend/redis"
	"github.com/alexandreLamarre/dlock/pkg/lock/broker"
	"github.com/alexandreLamarre/dlock/pkg/logger"
	"github.com/alexandreLamarre/dlock/pkg/test/conformance/integration"
	"github.com/alexandreLamarre/dlock/pkg/test/container"
	"github.com/alexandreLamarre/dlock/pkg/util/future"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	goredislib "github.com/redis/go-redis/v9"
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
		ctx := context.Background()
		ctxca, ca := context.WithCancel(ctx)
		DeferCleanup(
			func() {
				ca()
			},
		)
		By("verifying the redis container starts")
		redisC, err := container.StartRedisContainer(ctxca)
		Expect(err).NotTo(HaveOccurred())

		DeferCleanup(func() {
			redisC.Container.Terminate(ctx)
		})
		redisUrl, err := url.Parse(redisC.URI)
		Expect(err).NotTo(HaveOccurred())
		conf := []*goredislib.Options{
			{
				Network: "tcp",
				Addr:    redisUrl.Host,
			},
		}

		GinkgoWriter.Write([]byte("started redis...."))
		Expect(err).NotTo(HaveOccurred())
		pools := redis.AcquireRedisPool(conf)

		By("Verifying the redis pools are reachable from the clients")
		for _, pool := range pools {
			ctxca, ca := context.WithTimeout(ctx, 1*time.Second)
			defer ca()
			conn, err := pool.Get(ctxca)
			Expect(err).NotTo(HaveOccurred())
			defer conn.Close()
			_, err = conn.Get("test")
			Expect(err).NotTo(HaveOccurred())
		}

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

	}
})

var _ = Describe("Redis Lock Manager", Ordered, Label("integration", "slow"), integration.LockManagerTestSuite(lmF, lmSetF))
var _ = Describe("Redis Broker", Label("unit"), func() {
	When("we register the lock broker", func() {
		It("should register the redis lock manager as a broker", func() {
			rBroker, ok := broker.GetLockBroker(constants.RedisLockManager)
			Expect(ok).To(BeTrue())
			Expect(rBroker).NotTo(BeNil())
		})
	})
})
