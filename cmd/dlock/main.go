package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/alexandreLamarre/dlock/pkg/constants"
	"github.com/alexandreLamarre/dlock/pkg/instrumentation"
	"github.com/alexandreLamarre/dlock/pkg/logger"
	"github.com/alexandreLamarre/dlock/pkg/server"
	"github.com/alexandreLamarre/dlock/pkg/util"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

func main() {
	ctx := context.Background()

	exp, err := instrumentation.NewTraceExporter(ctx)
	if err != nil {
		log.Fatalf("failed to initialize exporter: %v", err)
	}

	// Create a new tracer provider with a batch span processor and the given exporter.
	tp := instrumentation.NewTracerProvider(exp)

	// Handle shutdown properly so nothing leaks.
	defer func() { _ = tp.Shutdown(ctx) }()

	otel.SetTracerProvider(tp)

	// Finally, set the tracer that can be used for this package.
	tracer := tp.Tracer("ExampleService")
	BuildRootCmd(tracer).Execute()
}

func logLevelFromString(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func BuildRootCmd(tracer trace.Tracer) *cobra.Command {
	var configPath string
	var addr string
	var metricsAddr string
	var logLevel string
	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, ca := context.WithCancelCause(cmd.Context())
			if _, err := os.Stat(configPath); err != nil {
				return err
			}
			metricsServer := instrumentation.NewMetricsServer(metricsAddr)
			servermetrics := server.NewLockServerMetrics(metricsServer.Provider())
			lockServer := server.NewLockServer(
				cmd.Context(),
				tracer,
				logger.New(
					logger.WithLogLevel(logLevelFromString(logLevel)),
				),
				configPath,
				servermetrics,
			)
			e1 := lo.Async(func() error {
				return lockServer.ListenAndServe(cmd.Context(), addr)
			})

			e2 := lo.Async(func() error {
				return metricsServer.ListenAndServe()
			})

			return util.WaitAll(ctx, ca, e1, e2)
		},
	}
	cmd.Flags().StringVarP(&configPath, "config", "c", "/var/opt/dlock/config.json", "path to config file")
	cmd.Flags().StringVarP(&addr, "addr", "a", constants.DefaultDlockGrpcAddr, "address to listen on")
	cmd.Flags().StringVarP(&logLevel, "log-level", "l", "info", "log level")
	cmd.Flags().StringVarP(&metricsAddr, "metrics-addr", "m", "127.0.0.1:8088", "address to listen on for metrics")
	return cmd
}
