package server

import (
	"context"
	"log/slog"
	"net"
	"net/url"
	"time"

	"github.com/alexandreLamarre/dlock/api/v1alpha1"
	"github.com/samber/lo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type LockServer struct {
	lg *slog.Logger
	v1alpha1.UnimplementedDlockServer
}

var _ v1alpha1.DlockServer = &LockServer{}

func (s *LockServer) Lock(req *v1alpha1.LockRequest, server v1alpha1.Dlock_LockServer) error {
	// TODO : implement server logic
	return nil
}

func NewLockServer(lg *slog.Logger) *LockServer {
	return &LockServer{
		lg: lg,
	}
}

func (s *LockServer) ListenAndServe(ctx context.Context, addr string) error {
	url, err := url.Parse(addr)
	if err != nil {
		return err
	}

	listener, err := net.Listen(url.Scheme, url.Host)
	if err != nil {
		return err
	}

	server := grpc.NewServer(
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             15 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    15 * time.Second,
			Timeout: 5 * time.Second,
		}),
	)
	server.RegisterService(&v1alpha1.Dlock_ServiceDesc, s)
	errC := lo.Async(func() error {
		s.lg.With("addr", addr).Info("starting distributed lock server...")
		return server.Serve(listener)
	})

	select {
	case <-ctx.Done():
		server.GracefulStop()
		return ctx.Err()
	case err := <-errC:
		return err
	}
}
