package server

import (
	"github.com/alexandreLamarre/dlock/pkg/util"
	api "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
)

type LockServerMetrics struct {
	LockAcquisitionCount    api.Float64Counter
	LockAcquisitionLatency  api.Float64Histogram
	LockTotalRequestCount   api.Float64Counter
	LockTotalRequestLatency api.Float64Histogram
}

func NewLockServerMetrics(provider *metric.MeterProvider) *LockServerMetrics {
	meter := provider.Meter("distributedlock")

	return &LockServerMetrics{
		LockAcquisitionCount:    util.Must(meter.Float64Counter("lock_acquisition_count", api.WithDescription("The number of times a lock was successfully acquired."))),
		LockAcquisitionLatency:  util.Must(meter.Float64Histogram("lock_acquisition_latency_ms", api.WithDescription("The latency of a lock acquisition."))),
		LockTotalRequestCount:   util.Must(meter.Float64Counter("lock_total_request_count", api.WithDescription("The total number of lock requests."))),
		LockTotalRequestLatency: util.Must(meter.Float64Histogram("lock_total_request_latency_ms", api.WithDescription("The latency of a lock request."))),
	}
}
