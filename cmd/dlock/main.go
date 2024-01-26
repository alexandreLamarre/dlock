package main

import (
	"log/slog"
	"os"

	"github.com/alexandreLamarre/dlock/pkg/constants"
	"github.com/alexandreLamarre/dlock/pkg/logger"
	"github.com/alexandreLamarre/dlock/pkg/server"
	"github.com/spf13/cobra"
)

func main() {
	BuildRootCmd().Execute()
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

func BuildRootCmd() *cobra.Command {
	var configPath string
	var addr string
	var logLevel string
	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := os.Stat(configPath); err != nil {
				return err
			}
			lockServer := server.NewLockServer(cmd.Context(), logger.New(
				logger.WithLogLevel(logLevelFromString(logLevel)),
			), configPath)
			return lockServer.ListenAndServe(cmd.Context(), addr)
		},
	}
	cmd.Flags().StringVarP(&configPath, "config", "c", "/var/opt/dlock/config.json", "path to config file")
	cmd.Flags().StringVarP(&addr, "addr", "a", constants.DefaultDlockGrpcAddr, "address to listen on")
	cmd.Flags().StringVarP(&logLevel, "log-level", "l", "info", "log level")
	return cmd
}
