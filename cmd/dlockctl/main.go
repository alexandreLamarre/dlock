package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/alexandreLamarre/dlock/api/v1alpha1"
	"github.com/alexandreLamarre/dlock/pkg/constants"
	"github.com/alexandreLamarre/dlock/pkg/logger"
	"github.com/alexandreLamarre/dlock/pkg/version"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

func main() {
	BuildRootCmd().Execute()
}

var (
	serverAddr string
	client     v1alpha1.DlockClient
	lg         *slog.Logger
)

func BuildRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Version: version.FriendlyVersion(),
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			var err error
			lg = logger.New()
			client, err = getDlockClient(serverAddr)
			if err != nil {
				panic(err)
			}
		},
	}
	cmd.PersistentFlags().StringVarP(&serverAddr, "addr", "a", constants.DefaultDlockGrpcAddr, "dlock server address")
	cmd.AddCommand(BuildLockCmd())
	cmd.AddCommand(BuildDlockHealthCmd())
	return cmd
}

func BuildLockCmd() *cobra.Command {
	var key string
	var block bool
	cmd := &cobra.Command{
		Use:   "lock",
		Short: "acquired a distributed lock at the given key and run the command",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			lg := lg.With("key", key, "block", block)

			lockRequest := &v1alpha1.LockRequest{
				Key:     key,
				TryLock: !block,
			}
			if err := lockRequest.Validate(); err != nil {
				return fmt.Errorf("invalid lock request: %w", err)
			}

			lg.Info("acquiring lock...")
			client, err := client.Lock(cmd.Context(), lockRequest)
			if err != nil {
				lg.Error("failed to acquire lock client")
				return err
			}
			lg.Info("acquired lock client")

			ctxca, ca := context.WithCancel(cmd.Context())
			defer ca()

			acquired := make(chan struct{})
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer func() {
					sendErr := client.CloseSend()
					if sendErr != nil {
						lg.Error("failed to close send stream")
					}
					wg.Done()
				}()
				var execCmd *exec.Cmd
				if len(args) > 0 {
					execCmd = exec.CommandContext(cmd.Context(), args[0], args[1:]...)
					execCmd.Stdout = cmd.OutOrStdout()
					execCmd.Stderr = cmd.ErrOrStderr()
					select {
					case <-ctxca.Done():
						lg.Info("cancelling command")
					case <-acquired:
						close(acquired)
						lg.Info(fmt.Sprintf("running command : '%s'", strings.Join(args, " ")))
						if err := execCmd.Run(); err != nil {
							lg.With(logger.Err(err)).Error("command failed")
						}
						lg.Info(fmt.Sprintf("command '%s' finished", strings.Join(args, " ")))
					}
				} else {
					<-cmd.Context().Done()
					lg.Info("no command provided, blocking until lock expires or is cancelled by user")
				}
			}()

			go func() {
				defer ca()
				for {
					lg.Info("waiting to receive lock event")
				RETRY:
					resp, err := client.Recv()
					lg.Info("received lock event")
					if errors.Is(err, io.EOF) {
						lg.Info("stream closed")
						break
					}
					if err != nil {
						errLg := lg.With(logger.Err(err))
						st, ok := status.FromError(err)
						if ok && st.Code() == codes.Canceled {
							errLg.Error("lock expired from remote backend")
							break
						}
						if ok && st.Code() == codes.Unavailable {
							errLg.Error("lock server unavailable, stopping...")
							break
						}
						if errors.Is(err, io.EOF) {
							errLg.Error("stream closed")
							break
						}
						errLg.Error("failed to receive lock event")
						goto RETRY
					}
					if resp.Event == v1alpha1.LockEvent_Acquired {
						lg.Info("lock acquired")
						acquired <- struct{}{}
					} else if resp.Event == v1alpha1.LockEvent_Failed {
						lg.Error("lock acquisition failed")
						break
					}
				}
			}()
			wg.Wait()
			return nil

		},
	}
	cmd.Flags().StringVarP(&key, "dlock.key", "k", "", "key to lock")
	cmd.Flags().BoolVarP(&block, "dlock.block", "b", false, "whether or not to block on lock acquisition")
	return cmd
}

func BuildDlockHealthCmd() *cobra.Command {
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:   "health",
		Short: "Checks the health of the dlock server",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getHealthClient(serverAddr)
			if err != nil {
				lg.With(logger.Err(err)).Error("failed to acquire health client")
				return err
			}
			ctxca, ca := context.WithTimeout(cmd.Context(), timeout)
			defer ca()
			st, err := client.Check(ctxca, &healthv1.HealthCheckRequest{})
			if err != nil {
				lg.With(logger.Err(err)).Error("failed to check health")
				return err
			}
			lg.With("status", st.Status).Info("health check successful")
			return nil
		},
	}
	cmd.Flags().DurationVarP(&timeout, "timeout", "t", 5*time.Second, "timeout for health check")
	return cmd
}

func getHealthClient(addr string) (healthv1.HealthClient, error) {
	cc, err := setupConn(addr)
	if err != nil {
		return nil, err
	}
	return healthv1.NewHealthClient(cc), nil
}

func getDlockClient(addr string) (v1alpha1.DlockClient, error) {
	cc, err := setupConn(addr)
	if err != nil {
		return nil, err
	}
	return v1alpha1.NewDlockClient(cc), nil
}

func setupConn(remoteAddr string) (*grpc.ClientConn, error) {
	remoteUrl, err := url.Parse(remoteAddr)
	if err != nil {
		return nil, err
	}
	conn, err := grpc.Dial(remoteUrl.Host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return conn, err
	}
	return conn, nil
}
