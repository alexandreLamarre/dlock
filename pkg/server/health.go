package server

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

func (l *LockServer) Check(ctx context.Context, req *healthv1.HealthCheckRequest) (*healthv1.HealthCheckResponse, error) {
	if !l.Initialized() {
		return &healthv1.HealthCheckResponse{
			Status: *healthv1.HealthCheckResponse_NOT_SERVING.Enum(),
		}, nil
	}
	ctxca, ca := context.WithTimeout(ctx, 60*time.Second)
	defer ca()
	conditions, err := l.lm.Health(ctxca)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if len(conditions) == 0 {
		return &healthv1.HealthCheckResponse{
			Status: *healthv1.HealthCheckResponse_SERVING.Enum(),
		}, nil

	}
	return &healthv1.HealthCheckResponse{},
		status.Error(codes.Unavailable,
			fmt.Sprintf("health check failed : %s", strings.Join(conditions, ", ")),
		)
}

func (l *LockServer) List(ctx context.Context, _ *healthv1.HealthListRequest) (*healthv1.HealthListResponse, error) {
	ret, err := l.Check(ctx, &healthv1.HealthCheckRequest{})
	if err != nil {
		return nil, err
	}
	return &healthv1.HealthListResponse{
		Statuses: map[string]*healthv1.HealthCheckResponse{
			"lock-manager": ret,
		},
	}, nil
}

func (l *LockServer) Watch(*healthv1.HealthCheckRequest, healthv1.Health_WatchServer) error {
	return status.Error(codes.Unimplemented, "method Watch not implemented")
}
