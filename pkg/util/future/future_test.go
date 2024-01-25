package future_test

import (
	"context"
	"runtime/debug"
	"time"

	"github.com/alexandreLamarre/dlock/pkg/util/future"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Future", Label("unit"), func() {
	BeforeEach(func() {
		// temporarily pause garbage collection to avoid interfering with timing
		gcPercent := debug.SetGCPercent(-1)
		DeferCleanup(func() {
			debug.SetGCPercent(gcPercent)
		})
	})

	Specify("Get should block until Set is called", func() {
		f := future.New[string]()
		go func() {
			time.Sleep(time.Millisecond * 100)
			f.Set("test")
		}()
		start := time.Now()
		Expect(f.Get()).To(Equal("test"))
		Expect(time.Since(start)).To(BeNumerically(">=", time.Millisecond*100))
	})
	Specify("GetContext should allow Get to be canceled", func() {
		f := future.New[string]()
		ctx, ca := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer ca()
		go func() {
			time.Sleep(time.Millisecond * 100)
			f.Set("test")
		}()
		start := time.Now()
		_, err := f.GetContext(ctx)
		Expect(err).To(MatchError(context.DeadlineExceeded))
		Expect(time.Since(start)).To(BeNumerically("~", time.Millisecond*50, time.Millisecond*10))

		f2 := future.New[string]()
		ctx, ca = context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer ca()
		go func() {
			time.Sleep(time.Millisecond * 25)
			f2.Set("test")
		}()
		start = time.Now()
		Expect(f2.GetContext(ctx)).To(Equal("test"))
		Expect(time.Since(start)).To(BeNumerically("~", time.Millisecond*25, time.Millisecond*10))
	})
})
