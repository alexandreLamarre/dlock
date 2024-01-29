package server

import (
	"sync"

	api "go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

var (
	meterProviderMu sync.Mutex
	meterProvider   *sdkmetric.MeterProvider

	LockAcquisitionCount api.Float64Counter
	LockRequestCount     api.Float64Counter
	LockHeldTime         api.Float64Histogram

	// TODO : unused
	LockAcquisitionLatency api.Float64Histogram
	LockRequestLatency     api.Float64Histogram
	UnlockLatency          api.Float64Histogram
	UnlockRequestCount     api.Float64Counter
	UnlockSuccessCount     api.Float64Counter
)

func RegisterMeterProvider(mp *sdkmetric.MeterProvider) {
	meterProviderMu.Lock()
	defer meterProviderMu.Unlock()
	meterProvider = mp
	createMetrics()
}

func createMetrics() {
	meter := meterProvider.Meter("dlock_server")
	lockAcquisitionCount, err := meter.Float64Counter("lock_acquisition_count")
	if err != nil {
		panic(err)
	}
	lockAcquisitionLatency, err := meter.Float64Histogram("lock_acquisition_latency", api.WithUnit("ns"))
	if err != nil {
		panic(err)
	}
	lockRequestCount, err := meter.Float64Counter("lock_total_request_count")
	if err != nil {
		panic(err)
	}
	lockRequestLatency, err := meter.Float64Histogram("lock_total_request_latency", api.WithUnit("ns"))
	if err != nil {
		panic(err)
	}
	unlockLatency, err := meter.Float64Histogram("unlock_latency", api.WithUnit("ns"))
	if err != nil {
		panic(err)
	}
	unlockRequestCount, err := meter.Float64Counter("unlock_total_request_count")
	if err != nil {
		panic(err)
	}
	unlockSuccessCount, err := meter.Float64Counter("unlock_success_count")
	if err != nil {
		panic(err)
	}
	// TODO : tweak buckets
	lockHeldTime, err := meter.Float64Histogram("lock_held_time", api.WithUnit("ms"))
	if err != nil {
		panic(err)
	}

	LockAcquisitionCount = lockAcquisitionCount
	LockAcquisitionLatency = lockAcquisitionLatency
	LockRequestCount = lockRequestCount
	LockRequestLatency = lockRequestLatency
	UnlockLatency = unlockLatency
	UnlockRequestCount = unlockRequestCount
	UnlockSuccessCount = unlockSuccessCount
	LockHeldTime = lockHeldTime
}

func init() {
	meterProviderMu.Lock()
	defer meterProviderMu.Unlock()
	if meterProvider == nil {
		meterProvider = sdkmetric.NewMeterProvider()
	}
	createMetrics()
}
