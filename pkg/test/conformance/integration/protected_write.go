package integration

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/alexandreLamarre/dlock/pkg/lock"
	"github.com/alexandreLamarre/dlock/pkg/util/future"
	"github.com/samber/lo"
)

func ProtectedWriteTestSuite(
	lmF future.Future[lock.LockManager],
	lmSetF future.Future[lo.Tuple3[
		lock.LockManager,
		lock.LockManager,
		lock.LockManager,
	]],
) func() {
	return func() {
		var lm lock.LockManager
		var lmSet lo.Tuple3[lock.LockManager, lock.LockManager, lock.LockManager]
		var ctx context.Context

		BeforeAll(func() {
			ctxca, ca := context.WithCancel(context.Background())
			DeferCleanup(func() {
				ca()
			})
			ctx = ctxca
			lm = lmF.Get()
			lmSet = lmSetF.Get()
		})

		When("using protected write distributed locks within the same client conn", func() {
			It("should lock and unlock locks of the same type", func() {
				lock1 := lm.PWLock("todo")
				done1, err := lock1.Lock(ctx)
				Expect(err).To(Succeed())
				Expect(lock1.Unlock()).To(Succeed())
				Eventually(done1).Should(Receive())
				lock2 := lm.PWLock("todo")
				done2, err := lock2.Lock(ctx)
				Expect(err).To(Succeed())
				Expect(lock2.Unlock()).To(Succeed())
				Eventually(done2).Should(Receive())
			})
		})

		When("using protected write distributed locks across different client conn", func() {
			It("should be able to lock and unlock locks", func() {
				x := lmSet.A.PWLock("todo")
				y := lmSet.B.PWLock("todo")
				z := lmSet.C.PWLock("todo")

				doneX, err := x.Lock(ctx)
				Expect(err).To(Succeed())
				Expect(x.Unlock()).To(Succeed())
				Eventually(doneX).Should(Receive())

				doneY, err := y.Lock(ctx)
				Expect(err).To(Succeed())
				Expect(y.Unlock()).To(Succeed())
				Eventually(doneY).Should(Receive())

				doneZ, err := z.Lock(ctx)
				Expect(err).To(Succeed())
				Expect(z.Unlock()).To(Succeed())
				Eventually(doneZ).Should(Receive())
			})
		})
	}
}
