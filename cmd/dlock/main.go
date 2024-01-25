package main

import (
	"github.com/alexandreLamarre/dlock/pkg/logger"
	"github.com/alexandreLamarre/dlock/pkg/server"
	"github.com/spf13/cobra"
)

func main() {
	BuildRootCmd().Execute()
}

func BuildRootCmd() *cobra.Command {
	var configPath string
	var addr string
	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO : read config file path
			lockServer := server.NewLockServer(logger.New())
			return lockServer.ListenAndServe(cmd.Context(), addr)
		},
	}
	cmd.Flags().StringVarP(&configPath, "config", "c", "/var/opt/dlock/config.json", "path to config file")
	cmd.Flags().StringVarP(&addr, "addr", "a", "tcp4://127.0.0.1:3001", "address to listen on")
	return cmd
}
