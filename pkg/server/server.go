package server

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/url"
	"os"
	"time"

	"github.com/alexandreLamarre/dlock/api/v1alpha1"
	configv1alpha1 "github.com/alexandreLamarre/dlock/pkg/config/v1alpha1"
	"github.com/alexandreLamarre/dlock/pkg/lock"
	"github.com/alexandreLamarre/dlock/pkg/lock/broker"
	"github.com/samber/lo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

type LockServer struct {
	lg *slog.Logger
	v1alpha1.UnimplementedDlockServer

	lm lock.LockManager
}

func NewLockServer(ctx context.Context, lg *slog.Logger, configPath string) *LockServer {
	ls := &LockServer{
		lg: lg,
	}
	configData, err := os.ReadFile(configPath)
	if err != nil {
		lg.With("configPath", configPath).Error("failed to read config file")
		return ls
	}
	config := &configv1alpha1.LockServerConfig{}
	if err := json.NewDecoder(bytes.NewReader(configData)).Decode(&config); err != nil {
		lg.With("configPath", configPath).Error("failed to decode config file")
		return ls
	}
	lm := broker.NewLockManager(ctx, lg, config)
	ls.lm = lm
	return ls
}

var _ v1alpha1.DlockServer = &LockServer{}

func (s *LockServer) Lock(in *v1alpha1.LockRequest, stream v1alpha1.Dlock_LockServer) error {
	if s.lm == nil {
		return status.Errorf(codes.Unimplemented, "no lock backend")
	}

	if err := in.Validate(); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	locker := s.lm.NewLock(in.Key)
	var expiredC <-chan struct{}
	if in.TryLock {
		acquired, expired, err := locker.TryLock(stream.Context())
		if err != nil {
			return status.Errorf(codes.Internal, err.Error())
		}
		expiredC = expired
		if !acquired {
			if err := stream.Send(&v1alpha1.LockResponse{
				Event: v1alpha1.LockEvent_Failed,
			}); err != nil {
				return err
			}
			return nil
		}
	} else {
		expired, err := locker.Lock(stream.Context())
		if err != nil {
			return status.Errorf(codes.Internal, err.Error())
		}
		expiredC = expired
	}
	defer locker.Unlock()

	if err := stream.Send(&v1alpha1.LockResponse{
		Event: v1alpha1.LockEvent_Acquired,
	}); err != nil {
		return err
	}
	var streamErr error
	select {
	case <-stream.Context().Done():
		streamErr = stream.Context().Err()
	case <-expiredC:
		streamErr = status.Error(codes.Canceled, "lock expired from storage backend") // fmt.Errorf("lock expired from storage backend")
	}
	if status.FromContextError(streamErr).Code() == codes.Canceled { //nolint
		return nil
	}
	return streamErr
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
