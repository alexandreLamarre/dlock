//go:build etcd && redis && nats

package main_test

import (
	"github.com/alexandreLamarre/dlock/pkg/constants"
	"github.com/alexandreLamarre/dlock/pkg/lock/broker"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("", Label("unit"), func() {
	When("we register lock manager brokers", func() {
		It("should register the etcd broker", func() {
			eBroker, ok := broker.GetLockBroker(constants.EtcdLockManager)
			Expect(ok).To(BeTrue())
			Expect(eBroker).NotTo(BeNil())
		})

		It("should register the etcd broker", func() {
			jBroker, ok := broker.GetLockBroker(constants.JetstreamLockManager)
			Expect(ok).To(BeTrue())
			Expect(jBroker).NotTo(BeNil())
		})

		It("should register the redis broker", func() {
			rBroker, ok := broker.GetLockBroker(constants.RedisLockManager)
			Expect(ok).To(BeTrue())
			Expect(rBroker).NotTo(BeNil())
		})
	})
})
