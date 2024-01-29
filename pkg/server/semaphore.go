package server

import (
	"context"

	"github.com/alexandreLamarre/dlock/api/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ v1alpha1.SemaphoreServer = (*LockServer)(nil)

func (ls *LockServer) CreateSemaphore(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "method CreateSemaphore not implemented")
}

func (ls *LockServer) DeleteSemaphore(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "method DeleteSemaphore not implemented")
}

func (ls *LockServer) Acquire(*v1alpha1.SemaphoreRequest, v1alpha1.Semaphore_AcquireServer) error {
	return status.Error(codes.Unimplemented, "method Acquire not implemented")
}
