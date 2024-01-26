package instrumentation

import (
	"log/slog"
	"net/http"

	"github.com/alexandreLamarre/dlock/pkg/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
)

func newMetricsExporter() (metric.Reader, error) {
	return prometheus.New()
}

type MetricsServer struct {
	lg *slog.Logger

	provider *metric.MeterProvider
	addr     string
}

func NewMetricsServer(addr string) *MetricsServer {
	promexporter, err := newMetricsExporter()
	if err != nil {
		panic(err)
	}
	provider := metric.NewMeterProvider(metric.WithReader(promexporter))
	metricsServer := &MetricsServer{
		lg:       logger.New().With("component", "metrics-server"),
		provider: provider,
		addr:     addr,
	}
	return metricsServer
}

func (s *MetricsServer) Provider() *metric.MeterProvider {
	return s.provider
}

func (s *MetricsServer) ListenAndServe() error {
	s.lg.With("addr", s.addr).Info("starting metrics server...")
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	httpServer := http.Server{
		Addr:    s.addr,
		Handler: mux,
	}
	return httpServer.ListenAndServe()
}
