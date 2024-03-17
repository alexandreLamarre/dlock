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
	"github.com/alexandreLamarre/dlock/pkg/logger"
	"github.com/samber/lo"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

type LockServer struct {
	v1alpha1.UnimplementedDlockServer

	lg     *slog.Logger
	tracer trace.Tracer

	lm lock.LockManager
}

func NewLockServer(
	ctx context.Context,
	tracer trace.Tracer,
	metric *sdkmetric.MeterProvider,
	lg *slog.Logger,
	configPath string,
) *LockServer {
	ls := &LockServer{
		lg:     lg,
		tracer: tracer,
	}
	RegisterMeterProvider(metric)
	configData, err := os.ReadFile(configPath)
	if err != nil {
		lg.With("configPath", configPath).Error("failed to read config file")
		return ls
	}
	config := &configv1alpha1.LockServerConfig{}
	if err := json.NewDecoder(bytes.NewReader(configData)).Decode(&config); err != nil {
		lg.With("configPath", configPath, logger.Err(err)).Error("failed to decode config file")
		panic(err)
	}
	broker := broker.NewLockBroker(lg, config, tracer)

	lm, err := broker.LockManager(ctx)
	if err != nil {
		panic("failed to acquire lock manager backend")
	}
	lg.Info("successfully acquired lock manager backend")
	ls.lm = lm
	return ls
}

var _ v1alpha1.DlockServer = &LockServer{}

func (s *LockServer) Lock(in *v1alpha1.LockRequest, stream v1alpha1.Dlock_LockServer) error {
	LockRequestCount.Add(stream.Context(), 1)
	lg := s.lg.With("key", in.Key, "block", !in.TryLock)
	lg.Debug("received lock request")
	if s.lm == nil {
		s.lg.Error("no lock backend")
		return status.Errorf(codes.Unavailable, "no lock backend")
	}

	if err := in.Validate(); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	locker := s.lm.NewLock(in.Key, lock.WithTracer(s.tracer))
	ctx, lockSpan := s.tracer.Start(stream.Context(), "acquire-lock", trace.WithAttributes(
		attribute.KeyValue{
			Key:   "key",
			Value: attribute.StringValue(in.Key),
		},
		attribute.KeyValue{
			Key:   "block",
			Value: attribute.BoolValue(!in.TryLock),
		}),
	)
	var expiredC <-chan struct{}
	if in.TryLock {
		acquired, expired, err := locker.TryLock(ctx)
		if err != nil {
			lg.With(logger.Err(err)).Error("failed to acquire lock")
			lockSpan.RecordError(err)
			lockSpan.End()
			return status.Errorf(codes.Internal, err.Error())
		}
		expiredC = expired
		if !acquired {
			lg.Warn("failed to acquire non-blocking lock")
			if err := stream.Send(&v1alpha1.LockResponse{
				Event: v1alpha1.LockEvent_Failed,
			}); err != nil {
				lockSpan.End()
				return err
			}
			lockSpan.End()
			return nil
		}
	} else {
		expired, err := locker.Lock(ctx)
		if err != nil {
			lg.With(logger.Err(err)).Error("failed to acquire blocking lock", "key", in.Key)
			lockSpan.RecordError(err)
			lockSpan.End()
			return status.Errorf(codes.Internal, err.Error())
		}
		expiredC = expired
	}
	lockSpan.End()
	defer func() {
		lg.Debug("unlocking key")
		locker.Unlock()
	}()
	LockAcquisitionCount.Add(stream.Context(), 1)
	lockHoldStart := time.Now()
	lg.Debug("acquired lock")
	if err := stream.Send(&v1alpha1.LockResponse{
		Event: v1alpha1.LockEvent_Acquired,
	}); err != nil {
		return err
	}
	var streamErr error
	select {
	case <-stream.Context().Done():
		lg.Debug("lock request terminated due to stream context deadline", "key", in.Key)
		streamErr = stream.Context().Err()
		if status.FromContextError(streamErr).Code() == codes.Canceled { //nolint
			lg.Debug("lock cancelled normally")
			streamErr = nil
		}
	case <-expiredC:
		lg.Warn("lock expired from storage backend")
		streamErr = status.Error(codes.Canceled, "lock expired from storage backend") // fmt.Errorf("lock expired from storage backend")
	}
	lockHoldDur := time.Since(lockHoldStart)
	LockHeldTime.Record(stream.Context(), float64(lockHoldDur.Milliseconds()))
	if streamErr != nil {
		lg.With(logger.Err(streamErr)).Error("lock request cancelled")
	}
	return streamErr
}

func (s *LockServer) ListenAndServe(ctx context.Context, addr string) error {
	url, err := url.Parse(addr)
	if err != nil {
		s.lg.With(logger.Err(err)).Error("failed to parse dlock listen server address")
		return err
	}

	listener, err := net.Listen(url.Scheme, url.Host)
	if err != nil {
		s.lg.With(logger.Err(err)).Error("failed to listen on dlock server address")
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
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
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
